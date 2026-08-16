CREATE TYPE walk_status AS ENUM (
    'DRAFT',
    'ACTIVE',
    'REVIEW',
    'COMPLETED',
    'CANCELLED'
);

CREATE TYPE route_coverage_status AS ENUM (
    'COMPLETED',
    'PARTIAL',
    'CONNECTOR'
);

CREATE TYPE exploration_delta_segment_kind AS ENUM (
    'NEW',
    'REVISITED'
);

CREATE TYPE exploration_state_status AS ENUM (
    'READY',
    'REBUILD_REQUIRED'
);

CREATE TABLE routes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    geo_data_version_id uuid NOT NULL,
    source_type text NOT NULL DEFAULT 'MANUAL' CHECK (source_type = 'MANUAL'),
    profile text NOT NULL CHECK (profile = 'pedestrian'),
    waypoints jsonb NOT NULL CHECK (jsonb_typeof(waypoints) = 'array' AND jsonb_array_length(waypoints) BETWEEN 2 AND 10),
    geometry geometry(LineString, 4326) NOT NULL,
    normalized_geometry geometry(MultiLineString, 4326) NOT NULL,
    distance_m double precision NOT NULL CHECK (distance_m > 0 AND distance_m < 'Infinity'::double precision),
    estimated_duration_sec integer NOT NULL CHECK (estimated_duration_sec > 0),
    routing_provenance jsonb NOT NULL,
    analysis_provenance jsonb NOT NULL,
    materialization_fingerprint text NOT NULL,
    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    finalized_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT routes_version_city_fk FOREIGN KEY (geo_data_version_id, city_id)
        REFERENCES geo_data_versions(id, city_id),
    CONSTRAINT routes_geometry_check CHECK (
        NOT ST_IsEmpty(geometry) AND ST_IsValid(geometry) AND ST_NPoints(geometry) >= 2
    ),
    CONSTRAINT routes_normalized_geometry_check CHECK (
        NOT ST_IsEmpty(normalized_geometry) AND ST_IsValid(normalized_geometry)
    )
);

CREATE TABLE route_segment_matches (
    route_id uuid NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    street_segment_id uuid NOT NULL REFERENCES street_segments(id),
    classification street_segment_classification NOT NULL,
    matched_length_m double precision NOT NULL DEFAULT 0 CHECK (matched_length_m >= 0),
    covered_length_m double precision NOT NULL CHECK (covered_length_m >= 0),
    direct_length_m double precision NOT NULL DEFAULT 0 CHECK (direct_length_m >= 0),
    required_length_m double precision NOT NULL CHECK (required_length_m >= 0),
    coverage_status route_coverage_status NOT NULL,
    provenance text,
    confidence double precision CHECK (confidence IS NULL OR confidence BETWEEN 0 AND 1),
    PRIMARY KEY (route_id, street_segment_id)
);

CREATE TABLE walks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    route_id uuid NOT NULL REFERENCES routes(id),
    client_request_id uuid NOT NULL,
    request_fingerprint text NOT NULL,
    status walk_status NOT NULL DEFAULT 'DRAFT',
    started_at timestamptz,
    finished_at timestamptz,
    completed_at timestamptz,
    duration_sec integer CHECK (duration_sec IS NULL OR duration_sec >= 0),
    distance_m double precision CHECK (distance_m IS NULL OR distance_m >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (actor_id, client_request_id),
    CONSTRAINT walks_lifecycle_check CHECK (
        (status = 'DRAFT' AND started_at IS NULL AND finished_at IS NULL AND completed_at IS NULL)
        OR (status = 'ACTIVE' AND started_at IS NOT NULL AND finished_at IS NULL AND completed_at IS NULL)
        OR (status = 'REVIEW' AND started_at IS NOT NULL AND finished_at IS NOT NULL AND completed_at IS NULL)
        OR (status = 'COMPLETED' AND started_at IS NOT NULL AND finished_at IS NOT NULL AND completed_at IS NOT NULL
            AND duration_sec IS NOT NULL AND distance_m IS NOT NULL)
        OR (status = 'CANCELLED' AND completed_at IS NULL)
    )
);

CREATE TABLE user_street_segment_progress (
    actor_id uuid NOT NULL,
    street_segment_id uuid NOT NULL REFERENCES street_segments(id),
    first_explored_at timestamptz NOT NULL,
    last_explored_at timestamptz NOT NULL,
    visit_count integer NOT NULL CHECK (visit_count > 0),
    first_walk_id uuid NOT NULL REFERENCES walks(id),
    last_walk_id uuid NOT NULL REFERENCES walks(id),
    PRIMARY KEY (actor_id, street_segment_id),
    CHECK (last_explored_at >= first_explored_at)
);

CREATE TABLE exploration_states (
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    geo_data_version_id uuid NOT NULL,
    status exploration_state_status NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    rebuilt_at timestamptz,
    PRIMARY KEY (actor_id, city_id),
    FOREIGN KEY (geo_data_version_id, city_id) REFERENCES geo_data_versions(id, city_id)
);

CREATE TABLE exploration_deltas (
    walk_id uuid PRIMARY KEY REFERENCES walks(id),
    actor_id uuid NOT NULL,
    geo_data_version_id uuid NOT NULL REFERENCES geo_data_versions(id),
    new_segments_count integer NOT NULL CHECK (new_segments_count >= 0),
    revisited_segments_count integer NOT NULL CHECK (revisited_segments_count >= 0),
    new_network_length_m double precision NOT NULL CHECK (new_network_length_m >= 0),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE exploration_delta_segments (
    walk_id uuid NOT NULL REFERENCES exploration_deltas(walk_id) ON DELETE CASCADE,
    street_segment_id uuid NOT NULL REFERENCES street_segments(id),
    kind exploration_delta_segment_kind NOT NULL,
    segment_length_m double precision NOT NULL CHECK (segment_length_m > 0),
    covered_length_m double precision NOT NULL CHECK (covered_length_m >= 0),
    PRIMARY KEY (walk_id, street_segment_id)
);

CREATE TABLE walk_district_deltas (
    walk_id uuid NOT NULL REFERENCES exploration_deltas(walk_id) ON DELETE CASCADE,
    district_id uuid NOT NULL REFERENCES districts(id),
    district_data_version_id uuid NOT NULL REFERENCES district_data_versions(id),
    geo_data_version_id uuid NOT NULL REFERENCES geo_data_versions(id),
    eligible_length_m double precision NOT NULL CHECK (eligible_length_m > 0),
    explored_before_m double precision NOT NULL CHECK (explored_before_m >= 0),
    explored_after_m double precision NOT NULL CHECK (explored_after_m >= explored_before_m),
    new_length_m double precision NOT NULL CHECK (new_length_m >= 0),
    percentage_before double precision NOT NULL CHECK (percentage_before BETWEEN 0 AND 1),
    percentage_after double precision NOT NULL CHECK (percentage_after BETWEEN 0 AND 1),
    PRIMARY KEY (walk_id, district_id)
);

CREATE INDEX routes_actor_created_at ON routes(actor_id, created_at DESC);
CREATE INDEX routes_city_version ON routes(city_id, geo_data_version_id);
CREATE INDEX route_segment_matches_segment ON route_segment_matches(street_segment_id);
CREATE INDEX walks_actor_status ON walks(actor_id, status);
CREATE INDEX walks_actor_created_at ON walks(actor_id, created_at DESC);
CREATE INDEX user_progress_actor_last ON user_street_segment_progress(actor_id, last_explored_at DESC);
CREATE INDEX exploration_delta_segments_segment ON exploration_delta_segments(street_segment_id);
