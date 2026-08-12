# ADR-0013: Actor-scoped Stage 3 data before authentication

**Status:** Proposed  
**Date:** 2026-08-12  
**Stage:** 3

## Context

Stage 3 needs personal Walk/progress semantics, but roadmap places registration/authentication in
Stage 4.

Hardcoding unscoped global progress would require a domain/schema rewrite later and would not test
ownership boundaries.

Implementing full identity now would expand Stage 3 scope.

## Decision

All Stage 3 user-owned persistence uses:

```text
actor_id UUID
```

Application services receive ActorID explicitly.

In development/test HTTP, ActorID is resolved server-side from:

```text
DEVELOPMENT_ACTOR_ID
```

Browser cannot provide arbitrary actor ID.

No account/authentication UI or identity provider is introduced.

Stage 4 replaces the actor resolver with authenticated principal resolution and may introduce a User
table/FK without changing ownership semantics.

## Consequences

Pros:

- personal progress semantics are real in Stage 3;
- auth remains outside scope;
- ownership tests can use multiple actor IDs;
- Stage 4 integration point is explicit.

Cons:

- development runtime is effectively single-user;
- actor referential integrity may be weaker until Stage 4 User model exists;
- this resolver must never be mistaken for production authentication.

## Alternatives rejected

### Global single-user rows without actor_id

Rejected because it creates schema/domain migration debt.

### Implement authentication in Stage 3

Rejected because it does not validate the exploration core and belongs to Stage 4.

### Client-supplied actor header

Rejected because browser-controlled owner selection is an unsafe boundary even in an early product.
