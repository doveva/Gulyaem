CREATE TYPE street_segment_classification AS ENUM (
    'EXPLORE',
    'ROUTABLE_ONLY',
    'IGNORE'
);

ALTER TABLE geo_data_versions
    ADD CONSTRAINT geo_data_versions_id_city_unique UNIQUE (id, city_id);

CREATE TABLE streets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id uuid NOT NULL REFERENCES cities(id),
    name text,
    normalized_name text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT streets_id_city_unique UNIQUE (id, city_id)
);

CREATE INDEX streets_city_normalized_name
    ON streets (city_id, normalized_name)
    WHERE normalized_name IS NOT NULL;

CREATE TABLE street_segments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    city_id uuid NOT NULL,
    geo_data_version_id uuid NOT NULL,
    street_id uuid,
    geometry geometry(LineString, 4326) NOT NULL,
    length_m double precision NOT NULL CHECK (
        length_m > 0 AND length_m < 'Infinity'::double precision
    ),
    classification street_segment_classification NOT NULL,
    attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT street_segments_version_city_fk
        FOREIGN KEY (geo_data_version_id, city_id)
        REFERENCES geo_data_versions (id, city_id)
        ON DELETE CASCADE,
    CONSTRAINT street_segments_street_city_fk
        FOREIGN KEY (street_id, city_id)
        REFERENCES streets (id, city_id),
    CONSTRAINT street_segments_geometry_check CHECK (
        NOT ST_IsEmpty(geometry)
        AND ST_IsValid(geometry)
        AND ST_NPoints(geometry) >= 2
    )
);

CREATE INDEX street_segments_geometry_gist
    ON street_segments USING gist (geometry);

CREATE INDEX street_segments_version_classification
    ON street_segments (geo_data_version_id, classification);

CREATE INDEX street_segments_city_version
    ON street_segments (city_id, geo_data_version_id);
