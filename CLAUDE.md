# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Brewmaster is a PWA espresso assistant that takes coffee bean info (text, image, URL) and returns AI-generated brew parameters. Go backend + React TypeScript frontend, deployed on Google Cloud Run. AI is always Anthropic or OpenAI, switchable via `AI_PROVIDER` env var.

## Development Commands

| Target | Description |
|--------|-------------|
| `make dev/api` | Run Go API server locally |
| `make dev/pwa` | Run Vite dev server on :5173 |
| `make install/pwa` | `npm ci` for the PWA |
| `make build` | Build PWA then Go binary (runs `build/pwa` + `build/api`) |
| `make build/api` | Compile Go binary only |
| `make build/pwa` | Build PWA into `api/static/` only |
| `make test` | Go tests with race detector |
| `make lint` | golangci-lint + ESLint |
| `make typecheck` | TypeScript type-check only (no emit) |
| `make docker/build` | Build Docker image (pass `TAG=x` to override, default `latest`) |
| `make docker/run` | Run image locally on :8080 (reads `ANTHROPIC_API_KEY` from env) |
| `make clean` | Remove `brewmaster` binary and `api/static/` |

To run a single Go test package: `cd api && go test -race ./internal/handler/...`

### Local environment

Copy `.envrc.example` to `.envrc` and source it (or use direnv). Required vars: `ANTHROPIC_API_KEY`, `AI_PROVIDER`, `AI_MODEL`.

## Architecture

### Request flow

1. Vite dev server (:5173) proxies `/api/*` and `/health` to Go server (:8080).
2. In production, Go serves the React SPA from the embedded `./static` directory (Vite builds there). All unknown paths fall back to `index.html` for client-side routing.

### Backend (`api/`)

- `main.go` — server setup, graceful shutdown, timeouts.
- `internal/router/router.go` — route registration, SPA fallback, permissive CORS middleware.
- `internal/handler/` — one file per handler. Add new handlers here, register them in router.

AI calls are driven by env: `AI_PROVIDER` (anthropic|openai), `AI_MODEL`, and the respective `*_API_KEY`. The planned AI layer is provider-agnostic — keep AI abstractions in their own package.

### Frontend (`pwa/src/`)

React 19 + TypeScript 6. No component library yet. State is localStorage (Phase 1; Firestore is a later migration). Vite build outputs directly into `api/static/`, which gets embedded in the Go binary.

### Infrastructure (`infra/`)

Terraform manages all GCP resources: Cloud Run, Artifact Registry, Secret Manager (API keys), and GitHub Actions workload identity federation. Backend state in GCS (`the-coffee-brewmaster` project, `brewmaster/terraform/state` prefix). Target region: `europe-west3`.

GCP project: `the-coffee-brewmaster`. Cloud Run SA: `brewmaster-api@the-coffee-brewmaster.iam.gserviceaccount.com`.

## Key Constraints

- **golangci-lint is strict** — functions ≤100 lines, cyclomatic complexity ≤15, errors must be wrapped (`wrapcheck`). See `api/.golangci.yml` for full config. CI will fail on lint errors.
- **TypeScript strict mode** — `noUnusedLocals`, `noUnusedParameters` are errors.
- **No auth in Phase 1** — Cloud Run allows unauthenticated access by design.
- **Scale-to-zero** — the Cloud Run service has min instances = 0. Avoid cold-start-sensitive designs.
- **Static embedding** — `api/static/` is populated by the PWA build; never commit generated files there.

## Implementation Roadmap

See `docs/plan.md` for the full phased plan. Phase 1 focus:
- `POST /api/parse-bean` → `BeanProfile`
- `POST /api/generate-parameters` → `BrewParameters`
- Three frontend screens: Home → Bean Review → Brew Parameters

Key data types (Go structs + TypeScript interfaces should match):
- `BeanProfile`: producer, origin_country, altitude_m, varietal, process, roast_level, flavour_notes, confidence
- `BrewParameters`: dose_g, yield_g, ratio, temp_c, time_s, preinfusion_s — each with value + range