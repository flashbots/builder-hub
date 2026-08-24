ALTER TABLE measurements_whitelist
    ALTER COLUMN measurement DROP NOT NULL,
    ADD COLUMN dcap_image_hashes JSONB,
    ADD CONSTRAINT measurement_policy_shape CHECK (
        (measurement IS NULL) <> (dcap_image_hashes IS NULL)
        AND (measurement IS NULL OR jsonb_typeof(measurement) = 'object')
        AND (dcap_image_hashes IS NULL OR jsonb_typeof(dcap_image_hashes) = 'object')
    );
