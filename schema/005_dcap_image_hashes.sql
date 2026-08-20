ALTER TABLE measurements_whitelist
    ALTER COLUMN measurement DROP NOT NULL,
    ADD COLUMN dcap_image_hashes JSONB,
    ADD CONSTRAINT measurement_policy_shape CHECK (
        measurement IS NULL OR dcap_image_hashes IS NULL
    );
