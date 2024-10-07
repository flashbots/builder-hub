package application

import (
	"context"
	"fmt"
	"net"

	"github.com/flashbots/builder-hub/domain"
)

type SecretAccessor interface {
	GetSecretValues(secretName string) (map[string]string, error)
}

type BuilderDataAccessor interface {
	GetActiveMeasurements(ctx context.Context) ([]domain.Measurement, error)
	GetActiveBuildersWithServiceCredentials(ctx context.Context) ([]domain.BuilderWithServices, error)
	GetMeasurementByTypeAndHash(attestationType string, hash []byte) (*domain.Measurement, error)
	GetBuilderByIP(ip net.IP) (*domain.Builder, error)
}

type BuilderHub struct {
	dataAccessor BuilderDataAccessor
}

func NewBuilderHub(dataAccessor BuilderDataAccessor) *BuilderHub {
	return &BuilderHub{dataAccessor: dataAccessor}
}

func (b *BuilderHub) GetAllowedMeasurements(ctx context.Context) ([]domain.Measurement, error) {
	return b.dataAccessor.GetActiveMeasurements(ctx)
}

func (b *BuilderHub) GetActiveBuilders(ctx context.Context) ([]domain.BuilderWithServices, error) {
	return b.dataAccessor.GetActiveBuildersWithServiceCredentials(ctx)
}

//	func (b *BuilderHub) GetConfig(ctx context.Context, builderName string) []string {
//		// get config according to builder
//		// get secrets according to builder
//		return nil
//	}
func (b *BuilderHub) VerifyIpAndMeasurements(ctx context.Context, ip net.IP, measurement *domain.Measurement) (*domain.Builder, error) {
	_, err := b.dataAccessor.GetMeasurementByTypeAndHash(measurement.AttestationType, measurement.Hash)
	if err != nil {
		return nil, fmt.Errorf("failing to fetch corresponding measurement data %x %w", measurement.Hash, err)
	}
	builder, err := b.dataAccessor.GetBuilderByIP(ip)
	if err != nil {
		// TODO: might avoid logging ip though it should be ok, at least keep it for development state
		return nil, fmt.Errorf("failing to fetch builder by ip %s %w", ip.String(), err)
	}
	return builder, nil
}
