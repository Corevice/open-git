[![Chart Release](https://github.com/Corevice/open-git/actions/workflows/chart-release.yml/badge.svg)](https://github.com/Corevice/open-git/actions/workflows/chart-release.yml)

# Open Git Helm Chart

## Overview

This Helm chart deploys **Open Git** (GitForge), a self-hosted Git hosting platform, on Kubernetes. The chart provisions the web/API server (Next.js frontend + Go backend), Git SSH access, CI/CD worker, and optional dependencies (PostgreSQL, Redis, MinIO) with persistent repository storage, ingress, autoscaling, and database migration hooks.

Published charts are available via OCI:

```bash
helm pull oci://ghcr.io/Corevice/open-git/charts/open-git --version v1.0.0
```

## Prerequisites

- Kubernetes **>= 1.24**
- Helm **>= 3.12**
- (Optional) [cert-manager](https://cert-manager.io/) for automatic TLS certificate provisioning
- (Optional) [Prometheus Operator](https://prometheus-operator.dev/) when enabling `serviceMonitor`

## Quick Start

Update chart dependencies, then install with a release name and host:

```bash
helm dependency update helm/
helm install gitforge ./helm \
  --set ingress.enabled=true \
  --set ingress.host=git.example.com
```

To install from the OCI registry after a tagged release:

```bash
helm install gitforge oci://ghcr.io/Corevice/open-git/charts/open-git \
  --version v1.0.0 \
  --set ingress.enabled=true \
  --set ingress.host=git.example.com
```

## Configuration

| Key | Default | Description |
|-----|---------|-------------|
| `global.databaseType` | `internal` | Database mode: `internal` (Bitnami PostgreSQL subchart) or `external` (managed/external PostgreSQL). |
| `replicaCount` | `1` | Number of web/API pod replicas. |
| `ingress.host` | `""` | Hostname for the HTTP(S) ingress. Required when `ingress.enabled` is `true`. |
| `ssh.serviceType` | `LoadBalancer` | Kubernetes Service type for Git SSH access (`LoadBalancer` or `NodePort`). |
| `autoscaling.enabled` | `false` | Enable HorizontalPodAutoscaler for web/API pods. |
| `persistence.repositories.size` | `50Gi` | Size of the PersistentVolumeClaim for Git repository storage. |
| `secrets.existingSecret` | `""` | Name of an existing Secret containing JWT, OAuth, and webhook signing keys. When empty, the chart generates a Secret automatically. |

See `values.yaml` for the full list of configurable values.

## Upgrade Procedure

1. **Back up the database** before upgrading. For internal PostgreSQL, take a logical dump (`pg_dump`) or a volume snapshot. For external databases, follow your provider's backup procedure.
2. Run `helm upgrade` with the new chart version and any updated values:

   ```bash
   helm upgrade gitforge ./helm --version <new-version> -f my-values.yaml
   ```

3. **Watch the migration Job** created by the pre-upgrade Helm hook. The upgrade aborts if the migration fails:

   ```bash
   kubectl get jobs -l app.kubernetes.io/instance=gitforge
   kubectl logs job/<migration-job-name>
   ```

4. **Verify pods** are Ready and the application responds:

   ```bash
   kubectl get pods -l app.kubernetes.io/instance=gitforge
   kubectl rollout status deployment/gitforge
   ```

## Rollback Procedure

To revert to a previous release revision:

```bash
helm rollback gitforge <revision>
```

List available revisions with `helm history gitforge`.

**Note:** PersistentVolumeClaims for Git repository storage are **not** deleted on rollback. Data on PVCs is preserved across rollbacks and upgrades.

## Secret Management

Prefer referencing pre-created Secrets rather than storing sensitive values in plain-text `values.yaml`.

### existingSecret pattern

Create a Secret with the required keys before installing or upgrading:

```bash
kubectl create secret generic gitforge-app-secrets \
  --namespace default \
  --from-literal=jwt-secret='your-jwt-signing-key' \
  --from-literal=oauth-client-secret='your-oauth-secret' \
  --from-literal=webhook-secret='your-webhook-signing-key'
```

Then reference it in your values:

```yaml
secrets:
  existingSecret: gitforge-app-secrets
```

For external PostgreSQL, provide the database password via a separate Secret:

```yaml
global:
  databaseType: external
externalDatabase:
  host: postgres.example.com
  existingSecret: gitforge-db-credentials
```

### External Secrets Operator

Sync secrets from a cloud provider (e.g. AWS Secrets Manager) into Kubernetes:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: gitforge-app-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: ClusterSecretStore
  target:
    name: gitforge-app-secrets
    creationPolicy: Owner
  data:
    - secretKey: jwt-secret
      remoteRef:
        key: prod/gitforge/jwt-secret
    - secretKey: oauth-client-secret
      remoteRef:
        key: prod/gitforge/oauth-client-secret
    - secretKey: webhook-secret
      remoteRef:
        key: prod/gitforge/webhook-secret
```

Set `secrets.existingSecret: gitforge-app-secrets` in your Helm values to use the synced Secret.

## Known Limitations

- **Internal PostgreSQL is for development only.** The bundled Bitnami PostgreSQL subchart is convenient for local and dev clusters but is **not recommended for production**. Use `global.databaseType: external` with a managed or dedicated PostgreSQL instance for production workloads.
- **PVCs are retained after `helm uninstall`.** Git repository data stored on PersistentVolumeClaims uses a retain policy by default and is **not** automatically deleted when the release is uninstalled. Manually delete PVCs if you intend to permanently remove repository data.
- **Run `helm dependency update` before first install.** Subchart dependencies (PostgreSQL, Redis, MinIO) must be downloaded locally before packaging or installing from source:

  ```bash
  helm dependency update helm/
  ```
