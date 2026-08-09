CREATE TYPE geo_data_version_status AS ENUM (
    'IMPORTING',
    'READY',
    'FAILED',
    'SUPERSEDED'
);

CREATE TABLE cities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL UNIQUE CHECK (code ~ '^[a-z][a-z0-9-]*$'),
    name text NOT NULL,
    country_code varchar(2) NOT NULL CHECK (country_code ~ '^[A-Z]{2}$'),
    timezone text NOT NULL,
    boundary geometry(MultiPolygon, 4326),
    created_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON COLUMN cities.boundary IS
    'Nullable during Stage 1.2; the authoritative city boundary is added with district geo data.';

INSERT INTO cities (id, code, name, country_code, timezone)
VALUES (
    '01900000-0000-7000-8000-000000000001',
    'spb',
    'Санкт-Петербург',
    'RU',
    'Europe/Moscow'
);

CREATE TABLE geo_data_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id uuid NOT NULL REFERENCES cities(id),
    source text NOT NULL,
    source_url text,
    source_timestamp timestamptz,
    source_checksum varchar(64) NOT NULL CHECK (source_checksum ~ '^[0-9a-f]{64}$'),
    source_file_name text NOT NULL,
    source_size_bytes bigint NOT NULL CHECK (source_size_bytes > 0),
    normalization_version text NOT NULL,
    status geo_data_version_status NOT NULL,
    import_started_at timestamptz NOT NULL DEFAULT now(),
    import_finished_at timestamptz,
    imported_at timestamptz,
    import_report jsonb NOT NULL DEFAULT '{}'::jsonb,
    import_error text,
    CONSTRAINT geo_data_versions_lifecycle_check CHECK (
        (status = 'IMPORTING' AND import_finished_at IS NULL AND imported_at IS NULL AND import_error IS NULL)
        OR
        (status = 'FAILED' AND import_finished_at IS NOT NULL AND imported_at IS NULL AND import_error IS NOT NULL)
        OR
        (status IN ('READY', 'SUPERSEDED') AND import_finished_at IS NOT NULL AND imported_at IS NOT NULL AND import_error IS NULL)
    )
);

CREATE UNIQUE INDEX geo_data_versions_one_importing_per_city
    ON geo_data_versions (city_id)
    WHERE status = 'IMPORTING';

CREATE UNIQUE INDEX geo_data_versions_one_ready_per_city
    ON geo_data_versions (city_id)
    WHERE status = 'READY';

CREATE UNIQUE INDEX geo_data_versions_ready_source_identity
    ON geo_data_versions (city_id, source_checksum, normalization_version)
    WHERE status = 'READY';

CREATE INDEX geo_data_versions_city_started_at
    ON geo_data_versions (city_id, import_started_at DESC);
