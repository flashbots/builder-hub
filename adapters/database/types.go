package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/flashbots/builder-hub/domain"
	"github.com/jackc/pgtype"
)

type ActiveBuilder struct {
	BuilderID   int         `db:"builder_id"`
	Name        string      `db:"name"`
	IPAddress   pgtype.Inet `db:"ip_address"`
	TLSPubKey   []byte      `db:"tls_pubkey"`
	ECDSAPubKey []byte      `db:"ecdsa_pubkey"`
}

type Measurement struct {
	ID              int             `db:"id"`
	Name            string          `db:"name"`
	AttestationType string          `db:"attestation_type"`
	Measurement     json.RawMessage `db:"measurement"`
	IsActive        bool            `db:"is_active"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
	DeprecatedAt    *time.Time      `db:"deprecated_at"`
}

func convertMeasurementToDomain(measurement Measurement) (*domain.Measurement, error) {
	var m domain.Measurement
	m.AttestationType = measurement.AttestationType
	m.Measurement = make(map[string]domain.SingleMeasurement)
	err := json.Unmarshal(measurement.Measurement, &m.Measurement)
	if err != nil {
		return nil, err
	}
	m.Name = measurement.Name
	return &m, nil
}

type Builder struct {
	Name         string      `db:"name"`
	IPAddress    pgtype.Inet `db:"ip_address"`
	IsActive     bool        `db:"is_active"`
	CreatedAt    time.Time   `db:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at"`
	DeprecatedAt *time.Time  `db:"deprecated_at"`
}

func convertBuilderToDomain(builder Builder) (*domain.Builder, error) {
	if builder.IPAddress.IPNet == nil {
		return nil, domain.ErrIncorrectBuilder
	}
	return &domain.Builder{
		Name:      builder.Name,
		IPAddress: builder.IPAddress.IPNet.IP,
		IsActive:  builder.IsActive,
	}, nil
}

type ServiceCredentialRegistration struct {
	ID          int       `db:"id"`
	BuilderName string    `db:"builder_name"`
	Service     string    `db:"service"`
	TLSCert     string    `db:"tls_cert"`
	ECDSAPubKey []byte    `db:"ecdsa_pubkey"`
	CreatedAt   time.Time `db:"created_at"`
}

type BuilderConfig struct {
	ID          int             `db:"id"`
	BuilderName string          `db:"builder_name"`
	Config      json.RawMessage `db:"config"`
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

type BuilderWithCredentials struct {
	Name        string
	IPAddress   pgtype.Inet
	Credentials []ServiceCredential
}
type ServiceCredential struct {
	Service     string
	TLSCert     sql.NullString
	ECDSAPubKey []byte
}

func toDomainBuilderWithCredentials(builder BuilderWithCredentials) (*domain.BuilderWithServices, error) {
	if builder.IPAddress.IPNet == nil {
		return nil, domain.ErrIncorrectBuilder
	}
	s := domain.BuilderWithServices{
		Builder: domain.Builder{
			Name:      builder.Name,
			IPAddress: builder.IPAddress.IPNet.IP,
			IsActive:  true,
		},
		Services: make([]domain.BuilderServices, 0, len(builder.Credentials)),
	}
	for _, cred := range builder.Credentials {
		s.Services = append(s.Services, domain.BuilderServices{
			TLSCert:     cred.TLSCert.String,
			ECDSAPubKey: cred.ECDSAPubKey,
			Service:     cred.Service,
		})
	}
	return &s, nil
}
