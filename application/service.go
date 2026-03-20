package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/flashbots/builder-hub/domain"
)

type BuilderDataAccessor interface {
	GetActiveMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuildersWithServiceCredentials(ctx context.Context, network string) ([]domain.BuilderWithServices, error)
	GetActiveMeasurementsByType(ctx context.Context, attestationType string) ([]domain.Measurement, error)
	GetBuilderByIP(ip net.IP) (*domain.Builder, error)
	GetActiveConfigForBuilder(ctx context.Context, builderName string) (json.RawMessage, error)
	RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType string) error
	LogEvent(ctx context.Context, eventName, builderName, name string) error
}

var ErrMissingSecret = errors.New("missing secret for builder")

type SecretAccessor interface {
	GetSecretValues(ctx context.Context, builderName string) (json.RawMessage, error)
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

func (b *BuilderHub) GetActiveBuilders(ctx context.Context, network string) ([]domain.BuilderWithServices, error) {
	return b.dataAccessor.GetActiveBuildersWithServiceCredentials(ctx, network)
}

func (b *BuilderHub) LogEvent(ctx context.Context, eventName, builderName, name string) error {
	return b.dataAccessor.LogEvent(ctx, eventName, builderName, name)
}

func (b *BuilderHub) RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName, attestationType string) error {
	return b.dataAccessor.RegisterCredentialsForBuilder(ctx, builderName, service, tlsCert, ecdsaPubKey, measurementName, attestationType)
}

func (b *BuilderHub) GetConfigWithSecrets(ctx context.Context, builderName string) ([]byte, error) {
	_, err := b.dataAccessor.GetActiveConfigForBuilder(ctx, builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch config for builder %s %w", builderName, err)
	}
	secr, err := b.secretAccessor.GetSecretValues(ctx, builderName)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch secrets for builder %s %w", builderName, err)
	}
	return secr, nil
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

// validates that all fields from measurementTemplate are the same in measurement.
// For each field, the measurement value must match at least one of the expected values (OR semantics).
func checkMeasurement(measurement map[string]string, measurementTemplate domain.Measurement) bool {
	for k, v := range measurementTemplate.Measurement {
		val, ok := measurement[k]
		if !ok {
			return false
		}
		if !matchesAnyExpected(val, v.GetExpectedValues()) {
			return false
		}
	}
	return true
}

// matchesAnyExpected returns true if the value matches any of the expected values.
func matchesAnyExpected(value string, expected []string) bool {
	for _, exp := range expected {
		if value == exp {
			return true
		}
	}
	return false
}
