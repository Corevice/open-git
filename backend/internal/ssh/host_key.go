package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"

	"github.com/open-git/backend/internal/domain/entity"
	domainrepo "github.com/open-git/backend/internal/domain/repository"
)

const AlgorithmEd25519 = "ed25519"

func LoadOrGenerateHostKey(ctx context.Context, repo domainrepo.IHostKeyRepository, algorithm string) (gossh.Signer, error) {
	existing, err := repo.FindByAlgorithm(ctx, algorithm)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		signer, err := gossh.ParsePrivateKey([]byte(existing.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse host key: %w", err)
		}
		return signer, nil
	}

	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	})

	hostKey := &entity.HostKey{
		ID:         uuid.New(),
		Algorithm:  algorithm,
		PrivateKey: string(pemBytes),
		CreatedAt:  time.Now().UTC(),
	}
	if err := repo.Create(ctx, hostKey); err != nil {
		return nil, err
	}

	signer, err := gossh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse generated host key: %w", err)
	}
	return signer, nil
}
