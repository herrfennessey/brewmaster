# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Brewmaster is a PWA espresso assistant that takes coffee bean info (text, image, or URL) and returns AI-generated brew parameters. Go backend + React TypeScript frontend, deployed on Google Cloud Run. AI is **OpenAI only** — no Anthropic, no provider factory.

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
| `make docker/run` | Run image locally on :8080 (reads `OPENAI_API_KEY` from env) |
| `make clean` | Remove `brewmaster` binary and `api/static/` |

To run a single Go test package: `cd api && go test -race ./internal/handler/...`

### Local environment

Copy `.envrc.example` to `.envrc` and source it (or use direnv). Required vars: `OPENAI_API_KEY`, `AI_PROVIDER=openai`, `AI_MODEL`.

## Architecture

### Request flow

1. Vite dev server (:5173) proxies `/api/*` and `/health` to Go server (:8080).
2. In production, Go serves the React SPA from the embedded `./static` directory (Vite builds there). All unknown paths fall back to `index.html` for client-side routing.

### Backend (`api/`)

- `main.go` — server setup, graceful shutdown, timeouts.
- `internal/router/router.go` — route registration, SPA fallback, permissive CORS middleware.
- `internal/handler/` — one file per handler. Add new handlers here, register them in router.
- `internal/ai/` — OpenAI provider only. Three methods on `Provider`:
  - `Complete` — Chat Completions with forced tool call (structured JSON output)
  - `CompleteWithImage` — vision call with base64 image + forced tool call
  - `FindRoasterContent` — Responses API with `web_search` tool; returns synthesised plain text
- `internal/models/types.go` — shared data types (Go structs match TypeScript interfaces 1:1)

**AI provider**: `OpenAIProvider` in `internal/ai/openai.go`. Uses `github.com/openai/openai-go/v3`. Model from `AI_MODEL` env — never hardcode or change model names. Do not add AnthropicProvider or a provider factory.

**parse-bean handler** (`internal/handler/parse.go`) dispatches on `Content-Type`:
- `multipart/form-data` → `handleImage`: vision parse → web enrichment via `FindRoasterContent` → merge → `BeanProfile{source_type:"image+web"}` or fallback `"image"`
- JSON `input_type:"url"` → `handleURL`: server-side fetch + goquery HTML strip → `Complete`
- JSON `input_type:"text"` → `handleText`: `Complete` directly

### Frontend (`pwa/src/`)

React 19 + TypeScript. No component library. State is localStorage (Firestore is a later migration). Vite build outputs directly into `api/static/`, which gets embedded in the Go binary.

Home screen has three input tabs: **Text**, **Image** (drag-and-drop + click + paste), **URL**. BeanReview shows an enrichment badge when `source_type === "image+web"`.

### Infrastructure (`infra/`)

Terraform manages all GCP resources: Cloud Run, Artifact Registry, Secret Manager (`OPENAI_API_KEY`), and GitHub Actions workload identity federation. Backend state in GCS (`the-coffee-brewmaster` project, `brewmaster/terraform/state` prefix). Target region: `europe-west3`.

GCP project: `the-coffee-brewmaster`. Cloud Run SA: `brewmaster-api@the-coffee-brewmaster.iam.gserviceaccount.com`.

## Key Constraints

- **golangci-lint is strict** — functions ≤100 lines, cyclomatic complexity ≤15, errors must be wrapped (`wrapcheck`), no huge params by value. See `api/.golangci.yml` for full config. CI will fail on lint errors.
- **TypeScript strict mode** — `noUnusedLocals`, `noUnusedParameters` are errors.
- **OpenAI only** — do not add Anthropic support or a multi-provider factory.
- **No auth** — Cloud Run allows unauthenticated access by design. Personal tool.
- **Scale-to-zero** — the Cloud Run service has min instances = 0. Avoid cold-start-sensitive designs.
- **Static embedding** — `api/static/` is populated by the PWA build; never commit generated files there.
- **US spelling** — misspell linter uses locale:US. Use `flavor` not `flavour` everywhere.
- **Model selection** — never modify `AI_MODEL` env var references or model name constants; the user manages model selection.

## Implementation Status

- **Phase 1** (done): text input → `POST /api/parse-bean` → `BeanProfile`, `POST /api/generate-parameters` → `BrewParameters`, three screens (Home → BeanReview → BrewParameters)
- **Phase 2** (done): image upload (vision API + web enrichment), URL scraping (goquery), tabbed Home UI
- **Phase 3–5** (deferred): shot feedback loop, cross-bean learning, machine profiles, Firestore migration

Key data types (Go structs + TypeScript interfaces match 1:1):
- `BeanProfile`: `id`, `source_type` (`"text"` | `"image"` | `"url"` | `"image+web"`), `parsed` (ParsedBean), `confidence`, `created_at`
- `ParsedBean`: `producer`, `origin_country`, `origin_region`, `altitude_m`, `varietal`, `process`, `roast_level`, `roast_date`, `roaster_name`, `flavor_notes`, `lot_year`
- `BrewParameters`: `dose_g`, `yield_g`, `ratio`, `temp_c`, `time_s`, `preinfusion_s` — each with `value` + `range [2]float64`
