package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type hashicorpVaultService struct {
	client     *vault.Client
	secretPath string
	mountPath  string
	log        *slog.Logger
}

type VaultConfig struct {
	Address      string // Vault server address (e.g., http://localhost:8200)
	Token        string // Vault token for authentication (used when AuthMethod=="token")
	SecretPrefix string // Path prefix for secrets (e.g., "secrets/builder-hub")
	MountPath    string // Vault KV v2 mount path (e.g., "secret", defaults to "secret")
	AuthMethod   string // "token" (default) or "kubernetes"
	Role         string // Role name for Kubernetes auth (required if AuthMethod=="kubernetes")
	Jwt          string // ServiceAccount JWT for Kubernetes auth (required if AuthMethod=="kubernetes")
}

func NewHashicorpVaultService(ctx context.Context, log *slog.Logger, cfg VaultConfig) (*hashicorpVaultService, error) {
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}

	client, err := vault.New(vault.WithAddress(cfg.Address))
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	if cfg.AuthMethod != "token" && cfg.AuthMethod != "kubernetes" && cfg.AuthMethod != "" {
		return nil, fmt.Errorf("unsupported AuthMethod %s", cfg.AuthMethod)
	}

	svc := &hashicorpVaultService{
		client:     client,
		secretPath: cfg.SecretPrefix,
		mountPath:  cfg.MountPath,
		log:        log,
	}

	if cfg.AuthMethod == "kubernetes" {
		if cfg.Jwt == "" {
			return nil, fmt.Errorf("Jwt is required for Kubernetes auth")
		}
		authResp, err := client.Auth.KubernetesLogin(ctx, schema.KubernetesLoginRequest{Role: cfg.Role, Jwt: cfg.Jwt})
		if err != nil {
			return nil, fmt.Errorf("Kubernetes auth failed: %w", err)
		}
		if err := client.SetToken(authResp.Auth.ClientToken); err != nil {
			return nil, fmt.Errorf("failed to set Vault token after Kubernetes auth: %w", err)
		}
		svc.startTokenRenewal(ctx, authResp.Auth.LeaseDuration, authResp.Auth.Renewable)
	} else {
		if err := client.SetToken(cfg.Token); err != nil {
			return nil, fmt.Errorf("failed to set Vault token: %w", err)
		}
	}

	// Verify connection by attempting a read (404 is fine — path may not exist yet)
	_, verifyErr := client.Secrets.KvV2Read(context.Background(), cfg.SecretPrefix, vault.WithMountPath(cfg.MountPath))
	if verifyErr != nil && !isVault404(verifyErr) {
		return nil, fmt.Errorf("failed to verify Vault connection: %w", verifyErr)
	}

	return svc, nil
}

// startTokenRenewal runs a background goroutine that periodically renews the Vault token.
// It exits when ctx is cancelled or if the token is non-renewable.
func (s *hashicorpVaultService) startTokenRenewal(ctx context.Context, leaseDurationSeconds int, renewable bool) {
	if leaseDurationSeconds <= 0 || !renewable {
		return
	}

	// Renew at 2/3 of the initial lease duration.
	waitDuration := time.Duration(leaseDurationSeconds) * time.Second * 2 / 3

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDuration):
				if _, err := s.client.Auth.TokenRenewSelf(ctx, schema.TokenRenewSelfRequest{}); err != nil {
					s.log.Error("vault token renewal failed", "err", err)
				}
			}
		}
	}()
}

func isVault404(err error) bool {
	var responseErr *vault.ResponseError
	return errors.As(err, &responseErr) && responseErr.StatusCode == 404
}

func (s *hashicorpVaultService) secretKVPath(builderName string) string {
	if s.secretPath == "" {
		return builderName
	}
	return fmt.Sprintf("%s/%s", s.secretPath, builderName)
}

// GetSecretValues retrieves secrets for a specific builder from Vault KV v2.
// Implements application.SecretAccessor interface.
func (s *hashicorpVaultService) GetSecretValues(ctx context.Context, builderName string) (json.RawMessage, error) {
	path := s.secretKVPath(builderName)

	response, err := s.client.Secrets.KvV2Read(ctx, path, vault.WithMountPath(s.mountPath))
	if err != nil {
		if isVault404(err) {
			return json.RawMessage("{}"), nil
		}
		return nil, fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	if response.Data.Data == nil {
		return json.RawMessage("{}"), nil
	}

	secretJSON, err := json.Marshal(response.Data.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Vault secret: %w", err)
	}

	return json.RawMessage(secretJSON), nil
}

// SetSecretValues stores secrets for a specific builder in Vault KV v2.
// Implements ports.AdminSecretService interface.
func (s *hashicorpVaultService) SetSecretValues(ctx context.Context, builderName string, values json.RawMessage) error {
	path := s.secretKVPath(builderName)

	var dataMap map[string]any
	if err := json.Unmarshal(values, &dataMap); err != nil {
		return fmt.Errorf("failed to unmarshal secret values: %w", err)
	}

	_, err := s.client.Secrets.KvV2Write(ctx, path, schema.KvV2WriteRequest{Data: dataMap}, vault.WithMountPath(s.mountPath))
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	return nil
}
