# ChainMind

AI-powered Solidity smart contract auditor. ChainMind parses a contract, chunks it by function, retrieves similar real-world exploit patterns from pgvector, asks an LLM to review each chunk, and returns a structured, severity-ranked audit report.

## Features

- Solidity parsing and function-level chunking (contract name, functions, state variables, imports, inheritance)
- Vector search over 21 curated exploit patterns (reentrancy, overflow, access control, flash loans, oracles, unchecked calls)
- Anthropic Claude and OpenAI providers with timeout, retry and automatic provider fallback
- Asynchronous audit jobs with pollable status and chunk-level progress
- Deduplication, severity sorting, confidence scoring and persisted findings
- Structured JSON API with documented error codes
- Next.js audit console with a Monaco Solidity editor, live progress and a full report viewer
- Keyless development mode using deterministic mock providers

## UI overview

The app has two routes:

| Route | Purpose |
|---|---|
| `/` | Product overview and the audit console (`#console`) |
| `/audit/[id]` | Live job progress and the final report |

### Landing page

![ChainMind landing page](docs/screenshots/landing.png)

### Audit console

Monaco editor with Solidity syntax highlighting, contract name input, depth selector, character counter and inline validation. Submitting redirects to the live audit page.

![ChainMind audit console](docs/screenshots/audit-console.png)

### Audit report

The audit page polls every 1.5 seconds, renders the Parse → Analyze → Report → Done timeline, and shows severity-ranked findings with vulnerable code, recommended fixes, matched exploit patterns and references.

![ChainMind audit report](docs/screenshots/report.png)

Responsive report on mobile:

<img src="docs/screenshots/mobile-report.png" alt="ChainMind audit report on mobile" width="360" />

Screenshots were captured in keyless mock mode. Live mode shows the configured Claude/OpenAI model and real embedding similarities.

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go 1.25, Gin, pgx v5, pgvector, zerolog, viper, testify |
| Database | PostgreSQL 16 with pgvector |
| LLM | Anthropic Messages API, OpenAI Chat Completions, OpenAI embeddings |
| Frontend | Next.js 14 App Router, TypeScript, Tailwind CSS, Zustand, Monaco, react-markdown |
| Local infra | Docker Compose |

## Quick start (no API keys required)

Prerequisites: Go 1.25+, Node.js 20+, npm, Docker Desktop.

```bash
cd chainmind
cp .env.example .env
```

Edit `.env` for the keyless demo:

```env
DEFAULT_LLM_PROVIDER=mock
RAG_SIMILARITY_THRESHOLD=0.5
```

Then:

```bash
make up                  # Postgres 16 + pgvector on localhost:5433
make seed                # 21 exploit patterns embedded with mock embeddings

cd frontend && npm install && cd ..   # once, installs Next.js/Monaco dependencies
cp frontend/.env.local.example frontend/.env.local

make run-backend         # http://localhost:8080 (separate terminal)
make run-frontend        # http://localhost:3000 (separate terminal)
```

Open http://localhost:3000, scroll to **Audit console**, paste a contract, and start the audit. The mock provider runs the complete pipeline so you can see real progress states, persistence and report rendering without spending tokens.

> The mock provider is a development aid that applies shallow heuristics. It is not a security analyzer and is rejected when `ENV=production`.

## Live provider mode

Set your keys and the real models in `.env`:

```env
DEFAULT_LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
ANTHROPIC_MODEL=claude-sonnet-4-20250514
OPENAI_MODEL=gpt-4o
EMBEDDING_MODEL=text-embedding-3-small
RAG_SIMILARITY_THRESHOLD=0.75
```

Re-run `make seed` so the exploit patterns are re-embedded with the real OpenAI embedding model. OpenAI is required for RAG; Anthropic alone still runs LLM analysis but skips retrieval with a warning in the report metadata.

Provider behavior:

- Primary provider: `DEFAULT_LLM_PROVIDER`
- Secondary provider: the other provider, used automatically when its key is configured
- Each attempt has a 60s timeout and one retry for transient failures (timeouts, 429, 5xx)

## Demo walkthrough

1. Open http://localhost:3000 and scroll to the console.
2. Contract name: `VulnerableVault`.
3. Paste the contents of `backend/testdata/vulnerable/reentrancy.sol`.
4. Keep depth on **Standard** and submit.
5. The audit page polls the job and advances through Parse → Analyze → Report → Done.
6. The report shows one critical reentrancy finding in `withdraw()` with the vulnerable code, a recommended fix and the matched exploit pattern.

Other fixtures:

| Contract | Path | Expected |
|---|---|---|
| Reentrancy | `backend/testdata/vulnerable/reentrancy.sol` | 1 critical reentrancy |
| Access control | `backend/testdata/vulnerable/access_control.sol` | 2 critical access-control findings |
| Safe vault | `backend/testdata/safe/safe_vault.sol` | no critical/high findings |

A captured mock-mode report is available in [`example_report.json`](./example_report.json).

## API

Base URL: `http://localhost:8080/api/v1`

### Submit an audit

```bash
jq -Rs '{contract_name: "VulnerableVault", source_code: ., analysis_depth: "standard"}' \
  backend/testdata/vulnerable/reentrancy.sol > /tmp/chainmind-payload.json

curl -s -X POST http://localhost:8080/api/v1/audit \
  -H 'Content-Type: application/json' \
  -d @/tmp/chainmind-payload.json
```

Response: `202 Accepted` with `audit_id`, `status` and `estimated_seconds`.

### Poll status

```bash
curl -s http://localhost:8080/api/v1/audit/<audit_id>
```

Statuses: `PENDING`, `PARSING`, `ANALYZING`, `GENERATING_REPORT`, `COMPLETED`, `FAILED`. Progress includes `total_chunks`, `analyzed_chunks` and `current_stage`.

### Get the report

```bash
curl -s http://localhost:8080/api/v1/audit/<audit_id>/report
```

Returns summary counts, findings, matched exploit patterns, references and metadata (model, depth, RAG patterns, processing time, warnings).

### Exploit catalogue

```bash
curl -s 'http://localhost:8080/api/v1/exploits?category=reentrancy&severity=critical&page=1&limit=20'
```

### Health

```bash
curl -s http://localhost:8080/health
curl -s http://localhost:8080/api/v1/health
```

### Errors

All errors are JSON:

```json
{ "error": "INVALID_INPUT", "message": "source_code is required" }
```

| Case | Status / code |
|---|---|
| Missing or invalid fields | 400 `INVALID_INPUT` |
| Unparseable Solidity | 422 `PARSE_ERROR` |
| Unknown audit | 404 `AUDIT_NOT_FOUND` |
| Report requested before completion | 409 `AUDIT_NOT_READY` |
| Database/service unavailable | 503 `SERVICE_UNAVAILABLE` |

## Architecture

```text
Browser (Next.js + Monaco)
        │ POST /audit
        ▼
Gin API ──► audit_jobs (PostgreSQL)
        │
        ▼
Orchestrator (in-process worker)
  1. parse + chunk
  2. embed chunk ──► pgvector top-k exploit patterns
  3. build prompt ──► Anthropic / OpenAI (retry + fallback)
  4. validate JSON ──► dedupe + score ──► findings table
  5. COMPLETED
        │
        ▼
GET /audit/:id + /report
```

Audits run in-process with bounded concurrency. Jobs left in a non-terminal state when the server stops are marked `FAILED` on the next startup. This matches the single-instance MVP; horizontal scaling would require an external queue.

## Configuration

| Variable | Purpose | Default |
|---|---|---|
| `PORT` | API port | `8080` |
| `ENV` | `development` enables mock providers | `development` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins | `http://localhost:3000` |
| `DATABASE_URL` | PostgreSQL DSN | required |
| `DEFAULT_LLM_PROVIDER` | `anthropic`, `openai`, or `mock` (dev only) | `anthropic` |
| `ANTHROPIC_API_KEY` / `OPENAI_API_KEY` | Provider keys | empty |
| `ANTHROPIC_MODEL` / `OPENAI_MODEL` | Analysis models | Claude Sonnet / GPT-4o |
| `EMBEDDING_MODEL` | OpenAI embedding model | `text-embedding-3-small` |
| `RAG_TOP_K` | Patterns retrieved per chunk | `5` |
| `RAG_SIMILARITY_THRESHOLD` | Minimum cosine similarity | `0.75` |

Frontend: copy `frontend/.env.local.example` to `frontend/.env.local` and set `NEXT_PUBLIC_API_URL` if the API is not on `localhost:8080`.

Secrets are only read from the environment or a git-ignored `.env`; `.env.example` contains empty keys.

## Database

Migrations are embedded in the backend and applied automatically at startup and by the seed script:

1. `001_create_exploit_patterns.sql` — pgvector extension, exploit catalogue, ivfflat index
2. `002_create_audit_jobs.sql` — audit status enum and jobs
3. `003_create_findings.sql` — findings with matched pattern metadata
4. `004_add_audit_metadata.sql` — `rag_patterns_used` and `warnings` for report metadata

The `references` column is quoted in SQL because `REFERENCES` is a reserved PostgreSQL keyword.

Docker Compose maps the database to host port **5433** so it does not collide with a local PostgreSQL on 5432.

## Testing

```bash
make test          # backend unit tests; DB integration tests skip without a DSN
```

`make lint` requires [golangci-lint](https://golangci-lint.run/) to be installed.

With the database running:

```bash
cd backend
TEST_DATABASE_URL='postgres://chainmind:chainmind@localhost:5433/chainmind?sslmode=disable' go test ./... -v
```

Coverage includes the parser/chunker, chunk context, mock and OpenAI embeddings, pgvector search, prompt building, strict LLM JSON validation, retry/fallback, deduplication/scoring, orchestrator lifecycle and failure modes, all HTTP handlers, and a full API integration test that submits the vulnerable and safe fixtures.

Frontend:

```bash
cd frontend
npm run build
npm run lint
```

## Project layout

```text
chainmind/
├── backend/
│   ├── cmd/server/           # entrypoint, graceful shutdown
│   ├── internal/
│   │   ├── parser/           # Solidity parser + chunker
│   │   ├── rag/              # embeddings, pgvector store, retriever
│   │   ├── llm/              # providers, retry/fallback, prompt, JSON parser
│   │   ├── orchestrator/     # async audit worker
│   │   ├── report/           # dedup, scoring, report assembly
│   │   ├── repository/       # pgx repositories + migrations
│   │   ├── api/              # router, middleware, handlers
│   │   ├── config/ dto/
│   │   └── ...
│   ├── scripts/seed_exploits.go
│   └── testdata/             # vulnerable + safe fixtures
├── data/exploits/            # 21 seed patterns
├── frontend/src/             # App Router pages, components, store
├── docs/screenshots/         # README screenshots
├── docker-compose.yml
├── example_report.json
└── PROJECT_SPEC.md
```

## Limitations

- Regex-based parsing, not a full `solc` AST; unusual syntax can be missed.
- The in-process worker assumes a single backend instance.
- Solidity only.
- Mock mode is a deterministic demo, not a security product.
- No authentication, billing, GitHub integration or automatic fixes, per the MVP scope.

## Troubleshooting

- **`database_connect_retry` then exit**: ensure `make up` finished and `DATABASE_URL` uses port 5433.
- **RAG warning in reports**: `OPENAI_API_KEY` is unset or the patterns were seeded without embeddings; set the key and run `make seed`.
- **Audits fail immediately**: the configured LLM provider key is missing; check `GET /api/v1/health`.
- **Browser cannot reach the API**: confirm `NEXT_PUBLIC_API_URL` in `frontend/.env.local` and that the backend is on port 8080.
- **Port 5433 already in use**: stop the conflicting container or change the compose port mapping and `DATABASE_URL` together.