# Stage 2 — Frontend Route Builder Flow

# 1. Product route

```text
/map
```

Stage 2 turns `/map` from primarily a map shell into the first user-facing workflow.

`/debug/geo` remains available separately.

---

# 2. Builder state model

Recommended conceptual states:

```text
IDLE
EDITING
CALCULATING
READY
ERROR
```

State details should come from actual data rather than duplicated booleans where possible.

---

# 3. IDLE

Map is visible.

Primary action:

> + Прогулка

On click/tap:

```text
IDLE → EDITING
```

Instruction:

> Выберите начало маршрута

---

# 4. Start selected

First map tap:

```text
waypoints = [A]
```

Show marker A.

Instruction:

> Выберите точку назначения

No backend routing request yet.

---

# 5. Destination selected

Second tap:

```text
waypoints = [A, B]
```

Immediately enter:

```text
CALCULATING
```

Send route preview request.

---

# 6. READY

Render:

- route geometry;
- waypoint markers;
- distance;
- duration;
- potential completed segments;
- potential partial segments.

Suggested sheet:

```text
4.2 км
≈ 1 ч

Потенциально засчитывается
27 сегментов

Частично
6 сегментов
```

Avoid personal-history copy.

---

# 7. Add intermediate waypoint

Explicit action:

> + Точка

Activates one-shot `ADD_WAYPOINT` interaction.

Next map tap inserts a waypoint before destination:

```text
A → C → B
```

Then recalculate.

This is preferred to making every arbitrary route-map click mutate the route.

---

# 8. Drag waypoint

Waypoint markers are draggable.

During drag:

- move marker visually;
- do not continuously call route preview.

On `dragend`:

- update coordinate;
- invalidate current preview;
- request new preview.

---

# 9. Delete waypoint

Intermediate waypoint may be removed.

Start/destination replacement should use editing/dragging rather than leaving invalid one-point route state accidentally.

If user explicitly clears route:

```text
waypoints = []
preview = null
state = EDITING/initial
```

---

# 10. Reorder waypoints

Intermediate waypoints should be reorderable from ordered list.

Reorder triggers one recalculation after completion.

Start remains first, destination remains last.

---

# 11. Stale response protection

Example:

```text
Request #10 → route A-B
user moves B
Request #11 → route A-C
#11 returns first
#10 returns later
```

UI MUST keep A-C.

Implementation options:

- AbortController;
- request sequence;
- state-version token.

Prefer using both cancellation and state validation.

---

# 12. Loading state

Do not present old geometry as unquestionably current.

Allowed approaches:

- dim previous route;
- mark `Пересчитываем…`;
- temporarily hide exploration highlight.

Waypoint markers always represent latest user intent.

---

# 13. Error states

## No route

> Не удалось построить пешеходный маршрут между выбранными точками.

Keep waypoints editable.

## Routing unavailable

> Построение маршрута временно недоступно.

Do not clear waypoints.

## Dataset mismatch

Internal/development:

> Routing graph does not match current geo data.

For Stage 2 this should be obvious and actionable to developer.

## Low route match

Preview may render with warning:

> Часть маршрута не удалось точно сопоставить с городской сетью.

Do not imply precise exploration for unmatched fragment.

---

# 14. Product map layers

Recommended order:

```text
base
background street network (when detailed zoom)
potential PARTIAL
potential COMPLETED
route geometry
waypoint markers
```

Route must remain visually primary.

---

# 15. Debug diagnostics

Do not overload `/map` with Stage 1 diagnostics.

Detailed:

- match confidence;
- reason codes;
- source OSM metadata;
- strict/generous profiles;
- unmatched reasons;

remain in `/debug/geo` or dev inspector.

Product can expose only compact warning/state.

---

# 16. Mobile behavior

Mobile-first requirements:

- route actions reachable with one hand where practical;
- touch targets adequate;
- bottom sheet must not permanently hide map route;
- marker drag works with touch;
- sheet can collapse/expand.

No requirement for full gesture-perfect final design; functional usability is the Stage 2 target.

---

# 17. Desktop behavior

Same capabilities:

- map click;
- marker drag;
- waypoint list;
- route summary.

Desktop may use side panel instead of bottom sheet.

---

# 18. No Stage 3 CTA

Do not fake an operational:

> Начать прогулку

unless it is clearly disabled as future functionality.

Preferred Stage 2 final state is preview/editing.

Stage 3 introduces actual Walk lifecycle.
