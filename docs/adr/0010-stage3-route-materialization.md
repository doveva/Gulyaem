# ADR-0010: Server-side materialization of Stage 2 route preview

**Status:** Proposed  
**Date:** 2026-08-12  
**Stage:** 3

## Context

Stage 2 `RoutePreview` is intentionally stateless. Stage 3 needs a persistent Route that becomes
part of Walk history and exploration source data.

Persisting the route geometry returned back by the browser would make client data authoritative and
would allow stale/tampered geometry or a different geo/routing version to enter history.

Recomputing without checking what the user reviewed can silently persist a different route after
routing/geo data changes.

## Decision

Stage 2 preview gains opaque:

```text
previewFingerprint
```

When creating/correcting a persistent Walk route, client submits:

```text
ordered waypoints
profile
expectedPreviewFingerprint
```

Backend runs the existing Stage 2 RoutePreview pipeline again and computes the same versioned
fingerprint.

If fingerprints differ:

```text
409 route_preview_stale
```

Nothing persistent is changed.

If equal, backend persists the trusted server result.

The browser never provides authoritative route geometry or StreetSegment coverage.

Fingerprint is versioned and includes all server-side materialization state capable of changing the
route/analysis, including geo/routing provenance.

## Consequences

Pros:

- client is not route source of truth;
- user does not silently start a route different from reviewed preview;
- current Stage 2 pipeline is reused;
- persistence records explicit provenance.

Cons:

- Start/correction repeats routing + analysis;
- materialization latency remains roughly Stage 2 preview latency;
- fingerprint canonicalization becomes a tested internal contract.

## Alternatives rejected

### Persist browser response directly

Rejected because browser is untrusted and preview may be stale.

### Persist every Stage 2 preview server-side

Rejected because Stage 2 statelessness is useful and most previews are transient.

### Recompute silently without fingerprint

Rejected because route may change between review and materialization.
