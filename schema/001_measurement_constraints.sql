-- Drop the existing unique constraint
ALTER TABLE measurements_whitelist
    DROP CONSTRAINT unique_hash_attestation_type;

-- Create a new unique constraint on the 'name' column
ALTER TABLE measurements_whitelist
    ADD CONSTRAINT unique_name UNIQUE (name);
