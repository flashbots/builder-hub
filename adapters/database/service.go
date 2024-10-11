package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/flashbots/builder-hub/domain"
	"github.com/jackc/pgtype"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Service struct {
	DB *sqlx.DB
}

func NewDatabaseService(dsn string) (*Service, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.DB.SetMaxOpenConns(50)
	db.DB.SetMaxIdleConns(10)
	db.DB.SetConnMaxIdleTime(0)

	dbService := &Service{DB: db} //nolint:exhaustruct
	return dbService, err
}

func (s *Service) Close() error {
	return s.DB.Close()
}

// GetMeasurementByTypeAndHash retrieves a measurement by OID and hash
func (s *Service) GetActiveMeasurementsByType(ctx context.Context, attestationType string) ([]domain.Measurement, error) {
	var measurements []Measurement
	err := s.DB.SelectContext(ctx, &measurements, `SELECT * FROM measurements_whitelist WHERE is_active=true AND attestation_type=$1`, attestationType)
	var domainMeasurements []domain.Measurement
	for _, m := range measurements {
		domainM, err := convertMeasurementToDomain(m)
		if err != nil {
			return nil, err
		}
		domainMeasurements = append(domainMeasurements, *domainM)
	}
	return domainMeasurements, err
}

// GetBuilderByIP retrieves a builder by IP address
func (s *Service) GetBuilderByIP(ip net.IP) (*domain.Builder, error) {
	var paramIP pgtype.Inet
	err := paramIP.Set(ip)
	if err != nil {
		return nil, err
	}

	var b Builder
	err = s.DB.Get(&b, `
		SELECT * FROM builders
		WHERE ip_address = $1 and is_active = true
	`, paramIP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return convertBuilderToDomain(b)
}

// GetActiveMeasurements retrieves all measurements
func (s *Service) GetActiveMeasurements(ctx context.Context) ([]domain.Measurement, error) {
	var measurements []Measurement
	err := s.DB.SelectContext(ctx, &measurements, `SELECT * FROM measurements_whitelist WHERE is_active=true`)
	var domainMeasurements []domain.Measurement
	for _, m := range measurements {
		domainM, err := convertMeasurementToDomain(m)
		if err != nil {
			return nil, err
		}
		domainMeasurements = append(domainMeasurements, *domainM)
	}
	return domainMeasurements, err
}

// RegisterCredentialsForBuilder registers new credentials for a builder, deprecating all previous credentials
// It uses hash and attestation_type to fetch the corresponding measurement_id via a subquery.
func (s *Service) RegisterCredentialsForBuilder(ctx context.Context, builderName, service, tlsCert string, ecdsaPubKey []byte, measurementName string, attestationType string) error {
	// Start a transaction
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Deprecate all previous credentials for this builder and service
	_, err = tx.Exec(`
        UPDATE service_credential_registrations
        SET is_active = false, deprecated_at = NOW()
        WHERE builder_name = $1 AND service = $2
    `, builderName, service)
	if err != nil {
		return err
	}

	// Insert new credentials with a subquery to fetch the measurement_id
	var nullableTLSCert sql.NullString
	if tlsCert != "" {
		nullableTLSCert = sql.NullString{String: tlsCert, Valid: true}
	}

	_, err = tx.Exec(`
        INSERT INTO service_credential_registrations 
        (builder_name, service, tls_cert, ecdsa_pubkey, is_active, measurement_id)
        VALUES ($1, $2, $3, $4, true, 
            (SELECT id FROM measurements_whitelist WHERE name = $5 AND attestation_type = $6)
        )
    `, builderName, service, nullableTLSCert, ecdsaPubKey, measurementName, attestationType)
	if err != nil {
		return fmt.Errorf("failed to insert credentials for builder %s: %w", builderName, err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// GetActiveConfigForBuilder retrieves the active config for a builder by name
func (s *Service) GetActiveConfigForBuilder(ctx context.Context, builderName string) (json.RawMessage, error) {
	var config BuilderConfig
	err := s.DB.GetContext(ctx, &config, `
		SELECT * FROM builder_configs
		WHERE builder_name = $1 AND is_active = true
	`, builderName)
	return config.Config, err
}

func (s *Service) GetActiveBuildersWithServiceCredentials(ctx context.Context) ([]domain.BuilderWithServices, error) {
	rows, err := s.DB.QueryContext(ctx, `
        SELECT 
            b.name,
            b.ip_address,
            scr.service,
            scr.tls_cert,
            scr.ecdsa_pubkey
        FROM 
            builders b
        LEFT JOIN 
            service_credential_registrations scr ON b.name = scr.builder_name
        WHERE 
            b.is_active = true AND (scr.is_active = true OR scr.is_active IS NULL)
        ORDER BY 
            b.name, scr.service
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	buildersMap := make(map[string]*BuilderWithCredentials)

	for rows.Next() {
		var ipAddress pgtype.Inet
		var builderName string
		var service sql.NullString
		var tlsCert sql.NullString
		var ecdsaPubKey []byte

		err := rows.Scan(&builderName, &ipAddress, &service, &tlsCert, &ecdsaPubKey)
		if err != nil {
			return nil, err
		}

		builder, exists := buildersMap[builderName]
		if !exists {
			builder = &BuilderWithCredentials{
				Name:      builderName,
				IPAddress: ipAddress,
			}
			buildersMap[builderName] = builder
		}

		if service.Valid {
			builder.Credentials = append(builder.Credentials, ServiceCredential{
				Service:     service.String,
				TLSCert:     tlsCert,
				ECDSAPubKey: ecdsaPubKey,
			})
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Convert map to slice
	builders := make([]domain.BuilderWithServices, 0, len(buildersMap))
	for _, builder := range buildersMap {
		dBuilder, err := toDomainBuilderWithCredentials(*builder)
		if err != nil {
			return nil, err
		}
		builders = append(builders, *dBuilder)
	}

	return builders, nil
}

// LogEvent creates a new log entry in the event_log table.
// It uses hash and attestation_type to fetch the corresponding measurement_id via a subquery.
func (s *Service) LogEvent(ctx context.Context, eventName, builderName, hash, attestationType string) error {
	// Insert new event log entry with a subquery to fetch the measurement_id
	_, err := s.DB.ExecContext(ctx, `
        INSERT INTO event_log 
        (event_name, builder_name, measurement_id)
        VALUES ($1, $2, 
            (SELECT id FROM measurements_whitelist WHERE hash = $3 AND attestation_type = $4)
        )
    `, eventName, builderName, hash, attestationType)
	if err != nil {
		return fmt.Errorf("failed to insert event log for builder %s: %w", builderName, err)
	}

	return nil
}