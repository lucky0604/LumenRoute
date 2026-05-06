# CLAUDE.md

## Project Intent

This repository is currently a greenfield, PRD-driven project. The main source of truth is `ideas/prd.md`, which describes a lightweight internal model routing/control plane with a Go backend and React frontend.

When implementing, optimize for clarity, maintainability, and explicit architecture over speed alone.

## Mandatory Constraints

1. **Project structure must stay engineered and structured.**
   - Prefer deliberate top-level folders with clear ownership.
   - Do not create ad hoc directories for convenience.
   - Keep domain, transport, persistence, and UI concerns separated.

2. **Single code files should usually stay under 300 lines.**
   - This is the default expectation, not a suggestion.
   - Exceed it only when cohesion clearly benefits and the reason is defensible.
   - If a file grows too large, split by responsibility: handlers, services, validators, mappers, hooks, components, etc.

3. **Develop with DDD and best practices.**
   - Name things with domain language from the PRD.
   - Keep business rules inside the correct bounded context.
   - Let application services orchestrate across domains.
   - Keep infrastructure and framework code from leaking into the core model.

## Planned Architecture Direction

The PRD points toward this implementation shape:

```text
cmd/server/
internal/api/
internal/auth/
internal/config/
internal/db/
internal/models/
internal/provider/
internal/route/
internal/proxy/
internal/metrics/
internal/logs/
internal/scheduler/
web/
docker/
```

Treat those directories as architectural boundaries:
- `api/` and `web/` are interfaces
- `provider/`, `route/`, `models/` express core business language
- `db/`, `metrics/`, `scheduler/`, `docker/` are infrastructure/supporting concerns

## Implementation Rules

- Start from the PRD, then map work into the correct bounded context.
- Prefer small, composable modules.
- Avoid “misc”, “utils”, or “common” dumping grounds unless a shared abstraction is truly cross-cutting.
- Keep request/response transport models from becoming the domain model by accident.
- Avoid coupling frontend page structure directly to backend persistence structure.
- Update documentation whenever a structural decision changes.

## Suggested Layering

- **Domain**: entities, value objects, invariants, domain services
- **Application**: use cases, orchestration, transaction boundaries
- **Infrastructure**: DB access, external provider calls, background jobs, metrics, runtime wiring
- **Interface**: HTTP handlers, middleware, frontend pages/components, API clients

## Practical Workflow

Before implementing a feature:
- identify the bounded context
- identify invariants and entities
- decide whether the work belongs to domain, application, infrastructure, or interface
- sketch the file split so the 300-line rule is preserved early

During implementation:
- keep changes small and scoped
- prefer explicit names over clever abstractions
- add tests alongside business behavior once the test harness exists

After implementation:
- verify docs still match the code
- verify no module became an accidental cross-domain god object
- refactor early if file growth or layering drift appears

## Current Repo Facts

- `.editorconfig` exists and should be respected.
- `docs/` is currently empty.
- `ideas/prd.md` is the main architecture and scope reference.
- No build/test/lint commands are yet established in-repo; treat any future defaults as intentional additions, not existing reality.

## Future Defaults (when code exists)

These are intended likely defaults, not current guarantees:

```bash
# backend
go test ./...
go build ./cmd/server

# frontend
npm test
npm run build
```

If implementation chooses different tooling, document it immediately in this file and the root `AGENTS.md`.
