# Stage 3 — Persistence Model

This is a schema-oriented proposal, not a literal migration file. Names may be adjusted to match
repository conventions while preserving constraints.

# 1. New enum types

```sql
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
```

Avoid a PostgreSQL route-source enum unless Stage 3 needs more than `MANUAL`; a checked text column
is acceptable for easier future extension.

# 2. `routes`

Suggested:

```sql
CREATE TABLE routes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    geo_data_version_id uuid NOT NULL,
    source_type text NOT NULL CHECK (source_type = 'MANUAL'),
    profile text NOT NULL CHECK (profile = 'pedestrian'),

    waypoints jsonb NOT NULL,
    geometry geometry(LineString, 4326) NOT NULL,
    normalized_geometry geometry(MultiLineString, 4326) NOT NULL,

    distance_m double precision NOT NULL CHECK (distance_m > 0),
    estimated_duration_sec integer NOT NULL CHECK (estimated_duration_sec > 0),

    routing_provenance jsonb NOT NULL,
    analysis_provenance jsonb NOT NULL,

    revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
    finalized_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    FOREIGN KEY (geo_data_version_id, city_id)
      REFERENCES geo_data_versions(id, city_id)
);
```

Geometry checks should mirror existing StreetSegment standards:

```text
not empty
valid
LineString has >=2 points
```

# 3. `route_segment_matches`

```sql
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
    confidence double precision,

    PRIMARY KEY (route_id, street_segment_id),

    CHECK (confidence IS NULL OR (confidence >= 0 AND confidence <= 1))
);
```

Persist only `COMPLETED`, `PARTIAL`, `CONNECTOR` result rows.

# 4. `walks`

```sql
CREATE TABLE walks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    route_id uuid NOT NULL REFERENCES routes(id),

    client_request_id uuid NOT NULL,
    status walk_status NOT NULL DEFAULT 'DRAFT',

    started_at timestamptz,
    finished_at timestamptz,
    completed_at timestamptz,

    duration_sec integer,
    distance_m double precision,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    UNIQUE (actor_id, client_request_id)
);
```

Lifecycle check constraints are recommended so impossible timestamp/status combinations cannot be
persisted accidentally.

Do NOT make `route_id` globally unique: future domain may reuse route concepts. Stage 3 services
still create a dedicated Route per Walk.

# 5. `user_street_segment_progress`

```sql
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
```

StreetSegment itself identifies geo version; no duplicate `geo_data_version_id` is required here.

Suggested read index:

```sql
CREATE INDEX user_progress_actor_last
    ON user_street_segment_progress(actor_id, last_explored_at DESC);
```

# 6. `exploration_states`

```sql
CREATE TABLE exploration_states (
    actor_id uuid NOT NULL,
    city_id uuid NOT NULL REFERENCES cities(id),
    geo_data_version_id uuid NOT NULL,
    status exploration_state_status NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    rebuilt_at timestamptz,

    PRIMARY KEY (actor_id, city_id),

    FOREIGN KEY (geo_data_version_id, city_id)
      REFERENCES geo_data_versions(id, city_id)
);
```

This table prevents silently treating old-version progress as current.

# 7. `exploration_deltas`

```sql
CREATE TABLE exploration_deltas (
    walk_id uuid PRIMARY KEY REFERENCES walks(id),
    actor_id uuid NOT NULL,
    geo_data_version_id uuid NOT NULL REFERENCES geo_data_versions(id),

    new_segments_count integer NOT NULL CHECK (new_segments_count >= 0),
    revisited_segments_count integer NOT NULL CHECK (revisited_segments_count >= 0),
    new_network_length_m double precision NOT NULL CHECK (new_network_length_m >= 0),

    created_at timestamptz NOT NULL DEFAULT now()
);
```

One Walk has at most one completion delta.

This uniqueness is part of completion idempotency.

# 8. `exploration_delta_segments`

```sql
CREATE TABLE exploration_delta_segments (
    walk_id uuid NOT NULL REFERENCES exploration_deltas(walk_id) ON DELETE CASCADE,
    street_segment_id uuid NOT NULL REFERENCES street_segments(id),
    kind exploration_delta_segment_kind NOT NULL,
    segment_length_m double precision NOT NULL CHECK (segment_length_m > 0),
    covered_length_m double precision NOT NULL CHECK (covered_length_m >= 0),

    PRIMARY KEY (walk_id, street_segment_id)
);
```

# 9. `walk_district_deltas`

```sql
CREATE TABLE walk_district_deltas (
    walk_id uuid NOT NULL REFERENCES exploration_deltas(walk_id) ON DELETE CASCADE,
    district_id uuid NOT NULL REFERENCES districts(id),
    district_data_version_id uuid NOT NULL,
    geo_data_version_id uuid NOT NULL,

    eligible_length_m double precision NOT NULL CHECK (eligible_length_m > 0),
    explored_before_m double precision NOT NULL CHECK (explored_before_m >= 0),
    explored_after_m double precision NOT NULL CHECK (explored_after_m >= explored_before_m),
    new_length_m double precision NOT NULL CHECK (new_length_m >= 0),

    percentage_before double precision NOT NULL CHECK (percentage_before BETWEEN 0 AND 1),
    percentage_after double precision NOT NULL CHECK (percentage_after BETWEEN 0 AND 1),

    PRIMARY KEY (walk_id, district_id)
);
```

Add composite FKs/version checks where current schema makes them practical.

# 10. Route indexes

Suggested:

```text
routes(actor_id, created_at)
routes(city_id, geo_data_version_id)
walks(actor_id, status)
walks(actor_id, created_at)
route_segment_matches(street_segment_id)
exploration_delta_segments(street_segment_id)
```

# 11. Ownership queries

Every Walk/Route access path must include actor scope.

Preferred repository method shape:

```text
Walk(ctx, actorID, walkID)
```

not:

```text
Walk(ctx, walkID)
```

followed by authorization elsewhere.

# 12. Completion transaction locking

At minimum:

```sql
SELECT ... FROM walks
WHERE id = $walk_id AND actor_id = $actor_id
FOR UPDATE;
```

Completion also holds the current version stable through commit:

```sql
SELECT id FROM geo_data_versions
WHERE city_id = $city_id AND status = 'READY'
FOR SHARE;
```

Route materialization and correction similarly lock their expected version by ID/city/status before
writing `routes`. These locks serialize with the geo publisher's `READY → SUPERSEDED` update.

Walk ownership is also enforced structurally. `walks(route_id, actor_id, city_id)` references
`routes(id, actor_id, city_id)`, so an aggregate cannot attach another actor's or city's Route even
if a repository bug bypasses application-level ownership checks. Actor-scoped aggregate,
completion and rebuild queries repeat all three join predicates as defense in depth.

`route_segment_matches` stays narrow, but its repository insert is an `INSERT ... SELECT` joining
the target Route and StreetSegment on `geo_data_version_id`. Exactly one inserted row is required;
a missing or cross-version segment aborts the whole materialization as `route_preview_stale`.

Progress rows may be inserted/upserted in deterministic StreetSegment ID order to reduce deadlock
risk under concurrent requests.

# 13. Completion idempotency constraints

Use:

- Walk terminal status;
- one `exploration_deltas.walk_id`;
- one delta segment per Walk/StreetSegment;
- row locking.

Do not rely solely on an in-memory lock.

# 14. Walk creation idempotency

Unique:

```text
(actor_id, client_request_id)
```

On conflict, return existing Walk if request semantics are compatible.

If same `clientRequestId` is reused with different route materialization input, return conflict rather
than silently return unrelated Walk.

A materialization/request fingerprint may be stored on Walk/Route to verify this.

# 15. District calculation

No new district-membership table is mandatory initially.

Start with spatial clipping query using existing GiST indexes.

If completion/read performance requires materialization, introduce a derived table through separate
evidence/ADR while preserving:

```text
intersection length inside district
```

semantics.

# 16. Rebuild publication transaction

Rebuild may perform expensive route analysis before final write transaction.

Final transaction:

1. verify target version still READY/current;
2. delete/replace actor+city materialized progress;
3. set `exploration_states` to target version READY;
4. commit.

Historical deltas remain untouched.

# 17. Deletion semantics

Stage 3 does not expose deletion of COMPLETED Walk.

Do not add cascade behavior that would accidentally delete completed Walk history when materialized
progress is rebuilt.
