# PROJECT KNOWLEDGE BASE

**Generated:** 2026-05-01
**Branch:** dev
**Commit:** 5f03cd6

## OVERVIEW

This repository is pre-implementation. The active source of truth is `ideas/prd.md`, which defines a lightweight internal model-control plane with a Go backend and React frontend.

Work here should preserve three hard constraints:
- keep the project structure engineered and explicit
- keep most code files under 300 lines unless there is a clear reason not to
- implement with DDD boundaries and best practices from day one

## CURRENT STRUCTURE

```text
LumenRoute/
├── docs/           # currently empty
├── ideas/          # PRD and future design docs
├── .editorconfig   # baseline formatting rules
├── .gitattributes
└── .gitignore
```

## WHERE TO LOOK

| Task | Location | Notes |
|---|---|---|
| Understand product scope | `ideas/prd.md` | Primary source of truth right now |
| Check formatting defaults | `.editorconfig` | Spaces by default, tabs for Go/Makefile |
| Add design docs | `ideas/` | Keep implementation decisions traceable to PRD |
| Add long-form docs later | `docs/` | Use once implementation begins |

## INTENDED IMPLEMENTATION SHAPE

The PRD proposes this layout:

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

Treat those directories as future bounded contexts / infrastructure boundaries, not as a dumping ground.

## DDD CONVENTIONS

- Model domain language explicitly: provider, route, route target, API key, request log, health state.
- Keep business rules near the domain they protect.
- Use application services to orchestrate workflows across domains.
- Keep transport, persistence, and infrastructure concerns outside domain logic.
- Prefer small packages/modules with one clear responsibility.
- If a file grows past ~300 lines, split by responsibility unless cohesion would clearly suffer.

## ENGINEERING RULES

- Do not invent runtime/build/test commands as if they already exist.
- Do not treat planned directories from the PRD as implemented code.
- Add new top-level directories only with a documented rationale.
- Keep docs in sync with architecture decisions.
- Prefer explicit boundaries over convenience shortcuts.

## ANTI-PATTERNS (THIS PROJECT)

- dumping unrelated code into `internal/` without domain ownership
- mixing HTTP/database concerns directly into domain logic
- giant files for handlers/services/components when they can be split cleanly
- letting frontend models drift away from backend domain language
- adding infra/tooling that contradicts the lightweight first milestone in the PRD
- implementing features outside PRD scope without updating the design docs first

## UNIQUE STYLES

- This repo is currently doc-first: design quality matters before code exists.
- The PRD is detailed enough to drive initial folder layout and bounded contexts.
- `ideas/` is not scratch space; it is controlled architecture input.

## EXPECTED FUTURE DEFAULTS

These are intended defaults once code exists, not confirmed current commands:

```bash
# backend
go test ./...
go build ./cmd/server

# frontend
npm test
npm run build
```

If actual tooling differs, update this file and `CLAUDE.md` together.

## NOTES

- `docs/` is empty today; root guidance is enough there for now.
- `ideas/` is distinct enough to carry its own local `AGENTS.md`.
- Prefer shallow documentation hierarchy until real implementation complexity appears.


<claude-mem-context>
# Memory Context

# [LumenRoute] recent context, 2026-05-29 5:00pm GMT+8

No previous sessions found.
</claude-mem-context>