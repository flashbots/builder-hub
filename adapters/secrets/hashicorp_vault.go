package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	vault "github.com/hashicorp/vault/api"
	authkubernetes "github.com/hashicorp/vault/api/auth/kubernetes"
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

	if cfg.AuthMethod != "token" && cfg.AuthMethod != "kubernetes" && cfg.AuthMethod != "" {
		return nil, fmt.Errorf("unsupported AuthMethod %s", cfg.AuthMethod)
	}

	vcfg := vault.DefaultConfig()
	vcfg.Address = cfg.Address
	client, err := vault.NewClient(vcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	svc := &hashicorpVaultService{
		client:     client,
		secretPath: cfg.SecretPrefix,
		mountPath:  cfg.MountPath,
		log:        log,
	}

	if cfg.AuthMethod == "kubernetes" {
		if cfg.Jwt == "" {
			return nil, fmt.Errorf("JWT is required for Kubernetes auth")
		}
		k8sAuth, err := authkubernetes.NewKubernetesAuth(cfg.Role, authkubernetes.WithServiceAccountToken(cfg.Jwt))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Kubernetes auth: %w", err)
		}
		authInfo, err := client.Auth().Login(ctx, k8sAuth)
		if err != nil {
			return nil, fmt.Errorf("Kubernetes auth failed: %w", err)
		}
		watcher, err := client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{Secret: authInfo})
		if err != nil {
			return nil, fmt.Errorf("failed to create token lifetime watcher: %w", err)
		}
		go svc.watchTokenRenewal(ctx, watcher)
	} else {
		if cfg.Token == "" {
			return nil, errors.New("token is required for vault auth")
		}
		client.SetToken(cfg.Token)
	}

	// Verify connection by attempting a read (404 is fine — path may not exist yet)
	verifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_, verifyErr := client.KVv2(cfg.MountPath).Get(verifyCtx, cfg.SecretPrefix)
	if verifyErr != nil && !isVault404(verifyErr) {
		return nil, fmt.Errorf("failed to verify Vault connection: %w", verifyErr)
	}

	return svc, nil
}

func (s *hashicorpVaultService) watchTokenRenewal(ctx context.Context, watcher *vault.LifetimeWatcher) {
	go watcher.Start()
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-watcher.DoneCh():
			if err != nil {
				s.log.Error("vault token renewal stopped", "err", err)
			}
			return
		case <-watcher.RenewCh():
			s.log.Debug("vault token renewed")
		}
	}
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

	secret, err := s.client.KVv2(s.mountPath).Get(ctx, path)
	if err != nil {
		if isVault404(err) {
			return json.RawMessage("{}"), nil
		}
		return nil, fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return json.RawMessage("{}"), nil
	}

	secretJSON, err := json.Marshal(secret.Data)
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

	_, err := s.client.KVv2(s.mountPath).Put(ctx, path, dataMap)
	if err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	return nil
}
