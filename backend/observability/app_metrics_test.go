package observability_test

import (
	"testing"

	"github.com/open-git/backend/observability"
	"github.com/prometheus/client_golang/prometheus"
)

func gatherMetric(t *testing.T, name string) float64 {
	t.Helper()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if c := m.GetCounter(); c != nil {
				return c.GetValue()
			}
			if g := m.GetGauge(); g != nil {
				return g.GetValue()
			}
			if h := m.GetHistogram(); h != nil {
				return float64(h.GetSampleCount())
			}
		}
	}

	t.Fatalf("metric %q not found", name)
	return 0
}

func metricExists(t *testing.T, name string) bool {
	t.Helper()

	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, mf := range metricFamilies {
		if mf.GetName() == name {
			return true
		}
	}
	return false
}

func TestRecordGitOperation(t *testing.T) {
	observability.RecordGitOperation("upload_pack", "success")
	observability.RecordGitOperation("upload_pack", "success")

	value := gatherMetric(t, "githost_git_operations_total")
	if value < 2.0 {
		t.Fatalf("githost_git_operations_total = %v, want >= 2.0", value)
	}
}

func TestAddGitTransferBytes(t *testing.T) {
	observability.AddGitTransferBytes("upload", 1024)

	value := gatherMetric(t, "githost_git_transfer_bytes_total")
	if value < 1024 {
		t.Fatalf("githost_git_transfer_bytes_total = %v, want >= 1024", value)
	}
}

func TestRecordWebhookDelivery(t *testing.T) {
	observability.RecordWebhookDelivery("success", 0.05)

	if !metricExists(t, "githost_webhook_deliveries_total") {
		t.Fatal("githost_webhook_deliveries_total not found in gathered output")
	}
	if !metricExists(t, "githost_webhook_delivery_duration_seconds") {
		t.Fatal("githost_webhook_delivery_duration_seconds not found in gathered output")
	}
}

func TestSetDBConnections(t *testing.T) {
	observability.SetDBConnections(7)

	value := gatherMetric(t, "githost_db_connections")
	if value != 7.0 {
		t.Fatalf("githost_db_connections = %v, want 7.0", value)
	}
}

func TestRecordCIJob(t *testing.T) {
	observability.IncCIJobsRunning()
	observability.RecordCIJob("success")

	value := gatherMetric(t, "githost_ci_jobs_total")
	if value < 1.0 {
		t.Fatalf("githost_ci_jobs_total = %v, want >= 1.0", value)
	}
}
