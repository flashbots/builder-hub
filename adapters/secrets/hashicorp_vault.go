package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

type hashicorpVaultService struct {
	client     *vault.Client
	secretPath string
	mountPath  string
}

type VaultConfig struct {
	Address      string // Vault server address (e.g., http://localhost:8200)
	Token        string // Vault token for authentication
	SecretPrefix string // Path prefix for secrets (e.g., "secrets/builder-hub")
	MountPath    string // Vault KV v2 mount path (e.g., "secret", defaults to "secret")
}

func NewHashicorpVaultService(cfg VaultConfig) (*hashicorpVaultService, error) {
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}

	client, err := vault.New(
		vault.WithAddress(cfg.Address),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	if err := client.SetToken(cfg.Token); err != nil {
		return nil, fmt.Errorf("failed to set Vault token: %w", err)
	}

	// Verify connection by attempting a read (404 is fine — path may not exist yet)
	ctx := context.Background()
	_, verifyErr := client.Secrets.KvV2Read(ctx, cfg.SecretPrefix, vault.WithMountPath(cfg.MountPath))
	if verifyErr != nil && !isVault404(verifyErr) {
		return nil, fmt.Errorf("failed to verify Vault connection: %w", verifyErr)
	}

	return &hashicorpVaultService{
		client:     client,
		secretPath: cfg.SecretPrefix,
		mountPath:  cfg.MountPath,
	}, nil
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
