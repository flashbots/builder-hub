package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/flashbots/builder-hub/domain"
)

type BuilderDataAccessor interface {
	GetActiveMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuildersWithServiceCredentials(ctx context.Context) ([]domain.BuilderWithServices, error)
	GetActiveMeasurementsByType(ctx context.Context, attestationType string) ([]domain.Measurement, error)
	GetBuilderByIP(ip net.IP) (*domain.Builder, error)
	GetActiveConfigForBuilder(ctx context.Context, builderName string) (json.RawMessage, error)
	RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType string) error
}

type SecretAccessor interface {
	GetSecretValues(builderName string) (map[string]string, error)
}

type BuilderHub struct {
	dataAccessor   BuilderDataAccessor
	secretAccessor SecretAccessor
}

func NewBuilderHub(dataAccessor BuilderDataAccessor, secretAccessor SecretAccessor) *BuilderHub {
	return &BuilderHub{dataAccessor: dataAccessor, secretAccessor: secretAccessor}
}

func (b *BuilderHub) GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error) {
	return b.dataAccessor.GetActiveMeasurements(ctx)
}

func (b *BuilderHub) GetActiveBuilders(ctx context.Context) ([]domain.BuilderWithServices, error) {
	return b.dataAccessor.GetActiveBuildersWithServiceCredentials(ctx)
}

func (b *BuilderHub) RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType string) error {
	return b.dataAccessor.RegisterCredentialsForBuilder(ctx, builderName, service, tlsCert, ecdsaPubKey, measurementName, attestationType)
}

func (b *BuilderHub) GetConfigWithSecrets(ctx context.Context, builderName string) ([]byte, error) {
	configOpaque, err := b.dataAccessor.GetActiveConfigForBuilder(ctx, builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch config for builder %s %w", builderName, err)
	}
	secrets, err := b.secretAccessor.GetSecretValues(builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch secrets for builder %s %w", builderName, err)
	}
	res, err := MergeConfigSecrets(configOpaque, secrets)
	if err != nil {
		return nil, fmt.Errorf("failing to merge config and secrets %w", err)
	}
	return res, nil
}

func (b *BuilderHub) VerifyIPAndMeasurements(ctx context.Context, ip net.IP, measurement map[string]string, attestationType string) (*domain.Builder, string, error) {
	measurements, err := b.dataAccessor.GetActiveMeasurementsByType(ctx, attestationType)
	if err != nil {
		return nil, "", fmt.Errorf("failing to fetch corresponding measurement data %s %w", attestationType, err)
	}
	measurementName, err := validateMeasurement(measurement, measurements)
	if err != nil {
		return nil, "", fmt.Errorf("failing to validate measurement %w", err)
	}

	builder, err := b.dataAccessor.GetBuilderByIP(ip)
	if err != nil {
		// TODO: might avoid logging ip though it should be ok, at least keep it for development state
		return nil, "", fmt.Errorf("failing to fetch builder by ip %s %w", ip.String(), err)
	}
	return builder, measurementName, nil
}

func validateMeasurement(measurement map[string]string, measurementTemplate []domain.Measurement) (string, error) {
	for _, m := range measurementTemplate {
		if checkMeasurement(measurement, m) {
			return m.Name, nil
		}
	}
	return "", domain.ErrNotFound
}

// validates that all fields from measurementTemplate are the same in measurement
func checkMeasurement(measurement map[string]string, measurementTemplate domain.Measurement) bool {
	for k, v := range measurementTemplate.Measurement {
		if val, ok := measurement[k]; !ok || val != v.Expected {
			return false
		}
	}
	return true
}
