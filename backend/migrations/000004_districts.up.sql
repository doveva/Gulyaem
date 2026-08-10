CREATE TABLE district_data_versions (
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
    CONSTRAINT district_data_versions_id_city_unique UNIQUE (id, city_id),
    CONSTRAINT district_data_versions_lifecycle_check CHECK (
        (status = 'IMPORTING' AND import_finished_at IS NULL AND imported_at IS NULL AND import_error IS NULL)
        OR
        (status = 'FAILED' AND import_finished_at IS NOT NULL AND imported_at IS NULL AND import_error IS NOT NULL)
        OR
        (status IN ('READY', 'SUPERSEDED') AND import_finished_at IS NOT NULL AND imported_at IS NOT NULL AND import_error IS NULL)
    )
);

CREATE UNIQUE INDEX district_data_versions_one_importing_per_city
    ON district_data_versions (city_id)
    WHERE status = 'IMPORTING';

CREATE UNIQUE INDEX district_data_versions_one_ready_per_city
    ON district_data_versions (city_id)
    WHERE status = 'READY';

CREATE UNIQUE INDEX district_data_versions_ready_source_identity
    ON district_data_versions (city_id, source_checksum, normalization_version)
    WHERE status = 'READY';

CREATE INDEX district_data_versions_city_started_at
    ON district_data_versions (city_id, import_started_at DESC);

CREATE TABLE districts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id uuid NOT NULL,
    district_data_version_id uuid NOT NULL,
    external_id text NOT NULL,
    name text NOT NULL,
    kind text NOT NULL,
    boundary geometry(MultiPolygon, 4326) NOT NULL,
    label_point geometry(Point, 4326) NOT NULL,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT districts_version_city_fk
        FOREIGN KEY (district_data_version_id, city_id)
        REFERENCES district_data_versions (id, city_id)
        ON DELETE CASCADE,
    CONSTRAINT districts_version_external_id_unique
        UNIQUE (district_data_version_id, external_id),
    CONSTRAINT districts_boundary_check CHECK (
        NOT ST_IsEmpty(boundary)
        AND ST_IsValid(boundary)
    ),
    CONSTRAINT districts_label_point_check CHECK (
        NOT ST_IsEmpty(label_point)
        AND ST_Covers(boundary, label_point)
    )
);

CREATE INDEX districts_boundary_gist ON districts USING gist (boundary);
CREATE INDEX districts_city_version ON districts (city_id, district_data_version_id);

