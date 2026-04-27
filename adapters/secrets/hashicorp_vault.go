package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

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
	Address       string // Vault server address (e.g., http://localhost:8200)
	Token         string // Vault token for authentication (used when AuthMethod=="token")
	SecretPrefix  string // Path prefix for secrets (e.g., "secrets/builder-hub")
	MountPath     string // Vault KV v2 mount path (e.g., "secret", defaults to "secret")
	AuthMethod    string // "token" (default), "kubernetes", or "jwt"
	AuthMountPath string // Vault auth mount path for Kubernetes/JWT auth (e.g., "k8s/eth-l1-prod"); defaults to "kubernetes" for k8s auth, "jwt" for JWT auth
	Role          string // Role name for Kubernetes/JWT auth (required if AuthMethod is "kubernetes" or "jwt")
	Jwt           string // ServiceAccount JWT for Kubernetes/JWT auth (required if AuthMethod is "kubernetes" or "jwt")
}

func NewHashicorpVaultService(ctx context.Context, log *slog.Logger, cfg VaultConfig) (*hashicorpVaultService, error) {
	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}

	if cfg.AuthMethod != "token" && cfg.AuthMethod != "kubernetes" && cfg.AuthMethod != "jwt" && cfg.AuthMethod != "" {
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

	switch cfg.AuthMethod {
	case "kubernetes":
		if cfg.Jwt == "" {
			return nil, fmt.Errorf("JWT is required for Kubernetes auth")
		}
		k8sAuthOpts := []authkubernetes.LoginOption{authkubernetes.WithServiceAccountToken(cfg.Jwt)}
		if cfg.AuthMountPath != "" {
			k8sAuthOpts = append(k8sAuthOpts, authkubernetes.WithMountPath(cfg.AuthMountPath))
		}
		k8sAuth, err := authkubernetes.NewKubernetesAuth(cfg.Role, k8sAuthOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Kubernetes auth: %w", err)
		}
		authInfo, err := client.Auth().Login(ctx, k8sAuth)
		if err != nil {
			return nil, fmt.Errorf("kubernetes auth failed: %w", err)
		}
		watcher, err := client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{Secret: authInfo})
		if err != nil {
			return nil, fmt.Errorf("failed to create token lifetime watcher: %w", err)
		}
		go svc.watchTokenRenewal(ctx, watcher)

	case "jwt":
		if cfg.Jwt == "" {
			return nil, fmt.Errorf("JWT is required for JWT auth")
		}
		if cfg.Role == "" {
			return nil, fmt.Errorf("role is required for JWT auth")
		}
		mount := cfg.AuthMountPath
		if mount == "" {
			mount = "jwt"
		}
		authInfo, err := client.Logical().WriteWithContext(ctx, fmt.Sprintf("auth/%s/login", mount), map[string]interface{}{
			"role": cfg.Role,
			"jwt":  cfg.Jwt,
		})
		if err != nil {
			return nil, fmt.Errorf("JWT auth failed: %w", err)
		}
		if authInfo == nil || authInfo.Auth == nil {
			return nil, fmt.Errorf("JWT auth returned no authentication info")
		}
		client.SetToken(authInfo.Auth.ClientToken)
		watcher, err := client.NewLifetimeWatcher(&vault.LifetimeWatcherInput{Secret: authInfo})
		if err != nil {
			return nil, fmt.Errorf("failed to create token lifetime watcher: %w", err)
		}
		go svc.watchTokenRenewal(ctx, watcher)

	default:
		if cfg.Token == "" {
			return nil, errors.New("token is required for vault auth")
		}
		client.SetToken(cfg.Token)
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
