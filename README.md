# Brewmaster

A PWA espresso assistant. Give it a coffee bag — as text, a photo, or a roaster URL — and it returns dialed-in brew parameters using specialty coffee domain knowledge (roast level, altitude, varietal density, process, and freshness all factor in).

## How it works

1. **Input** — paste bag label text, upload a photo (drag-and-drop or paste from clipboard), or paste a roaster product URL
2. **Parse** — AI extracts origin, varietal, process, altitude, roast level, roast date, and flavor notes
3. **Enrich** — if a roaster name is recognised, the web is searched for the product page and any missing fields are filled in automatically
4. **Review** — confirm or edit the parsed fields before proceeding
5. **Brew parameters** — a deterministic rule engine computes dose, yield, ratio, temperature, time, and pre-infusion with confidence ranges. The LLM provides a short prose explanation but does not touch the numbers.

## Brew engine

The parameter calculation lives in `api/internal/brew/` and is fully deterministic — same input, same output. The temperature model is roast-primary (light ~94°C, medium-light ~93°C, medium ~92°C, dark ~90°C for espresso; +1°C for pourover) with small additive deltas for altitude, varietal density, process, and freshness. Output is clamped to the operating envelope (86–96°C) so adjustments can't compound out of range. Suitability for milk drinks is evaluated separately against a rule cascade. The current ruleset is `v2`.

## Stack

| Layer | Technology |
|-------|-----------|
| Frontend | React 19 + TypeScript, Vite, localStorage |
| Backend | Go, `net/http` |
| AI | OpenAI (vision + web search via Responses API) |
| Infrastructure | Google Cloud Run, Artifact Registry, Secret Manager, Terraform |

## Local development

Copy `.envrc.example` to `.envrc` and populate it, then run two terminals:

```bash
make dev/api   # Go API server on :8080
make dev/pwa   # Vite dev server on :5173 (proxies /api/* to :8080)
```

Open `http://localhost:5173`.

Required env vars: `OPENAI_API_KEY`, `AI_PROVIDER=openai`, `AI_MODEL`.

## Commands

| Command | Description |
|---------|-------------|
| `make dev/api` | Run Go API server locally |
| `make dev/pwa` | Run Vite dev server on :5173 |
| `make build` | Build PWA then Go binary |
| `make test` | Go tests with race detector |
| `make lint` | golangci-lint + ESLint |
| `make typecheck` | TypeScript type-check only |
| `make docker/build` | Build Docker image |
| `make docker/run` | Run image locally on :8080 |

## Project structure

```
api/          Go backend
  internal/
    ai/       Provider interface + OpenAI implementation (chat + vision + web search)
    brew/     Deterministic rule engine (parameter calculation, suitability, confidence)
    handler/  HTTP handlers (parse-bean, generate-parameters, health)
    models/   Shared data types
    router/   Route registration + SPA fallback
pwa/          React frontend
  src/
    screens/  Home, BeanReview, BrewParameters
    services/ API client, localStorage helpers
    types/    TypeScript interfaces mirroring Go structs
infra/        Terraform (Cloud Run, Artifact Registry, Secret Manager, WIF)
docs/         Implementation plan
```

## Input modes

**Text** — paste anything from the bag: origin info, tasting notes, roast date.

**Image** — upload or paste (Cmd+V) a JPEG/PNG/WEBP photo of the bag label. The vision model extracts all readable fields, then automatically searches the roaster's website to fill in anything the image didn't show (altitude, varietal, process, flavor notes). The BeanReview screen indicates when web data was used.

**URL** — paste a roaster product page URL. The page is fetched server-side, stripped of navigation/scripts, and the cleaned text is passed to the AI.

## Deployment

Infrastructure is managed by Terraform in `infra/`. CI/CD runs on GitHub Actions with keyless GCP auth via Workload Identity Federation. Deployments trigger automatically on push to `main`.

GCP project: `the-coffee-brewmaster` — region: `europe-west3`.
