# Stage 3 — Frontend Flow

# 1. Product flow

Stage 3 extends existing `/map`.

```text
MAP
 ↓
ROUTE_PREVIEW
 ↓
MATERIALIZING
 ↓
ACTIVE_WALK
 ↓
REVIEW
 ↓
COMPLETING
 ↓
WALK_SUMMARY
 ↓
MAP_WITH_UPDATED_EXPLORATION
```

# 2. Map idle state

Map loads:

- existing base map;
- current actor explored StreetSegment overlay;
- district progress summary/card;
- Stage 2 route builder CTA.

If actor has no progress, overlay is empty but valid.

If exploration state requires rebuild, show explicit development/product-safe message rather than
displaying zero progress as truth.

# 3. Route preview

Reuse Stage 2 builder.

Preview now also contains:

```text
previewFingerprint
```

Product copy may distinguish:

- already explored persistent layer;
- potential route completion layer.

Do not change Stage 2 coverage semantics.

# 4. Start CTA

When preview is READY:

> **Начать прогулку**

Flow:

```text
generate clientRequestId
↓
POST /walks
↓
persist returned walkId locally
↓
POST /walks/{id}/start
↓
ACTIVE_WALK
```

If create succeeds but start fails, retain DRAFT Walk ID and offer retry.

# 5. Stale preview during start

On:

```text
409 route_preview_stale
```

Frontend:

1. does not create optimistic active state;
2. refreshes preview;
3. explains that route changed;
4. requires user to press Start again.

# 6. Active Walk screen

Show:

- planned route;
- Start / Destination markers optionally simplified;
- elapsed time calculated from `startedAt`;
- planned route distance;
- Finish;
- Cancel.

Do NOT show:

- live location;
- walked distance;
- pace;
- calories;
- speed.

Stage 3 is not GPS tracking.

# 7. Durable recovery

Store:

```text
gulyaem.activeWalkId
```

or equivalent.

On app load:

```text
if activeWalkId:
    GET /walks/{id}
```

Restore:

- DRAFT;
- ACTIVE;
- REVIEW.

If backend says COMPLETED/CANCELLED/not found, clear local pointer.

Backend state wins over local state.

# 8. Finish

Finish action:

```text
POST /walks/{id}/finish
```

Then show REVIEW.

Do not display exploration reward before completion.

# 9. Review screen

Show final planned route and route-analysis overlay.

Primary actions:

- **Подтвердить прогулку**
- **Исправить маршрут**
- Cancel

No GPS trace exists, so initial review geometry is the materialized manual route.

# 10. Correct route

Reuse Stage 2 waypoint builder initialized from current Route waypoints.

User edits and receives normal stateless previews.

When satisfied:

```text
PUT /walks/{id}/route
```

with preview fingerprint.

On success return to REVIEW with new revision.

# 11. Geo-version conflict during review

If current geo data changed:

- fresh Stage 2 preview uses current version;
- saving corrected route rematerializes current version;
- old route remains until update succeeds.

Do not silently swap route in background.

# 12. Complete

On confirm:

```text
POST /walks/{id}/complete
```

Show blocking `COMPLETING` state.

No optimistic personal progress.

Only successful backend response updates local/query cache.

# 13. Rebuild-required conflict

If completion returns:

```text
exploration_rebuild_required
```

Stage 3 development UI may show:

> Данные исследования требуют пересчёта перед завершением прогулки.

No fake progress.

Automatic background rebuild is not required.

# 14. Walk Summary

Minimum reward screen:

```text
Новая часть города открыта

+3.1 км исследованной сети
17 новых сегментов
32 уже знакомых

Центральный район
31% → 34%
```

Map displays:

- final route;
- newly explored segments;
- previously explored context with lower emphasis.

Exact copy can be tuned, but metrics must preserve semantics.

# 15. No-new Walk

Valid summary:

```text
Новых сегментов нет
Маршрут прошёл по уже исследованной части города
```

Do not treat zero-new as an error.

# 16. Return to map

CTA:

> **Вернуться к карте**

Then:

- invalidate/refetch exploration state;
- remove activeWalkId;
- clear route builder;
- show newly persistent explored overlay.

# 17. Reload after completion

Map reload from backend must show same explored result without relying on client cache.

# 18. Cancellation

From DRAFT/ACTIVE/REVIEW:

- show confirmation;
- call cancel;
- clear activeWalkId;
- return to map;
- do not change progress.

# 19. Responsive/mobile

Physical-device requirements:

- Start/Finish/Complete reachable;
- bottom sheet does not hide relevant route;
- review correction remains usable with touch;
- newly explored overlay readable in sunlight-normal contrast where practical;
- reload recovery works.

# 20. State separation

Recommended frontend state categories:

## Server state

```text
Walk
Route
Exploration
```

## Route-builder state

Existing Stage 2 waypoints/preview.

## Durable local state

```text
activeWalkId
```

Do not persist exploration delta as durable client truth.

# 21. No Stage 4 UI

Do not add:

- login/register;
- Profile;
- Walk history list;
- account settings.

Navigation placeholders may remain.
