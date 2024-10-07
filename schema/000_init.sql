CREATE TABLE builders (
                          name VARCHAR(255) PRIMARY KEY,
                          ip_address INET NOT NULL,
                          is_active BOOLEAN NOT NULL DEFAULT true,
                          created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                          deprecated_at TIMESTAMP WITH TIME ZONE,
                          CONSTRAINT active_only_when_not_deprecated CHECK (
                                  (is_active = true AND deprecated_at IS NULL) OR
                                  (is_active = false)
                              )
);

-- Add an index on ip_address for faster lookups
CREATE INDEX idx_builders_ip_address ON builders(ip_address);

-- Trigger to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_builders_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_builders_updated_at
    BEFORE UPDATE ON builders
    FOR EACH ROW
EXECUTE FUNCTION update_builders_updated_at();

-- Measurements Whitelist table
CREATE TABLE measurements_whitelist (
                                        hash  BYTEA PRIMARY KEY , -- hash of the measurement
                                        attestation_type  TEXT NOT NULL, -- attestation type of the measurement
                                        measurement JSONB NOT NULL,
                                        is_active BOOLEAN NOT NULL DEFAULT true,

                                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                        deprecated_at TIMESTAMP WITH TIME ZONE,
                                        CONSTRAINT active_only_when_not_deprecated CHECK (
                                                (is_active = true AND deprecated_at IS NULL) OR
                                                (is_active = false)
                                            )
);



CREATE TABLE service_credential_registrations (
                                                  id SERIAL PRIMARY KEY,
                                                  builder_name VARCHAR(255) REFERENCES builders(name),
                                                  service TEXT NOT NULL,
                                                  tls_cert TEXT,
                                                  ecdsa_pubkey BYTEA,
                                                  is_active BOOLEAN NOT NULL DEFAULT false,
                                                  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                                  updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_unique_active_credential_per_builder_service
    ON service_credential_registrations (builder_name, service)
    WHERE is_active = true;

-- Trigger to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_service_credential_registrations_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_service_credential_registrations_updated_at
    BEFORE UPDATE ON service_credential_registrations
    FOR EACH ROW
EXECUTE FUNCTION update_service_credential_registrations_updated_at();

-- Updated builder_configs table
CREATE TABLE builder_configs (
                                 id SERIAL PRIMARY KEY,
                                 builder_name VARCHAR(255) REFERENCES builders(name), -- references name
                                 config JSONB NOT NULL,
                                 is_active BOOLEAN NOT NULL DEFAULT false,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);



-- Add a new constraint to ensure only one active config per builder
ALTER TABLE builder_configs
    ADD CONSTRAINT unique_active_config_per_builder
        EXCLUDE (builder_name WITH =)
        WHERE (is_active = true);



