package database

import (
	"context"
	"database/sql"
	"errors"
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

//
//// GetAllowedMeasurements retrieves all active measurements
//func (s *Service) GetAllowedMeasurements() ([]Measurement, error) {
//	var measurements []Measurement
//	err := s.DB.Select(&measurements, `
//        SELECT id, measurement
//        FROM measurements_whitelist
//        WHERE is_active = true
//    `)
//	return measurements, err
//}
//
//// GetActiveBuilders retrieves all active builders with their IPs and public keys
//func (s *Service) GetActiveBuilders() ([]ActiveBuilder, error) {
//	var builders []ActiveBuilder
//	err := s.DB.Select(&builders, `
//        SELECT b.id AS builder_id, b.name, iw.ip_address, scr.tls_pubkey, scr.ecdsa_pubkey
//        FROM builders b
//        JOIN ip_whitelist iw ON b.id = iw.builder_id
//        JOIN service_credential_registrations scr ON iw.id = scr.ip_whitelist_id
//        WHERE iw.is_active = true AND scr.service = 'builder'
//    `)
//	return builders, err
//}
//
//// GetConfigByIP retrieves the config for a builder based on their IP address
//func (s *Service) GetConfigByIP(ip string) (BuilderConfig, error) {
//	var config BuilderConfig
//	err := s.DB.Get(&config, `
//        SELECT bc.builder_id, bc.config
//        FROM builder_configs bc
//        JOIN ip_whitelist iw ON bc.builder_id = iw.builder_id
//        WHERE iw.ip_address = $1 AND iw.is_active = true AND bc.is_active = true
//    `, ip)
//	return config, err
//}
//
//// RegisterNewCredentials registers new credentials for a service
//func (s *Service) RegisterNewCredentials(ip, service string, tlsPubKey, ecdsaPubKey []byte) error {
//	_, err := s.DB.Exec(`
//        INSERT INTO service_credential_registrations (ip_whitelist_id, service, tls_pubkey, ecdsa_pubkey)
//        SELECT id, $2, $3, $4
//        FROM ip_whitelist
//        WHERE ip_address = $1 AND is_active = true
//    `, ip, service, tlsPubKey, ecdsaPubKey)
//	return err
//}
//
//// GetBuilderByIP retrieves an active builder by their IP address
//func (s *Service) GetActiveBuilderByIP(ip net.IP) (*ActiveBuilder, error) {
//	var paramIP pgtype.Inet
//	err := paramIP.Set(ip)
//	if err != nil {
//		return nil, err
//	}
//
//	var builder ActiveBuilder
//	err = s.DB.Get(&builder, `
//        SELECT b.id AS builder_id, b.name, iw.ip_address, scr.tls_pubkey, scr.ecdsa_pubkey
//        FROM builders b
//        JOIN ip_whitelist iw ON b.id = iw.builder_id
//        JOIN service_credential_registrations scr ON iw.id = scr.ip_whitelist_id
//        WHERE iw.ip_address = $1 AND iw.is_active = true AND scr.service = 'builder'
//    `, paramIP)
//
//	if err != nil {
//		return nil, fmt.Errorf("error getting builder by IP: %w", err)
//	}
//
//	return &builder, nil
//}
//
//type IPWhitelistEntry struct {
//	BuilderID int         `db:"builder_id"`
//	IPAddress pgtype.Inet `db:"ip_address"`
//	IsActive  bool        `db:"is_active"`
//	ValidFrom time.Time   `db:"valid_from"`
//	ValidTo   *time.Time  `db:"valid_to"`
//}
//
//// use for validation if needed
//func (s *Service) GetIPWhitelistByIP(ip net.IP) (*IPWhitelistEntry, error) {
//
//	var paramIP pgtype.Inet
//	err := paramIP.Set(ip)
//	if err != nil {
//		return nil, err
//	}
//
//	var entry IPWhitelistEntry
//	err = s.DB.Get(&entry, `
//        SELECT builder_id, ip_address, is_active, valid_from, valid_to
//        FROM ip_whitelist
//        WHERE ip_address = $1
//    `, paramIP)
//
//	if err != nil {
//		return nil, fmt.Errorf("error getting IP whitelist entry: %w", err)
//	}
//
//	//entry.IPAddress.IPNet.IP
//	return &entry, nil
//}

// GetMeasurementByOIDAndHash retrieves a measurement by OID and hash
func (s *Service) GetMeasurementByTypeAndHash(attestationType string, hash []byte) (*domain.Measurement, error) {
	var m Measurement
	err := s.DB.Get(&m, `
		SELECT * FROM measurements_whitelist
		WHERE attestation_type = $1 AND hash = $2 AND is_active = true
	`, attestationType, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return convertMeasurementToDomain(m)
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

// GetActiveBuilders retrieves all active builders
func (s *Service) GetActiveBuilders() ([]Builder, error) {
	var builders []Builder
	err := s.DB.Select(&builders, `
		SELECT * FROM builders
		WHERE is_active = true AND deprecated_at IS NULL
	`)
	return builders, err
}

// RegisterCredentialsForBuilder registers new credentials for a builder, deprecating all previous credentials
func (s *Service) RegisterCredentialsForBuilder(builderName, service, tlsCert string, ecdsaPubKey []byte) error {
	// Start a transaction
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback the transaction if it's not committed

	// Deprecate all previous credentials for this builder and service
	_, err = tx.Exec(`
        UPDATE service_credential_registrations
        SET is_active = false
        WHERE builder_name = $1 AND service = $2
    `, builderName, service)
	if err != nil {
		return err
	}

	// Insert new credentials
	var nullableTLSCert sql.NullString
	if tlsCert != "" {
		nullableTLSCert = sql.NullString{String: tlsCert, Valid: true}
	}

	_, err = tx.Exec(`
        INSERT INTO service_credential_registrations 
        (builder_name, service, tls_cert, ecdsa_pubkey, is_active)
        VALUES ($1, $2, $3, $4, true)
    `, builderName, service, nullableTLSCert, ecdsaPubKey)
	if err != nil {
		return err
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// GetActiveConfigForBuilder retrieves the active config for a builder by name
func (s *Service) GetActiveConfigForBuilder(builderName string) (*BuilderConfig, error) {
	var config BuilderConfig
	err := s.DB.Get(&config, `
		SELECT * FROM builder_configs
		WHERE builder_name = $1 AND is_active = true
	`, builderName)
	return &config, err
}

func (s *Service) GetActiveBuildersWithServiceCredentials(ctx context.Context) ([]domain.BuilderWithServices, error) {
	rows, err := s.DB.Query(`
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
		var builderName, service string
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

		if service != "" {
			builder.Credentials = append(builder.Credentials, ServiceCredential{
				Service:     service,
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

//
//// GetActiveBuildersWithCredentials retrieves all active builders along with their active service credentials
//func (s *Service) GetActiveBuildersServices(ctx context.Context) ([]domain.BuilderWithServices, error) {
//	rows, err := s.DB.QueryContext(ctx, `
//        SELECT
//            b.name,
//            b.ip_address,
//            json_agg(
//                json_build_object(
//                    'service', scr.service,
//                    'tls_cert', scr.tls_cert,
//                    'ecdsa_pubkey', encode(scr.ecdsa_pubkey, 'base64')
//                )
//            ) AS credentials
//        FROM
//            builders b
//        LEFT JOIN
//            service_credential_registrations scr ON b.name = scr.builder_name
//        WHERE
//            b.is_active = true AND scr.is_active = true
//        GROUP BY
//            b.name, b.ip_address
//    `)
//	if err != nil {
//		return nil, err
//	}
//	defer rows.Close()
//
//	var builders []BuilderWithCredentials
//	for rows.Next() {
//		var builder BuilderWithCredentials
//		var credentialsJSON []byte
//		err := rows.Scan(&builder.Name, &builder.IPAddress, &credentialsJSON)
//		if err != nil {
//			return nil, err
//		}
//
//		if credentialsJSON != nil {
//			err = json.Unmarshal(credentialsJSON, &builder.Credentials)
//			if err != nil {
//				return nil, err
//			}
//		}
//
//		builders = append(builders, builder)
//	}
//
//	if err = rows.Err(); err != nil {
//		return nil, err
//	}
//
//	var dBuilders []domain.BuilderWithServices
//	for _, b := range builders {
//		dBuilder, err := toDomainBuilderWithCredentials(b)
//		if err != nil {
//			return nil, err
//		}
//		dBuilders = append(dBuilders, *dBuilder)
//	}
//
//	return dBuilders, nil
//}
