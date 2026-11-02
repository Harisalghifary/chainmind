# PROJECT_SPEC.md — ChainMind

## 0. Instructions for AI Coding Assistant

Read this entire file before writing any code.

You are building **ChainMind**, an AI-powered Solidity smart contract auditor.

Rules:

1. Implement phase by phase in the order defined in this document.
2. Do not add features that are not specified here.
3. Do not change the tech stack.
4. Do not skip error handling.
5. Do not hardcode secrets.
6. Use environment variables for all API keys.
7. Prefer simple, working code over premature optimization.
8. Every API endpoint must return structured JSON.
9. Every backend module must have basic tests.
10. Stop after each phase if the user asks for review.

---

## 1. Product Overview

### 1.1 What ChainMind Does

ChainMind accepts Solidity smart contract source code and analyzes it for common security vulnerabilities using:

1. Solidity parsing and function-level chunking
2. RAG-based retrieval of known exploit patterns
3. LLM-powered security analysis
4. Structured vulnerability report generation

### 1.2 Core Pipeline

```text
Solidity Input
    ↓
Parse Contract
    ↓
Chunk by Function
    ↓
Retrieve Similar Exploit Patterns from Vector DB
    ↓
Send Code + Exploit Context to LLM
    ↓
Parse Structured LLM Response
    ↓
Deduplicate + Score Findings
    ↓
Generate Audit Report
```

### 1.3 Vulnerability Classes

ChainMind MVP must detect at least these vulnerability classes:

| Vulnerability | Severity | Example |
|---|---:|---|
| Reentrancy | Critical | DAO hack |
| Integer overflow / underflow | High | Token balance manipulation |
| Access control misconfiguration | Critical | Unprotected admin functions |
| Flash loan attack vector | High | Price manipulation |
| Oracle manipulation | High | Single-source price feed abuse |
| Unchecked external call | Medium | Ignored call failure |

### 1.4 Non-Goals

Do NOT build:

- User authentication
- Billing or subscription
- Multi-tenancy
- Mobile app
- Fine-tuning ML models
- Support for Vyper, Rust, Move, or other languages
- Real-time collaboration
- Automatic fixing of contracts
- GitHub integration in MVP

---

## 2. Tech Stack

### 2.1 Backend

| Component | Technology |
|---|---|
| Language | Go 1.22+ |
| HTTP framework | Gin |
| Database | PostgreSQL 16 |
| Vector search | pgvector |
| DB driver | pgx v5 |
| Logging | zerolog |
| Config | viper |
| Testing | testify |
| LLM providers | Anthropic Claude and OpenAI |
| Embedding model | OpenAI text-embedding-3-small |
| Containerization | Docker Compose |

### 2.2 Frontend

| Component | Technology |
|---|---|
| Framework | Next.js 14+ App Router |
| Language | TypeScript |
| Styling | Tailwind CSS |
| State management | Zustand |
| Code editor | Monaco Editor |
| Markdown rendering | react-markdown |
| HTTP client | native fetch |

---

## 3. Repository Structure

Create exactly this structure:

```text
chainmind/
├── PROJECT_SPEC.md
├── docker-compose.yml
├── .env.example
├── .gitignore
├── Makefile
│
├── backend/
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   │
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go
│   │   │
│   │   ├── api/
│   │   │   ├── router.go
│   │   │   ├── middleware.go
│   │   │   └── handlers/
│   │   │       ├── health.go
│   │   │       ├── audit.go
│   │   │       ├── report.go
│   │   │       └── exploits.go
│   │   │
│   │   ├── dto/
│   │   │   ├── request.go
│   │   │   └── response.go
│   │   │
│   │   ├── parser/
│   │   │   ├── parser.go
│   │   │   ├── chunker.go
│   │   │   └── types.go
│   │   │
│   │   ├── rag/
│   │   │   ├── embedder.go
│   │   │   ├── retriever.go
│   │   │   ├── store.go
│   │   │   └── types.go
│   │   │
│   │   ├── llm/
│   │   │   ├── client.go
│   │   │   ├── anthropic.go
│   │   │   ├── openai.go
│   │   │   ├── prompt.go
│   │   │   ├── parser.go
│   │   │   └── types.go
│   │   │
│   │   ├── orchestrator/
│   │   │   ├── orchestrator.go
│   │   │   └── types.go
│   │   │
│   │   ├── report/
│   │   │   ├── generator.go
│   │   │   ├── dedup.go
│   │   │   ├── scorer.go
│   │   │   └── types.go
│   │   │
│   │   └── repository/
│   │       ├── db.go
│   │       ├── audit_repo.go
│   │       ├── exploit_repo.go
│   │       └── migrations/
│   │           ├── 001_create_exploit_patterns.sql
│   │           ├── 002_create_audit_jobs.sql
│   │           └── 003_create_findings.sql
│   │
│   ├── pkg/
│   │   ├── errors/
│   │   │   └── errors.go
│   │   └── validator/
│   │       └── solidity.go
│   │
│   ├── scripts/
│   │   └── seed_exploits.go
│   │
│   └── testdata/
│       ├── vulnerable/
│       │   ├── reentrancy.sol
│       │   ├── overflow.sol
│       │   └── access_control.sol
│       └── safe/
│           └── safe_vault.sol
│
├── frontend/
│   ├── package.json
│   ├── next.config.js
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── .env.local.example
│   └── src/
│       ├── app/
│       │   ├── layout.tsx
│       │   ├── page.tsx
│       │   ├── globals.css
│       │   └── audit/
│       │       └── [id]/
│       │           └── page.tsx
│       ├── components/
│       │   ├── ContractInput.tsx
│       │   ├── AuditProgress.tsx
│       │   ├── ReportViewer.tsx
│       │   ├── FindingCard.tsx
│       │   └── SeverityBadge.tsx
│       ├── lib/
│       │   ├── api.ts
│       │   ├── types.ts
│       │   └── utils.ts
│       └── store/
│           └── useAuditStore.ts
│
└── data/
    └── exploits/
        ├── reentrancy.json
        ├── overflow.json
        ├── access_control.json
        ├── flash_loan.json
        ├── oracle.json
        └── unchecked_calls.json
```

---

## 4. Functional Requirements

### 4.1 Contract Submission

The system must allow a user to submit:

```json
{
  "contract_name": "VulnerableVault",
  "source_code": "pragma solidity ^0.8.0; ...",
  "solidity_version": "0.8.0",
  "analysis_depth": "standard"
}
```

Fields:

| Field | Required | Notes |
|---|---:|---|
| contract_name | Yes | String, max 255 chars |
| source_code | Yes | String, max 100,000 chars |
| solidity_version | No | String |
| analysis_depth | No | quick, standard, deep. Default: standard |

### 4.2 Audit Processing

The system must process audits asynchronously.

Statuses:

```text
PENDING
PARSING
ANALYZING
GENERATING_REPORT
COMPLETED
FAILED
```

### 4.3 Audit Progress

The system must expose progress:

```json
{
  "audit_id": "uuid",
  "status": "ANALYZING",
  "progress": {
    "total_chunks": 5,
    "analyzed_chunks": 2,
    "current_stage": "analyzing"
  }
}
```

### 4.4 Audit Report

The system must return a structured report containing:

- Summary counts by severity
- List of findings
- Metadata about model, analysis depth, and processing time
- Matched exploit pattern for each finding when available

### 4.5 Health Check

The system must expose:

```text
GET /health
GET /api/v1/health
```

Health response must include service status for:

- API
- Database
- Vector DB
- LLM provider

---

## 5. API Specification

Base URL:

```text
http://localhost:8080/api/v1
```

All responses must be JSON.

---

### 5.1 Submit Audit

```text
POST /api/v1/audit
```

Request:

```json
{
  "contract_name": "VulnerableVault",
  "source_code": "pragma solidity ^0.8.0;",
  "solidity_version": "0.8.0",
  "analysis_depth": "standard"
}
```

Success response: `202 Accepted`

```json
{
  "audit_id": "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
  "status": "PENDING",
  "estimated_seconds": 30,
  "created_at": "2026-10-30T00:00:00Z"
}
```

Error responses:

```json
{
  "error": "INVALID_INPUT",
  "message": "source_code is required"
}
```

```json
{
  "error": "PARSE_ERROR",
  "message": "Unable to parse Solidity contract"
}
```

---

### 5.2 Get Audit Status

```text
GET /api/v1/audit/:id
```

Success response: `200 OK`

```json
{
  "audit_id": "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
  "status": "ANALYZING",
  "progress": {
    "total_chunks": 5,
    "analyzed_chunks": 2,
    "current_stage": "analyzing"
  },
  "error_message": null,
  "created_at": "2026-10-30T00:00:00Z",
  "updated_at": "2026-10-30T00:00:10Z",
  "completed_at": null
}
```

---

### 5.3 Get Audit Report

```text
GET /api/v1/audit/:id/report
```

Success response: `200 OK`

```json
{
  "audit_id": "8f14e45f-ceea-467f-9d3a-8f7b2c1d9e42",
  "contract_name": "VulnerableVault",
  "summary": {
    "total_findings": 1,
    "critical": 1,
    "high": 0,
    "medium": 0,
    "low": 0,
    "informational": 0,
    "confidence_score": 0.93
  },
  "findings": [
    {
      "id": "f2d1c9e4-8f7b-4c1d-9e42-8f14e45fceea",
      "vulnerability_type": "reentrancy",
      "severity": "critical",
      "confidence": 0.95,
      "affected_function": "withdraw",
      "line_start": 12,
      "line_end": 16,
      "title": "Reentrancy vulnerability in withdraw()",
      "description": "The function calls an external contract before updating the user's balance.",
      "vulnerable_code": "function withdraw() external { ... }",
      "recommended_fix": "Update state before making the external call.",
      "fixed_code": "function withdraw() external nonReentrant { ... }",
      "matched_exploit_pattern": {
        "name": "Reentrancy via external call before state update",
        "real_world_case": "The DAO Hack, 2016",
        "similarity_score": 0.94
      },
      "references": [
        "https://swcregistry.io/docs/SWC-107"
      ]
    }
  ],
  "metadata": {
    "llm_model": "claude-sonnet-4-20250514",
    "analysis_depth": "standard",
    "rag_patterns_used": 5,
    "processing_time_ms": 28400
  }
}
```

---

### 5.4 List Exploit Patterns

```text
GET /api/v1/exploits
```

Query params:

```text
category
severity
page
limit
```

Success response:

```json
{
  "patterns": [
    {
      "id": "uuid",
      "name": "Reentrancy via external call before state update",
      "category": "reentrancy",
      "swc_id": "SWC-107",
      "severity": "critical",
      "description": "Attacker re-enters function before state update completes.",
      "real_world_case": "The DAO Hack, 2016"
    }
  ],
  "total": 21,
  "page": 1,
  "limit": 20
}
```

---

### 5.5 Health Check

```text
GET /api/v1/health
```

Success response:

```json
{
  "status": "healthy",
  "version": "0.1.0",
  "services": {
    "api": "up",
    "database": "up",
    "vector_db": "up",
    "llm_api": "up"
  }
}
```

---

## 6. Database Schema

Use PostgreSQL with pgvector.

### 6.1 Exploit Patterns

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE exploit_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    swc_id TEXT,
    severity TEXT NOT NULL CHECK (
        severity IN ('critical', 'high', 'medium', 'low', 'informational')
    ),
    description TEXT NOT NULL,
    vulnerable_code TEXT NOT NULL,
    fixed_code TEXT,
    real_world_case TEXT,
    references TEXT[],
    embedding vector(1536),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_exploit_embedding
ON exploit_patterns
USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

CREATE INDEX idx_exploit_category
ON exploit_patterns (category);

CREATE INDEX idx_exploit_severity
ON exploit_patterns (severity);
```

---

### 6.2 Audit Jobs

```sql
CREATE TYPE audit_status AS ENUM (
    'PENDING',
    'PARSING',
    'ANALYZING',
    'GENERATING_REPORT',
    'COMPLETED',
    'FAILED'
);

CREATE TABLE audit_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_name TEXT NOT NULL,
    source_code TEXT NOT NULL,
    solidity_version TEXT,
    analysis_depth TEXT NOT NULL DEFAULT 'standard' CHECK (
        analysis_depth IN ('quick', 'standard', 'deep')
    ),
    status audit_status NOT NULL DEFAULT 'PENDING',
    total_chunks INTEGER NOT NULL DEFAULT 0,
    analyzed_chunks INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    llm_model TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_audit_jobs_status
ON audit_jobs (status);

CREATE INDEX idx_audit_jobs_created_at
ON audit_jobs (created_at DESC);
```

---

### 6.3 Findings

```sql
CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id UUID NOT NULL REFERENCES audit_jobs(id) ON DELETE CASCADE,
    vulnerability_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (
        severity IN ('critical', 'high', 'medium', 'low', 'informational')
    ),
    confidence REAL NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    affected_function TEXT NOT NULL,
    line_start INTEGER,
    line_end INTEGER,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    vulnerable_code TEXT,
    recommended_fix TEXT,
    fixed_code TEXT,
    matched_pattern_id UUID REFERENCES exploit_patterns(id),
    matched_pattern_name TEXT,
    similarity_score REAL,
    references TEXT[],
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_findings_audit_id
ON findings (audit_id);

CREATE INDEX idx_findings_severity
ON findings (severity);
```

---

## 7. Backend Domain Types

### 7.1 Parser Types

```go
package parser

type ParsedContract struct {
    Name            string        `json:"name"`
    SolidityVersion string        `json:"solidity_version"`
    Functions       []FunctionDef `json:"functions"`
    StateVariables  []StateVar    `json:"state_variables"`
    Imports         []string      `json:"imports"`
    Inheritance     []string      `json:"inheritance"`
}

type FunctionDef struct {
    Name       string `json:"name"`
    Visibility string `json:"visibility"`
    Mutability string `json:"mutability"`
    Params     string `json:"params"`
    Body       string `json:"body"`
    LineStart  int    `json:"line_start"`
    LineEnd    int    `json:"line_end"`
}

type StateVar struct {
    Name       string `json:"name"`
    Type       string `json:"type"`
    Visibility string `json:"visibility"`
}

type CodeChunk struct {
    ContractName   string  `json:"contract_name"`
    FunctionName   string  `json:"function_name"`
    Code           string  `json:"code"`
    ContextSummary string  `json:"context_summary"`
    LineStart      int     `json:"line_start"`
    LineEnd        int     `json:"line_end"`
    Embedding      []float32 `json:"-"`
}
```

---

### 7.2 RAG Types

```go
package rag

type ExploitPattern struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Category       string   `json:"category"`
    SWCID          string   `json:"swc_id"`
    Severity       string   `json:"severity"`
    Description    string   `json:"description"`
    VulnerableCode string   `json:"vulnerable_code"`
    FixedCode      string   `json:"fixed_code"`
    RealWorldCase  string   `json:"real_world_case"`
    References     []string `json:"references"`
}

type RetrievalResult struct {
    Pattern         ExploitPattern `json:"pattern"`
    SimilarityScore float32        `json:"similarity_score"`
}
```

---

### 7.3 LLM Types

```go
package llm

type AnalysisRequest struct {
    Chunk   CodeChunk         `json:"chunk"`
    Patterns []RetrievalResult `json:"patterns"`
    Depth   string            `json:"depth"`
}

type LLMFinding struct {
    VulnerabilityType string  `json:"vulnerability_type"`
    Severity          string  `json:"severity"`
    Confidence        float32 `json:"confidence"`
    AffectedFunction  string  `json:"affected_function"`
    LineStart         int     `json:"line_start"`
    LineEnd           int     `json:"line_end"`
    Title             string  `json:"title"`
    Description       string  `json:"description"`
    VulnerableCode    string  `json:"vulnerable_code"`
    RecommendedFix    string  `json:"recommended_fix"`
    FixedCode         string  `json:"fixed_code"`
    MatchedPattern    string  `json:"matched_exploit_pattern"`
}

type AnalysisResponse struct {
    Findings   []LLMFinding `json:"findings"`
    Model      string       `json:"model"`
    TokensUsed int          `json:"tokens_used"`
}
```

---

### 7.4 Report Types

```go
package report

type AuditReport struct {
    AuditID      string    `json:"audit_id"`
    ContractName string    `json:"contract_name"`
    Summary      Summary   `json:"summary"`
    Findings     []Finding `json:"findings"`
    Metadata     Metadata  `json:"metadata"`
}

type Summary struct {
    TotalFindings   int     `json:"total_findings"`
    Critical        int     `json:"critical"`
    High            int     `json:"high"`
    Medium          int     `json:"medium"`
    Low             int     `json:"low"`
    Informational   int     `json:"informational"`
    ConfidenceScore float32 `json:"confidence_score"`
}

type Finding struct {
    ID                string          `json:"id"`
    VulnerabilityType string          `json:"vulnerability_type"`
    Severity          string          `json:"severity"`
    Confidence        float32         `json:"confidence"`
    AffectedFunction  string          `json:"affected_function"`
    LineStart         int             `json:"line_start"`
    LineEnd           int             `json:"line_end"`
    Title             string          `json:"title"`
    Description       string          `json:"description"`
    VulnerableCode    string          `json:"vulnerable_code"`
    RecommendedFix    string          `json:"recommended_fix"`
    FixedCode         string          `json:"fixed_code"`
    MatchedPattern    *MatchedPattern `json:"matched_exploit_pattern,omitempty"`
    References        []string        `json:"references"`
}

type MatchedPattern struct {
    Name            string  `json:"name"`
    RealWorldCase   string  `json:"real_world_case"`
    SimilarityScore float32 `json:"similarity_score"`
}

type Metadata struct {
    LLMModel         string `json:"llm_model"`
    AnalysisDepth    string `json:"analysis_depth"`
    RAGPatternsUsed  int    `json:"rag_patterns_used"`
    ProcessingTimeMs int64  `json:"processing_time_ms"`
}
```

---

## 8. Core Algorithms

### 8.1 Chunking Algorithm

Chunk Solidity code at function level.

Logic:

```text
1. Extract contract name.
2. Extract state variables.
3. Extract inheritance.
4. Extract each function body.
5. For each function, create a CodeChunk.
6. Add contract-level context summary to each chunk.
7. If a function body is larger than 4000 characters, split it into smaller logical blocks.
```

Context summary example:

```text
Contract state variables:
- mapping(address => uint256) balances (public)
- address owner (private)

Inherits from:
- ReentrancyGuard
```

---

### 8.2 RAG Retrieval Algorithm

```text
1. Take a CodeChunk.
2. Combine chunk.Code and chunk.ContextSummary.
3. Generate embedding using OpenAI text-embedding-3-small.
4. Query pgvector for top 5 similar exploit_patterns.
5. Use cosine similarity.
6. Filter results with similarity_score >= 0.75.
7. Return top results.
```

Default settings:

```text
top_k = 5
similarity_threshold = 0.75
```

---

### 8.3 LLM Prompt Strategy

System prompt must instruct the LLM to act as a smart contract security auditor.

Requirements:

1. Only report vulnerabilities visible in the provided code.
2. Do not speculate.
3. Use retrieved exploit patterns as reference.
4. Return strict JSON only.
5. Include severity, confidence, affected function, line numbers, and fix.
6. Return empty findings array if no issue is found.

---

### 8.4 LLM Output Schema

The LLM must return:

```json
{
  "findings": [
    {
      "vulnerability_type": "reentrancy",
      "severity": "critical",
      "confidence": 0.95,
      "affected_function": "withdraw",
      "line_start": 12,
      "line_end": 16,
      "title": "Reentrancy vulnerability in withdraw()",
      "description": "External call is made before state update.",
      "vulnerable_code": "function withdraw() external { ... }",
      "recommended_fix": "Update balance before external call.",
      "fixed_code": "function withdraw() external nonReentrant { ... }",
      "matched_exploit_pattern": "Reentrancy via external call before state update"
    }
  ]
}
```

If no findings:

```json
{
  "findings": []
}
```

---

## 9. Prompt Template

Use this as the base prompt.

System prompt:

```text
You are a senior smart contract security auditor.

Your task is to analyze Solidity code for security vulnerabilities.

Rules:
1. Only report vulnerabilities that are visible in the provided code.
2. Do not speculate about missing code.
3. Cross-reference the provided known exploit patterns.
4. Return only valid JSON.
5. If no vulnerability is found, return an empty findings array.
6. For each finding, include severity, confidence, affected function, line range, description, and recommended fix.
7. Be conservative. Avoid false positives.
```

User prompt:

```text
## Contract Code

Contract Name: {{contract_name}}
Function Name: {{function_name}}

{{code}}

## Contract Context

{{context_summary}}

## Known Exploit Patterns

{{exploit_patterns}}

## Required Output

Return JSON using this schema:

{
  "findings": [
    {
      "vulnerability_type": "string",
      "severity": "critical|high|medium|low|informational",
      "confidence": 0.0,
      "affected_function": "string",
      "line_start": 0,
      "line_end": 0,
      "title": "string",
      "description": "string",
      "vulnerable_code": "string",
      "recommended_fix": "string",
      "fixed_code": "string",
      "matched_exploit_pattern": "string|null"
    }
  ]
}
```

---

## 10. Exploit Pattern Seed Data

Seed data must be stored in:

```text
data/exploits/
```

Each JSON file must be an array of exploit patterns.

Example:

```json
[
  {
    "name": "Reentrancy via external call before state update",
    "category": "reentrancy",
    "swc_id": "SWC-107",
    "severity": "critical",
    "description": "An attacker can repeatedly call a function before the contract updates its internal balance.",
    "vulnerable_code": "function withdraw() external {\n    uint256 amount = balances[msg.sender];\n    (bool success, ) = msg.sender.call{value: amount}(\"\");\n    require(success);\n    balances[msg.sender] = 0;\n}",
    "fixed_code": "function withdraw() external nonReentrant {\n    uint256 amount = balances[msg.sender];\n    balances[msg.sender] = 0;\n    (bool success, ) = msg.sender.call{value: amount}(\"\");\n    require(success);\n}",
    "real_world_case": "The DAO Hack, 2016",
    "references": [
      "https://swcregistry.io/docs/SWC-107"
    ]
  }
]
```

Minimum seed requirements:

| File | Minimum Patterns |
|---|---:|
| reentrancy.json | 5 |
| overflow.json | 3 |
| access_control.json | 4 |
| flash_loan.json | 3 |
| oracle.json | 3 |
| unchecked_calls.json | 3 |

Total minimum: 21 exploit patterns.

---

## 11. Environment Variables

Create `.env.example`:

```env
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgres://chainmind:chainmind@localhost:5432/chainmind?sslmode=disable

# LLM Providers
DEFAULT_LLM_PROVIDER=anthropic
ANTHROPIC_API_KEY=your_anthropic_api_key
OPENAI_API_KEY=your_openai_api_key

# LLM Models
ANTHROPIC_MODEL=claude-sonnet-4-20250514
OPENAI_MODEL=gpt-4o
EMBEDDING_MODEL=text-embedding-3-small

# RAG
RAG_TOP_K=5
RAG_SIMILARITY_THRESHOLD=0.75
```

Frontend `.env.local.example`:

```env
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
```

---

## 12. Docker Compose

Use PostgreSQL with pgvector.

```yaml
version: "3.9"

services:
  postgres:
    image: pgvector/pgvector:pg16
    container_name: chainmind-postgres
    environment:
      POSTGRES_USER: chainmind
      POSTGRES_PASSWORD: chainmind
      POSTGRES_DB: chainmind
    ports:
      - "5432:5432"
    volumes:
      - chainmind_pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U chainmind -d chainmind"]
      interval: 5s
      timeout: 5s
      retries: 10

volumes:
  chainmind_pgdata:
```

---

## 13. Makefile

```makefile
.PHONY: up down run-backend run-frontend seed test lint build

up:
	docker-compose up -d

down:
	docker-compose down

run-backend:
	cd backend && go run cmd/server/main.go

run-frontend:
	cd frontend && npm run dev

seed:
	cd backend && go run scripts/seed_exploits.go

test:
	cd backend && go test ./... -v -cover

lint:
	cd backend && golangci-lint run ./...

build:
	cd frontend && npm run build
```

---

## 14. Implementation Phases

### Phase 1 — Project Skeleton

Goal: runnable backend, frontend, and database.

Tasks:

1. Create repository structure.
2. Create `docker-compose.yml`.
3. Create `.env.example`.
4. Create Makefile.
5. Initialize Go module.
6. Create Gin server.
7. Add `/health` endpoint.
8. Add `/api/v1/health` endpoint.
9. Connect to PostgreSQL.
10. Initialize Next.js frontend.
11. Add basic landing page.

Acceptance criteria:

- `docker-compose up -d` works.
- `make run-backend` starts API on port 8080.
- `GET /health` returns 200.
- `GET /api/v1/health` returns service status.
- `make run-frontend` starts frontend on port 3000.

---

### Phase 2 — Database Schema

Goal: create all required tables.

Tasks:

1. Add migration files.
2. Create `exploit_patterns` table.
3. Create `audit_jobs` table.
4. Create `findings` table.
5. Enable pgvector extension.
6. Add repository layer for DB access.

Acceptance criteria:

- Migrations run successfully.
- Tables exist.
- Vector column is created.
- Repository can insert and query exploit patterns.

---

### Phase 3 — Solidity Parser and Chunker

Goal: parse Solidity source into function-level chunks.

Tasks:

1. Create parser types.
2. Parse contract name.
3. Parse functions.
4. Parse state variables.
5. Parse imports.
6. Parse inheritance.
7. Create chunker.
8. Add test contracts.

Acceptance criteria:

- Parser extracts functions from simple Solidity contracts.
- Chunker produces one chunk per function.
- Each chunk includes contract context.
- Tests pass for vulnerable and safe contracts.

MVP note:

Regex-based parsing is acceptable for MVP. Full solc AST parsing is a stretch goal.

---

### Phase 4 — RAG Pipeline

Goal: retrieve relevant exploit patterns.

Tasks:

1. Create embedding service using OpenAI.
2. Create pgvector store.
3. Create exploit seed script.
4. Seed at least 21 exploit patterns.
5. Implement similarity search.
6. Return top 5 patterns with score >= 0.75.

Acceptance criteria:

- Seed script inserts patterns into database.
- Embeddings are stored.
- Similarity search returns relevant patterns.
- Reentrancy code retrieves reentrancy patterns first.

---

### Phase 5 — LLM Integration

Goal: send enriched code chunks to LLM and parse structured response.

Tasks:

1. Create LLM client interface.
2. Implement Anthropic client.
3. Implement OpenAI client.
4. Add retry logic.
5. Add timeout.
6. Add fallback provider.
7. Build prompt template.
8. Parse JSON response.
9. Validate response schema.

Acceptance criteria:

- LLM call returns structured findings.
- Invalid JSON is handled gracefully.
- If primary provider fails, fallback provider is used.
- API keys are loaded from environment variables.

---

### Phase 6 — Orchestrator

Goal: run the full audit pipeline.

Tasks:

1. Create audit job.
2. Set status to PENDING.
3. Parse contract.
4. Chunk contract.
5. For each chunk:
   - retrieve patterns
   - build prompt
   - call LLM
   - parse findings
6. Aggregate findings.
7. Deduplicate findings.
8. Score findings.
9. Generate report.
10. Persist report.
11. Update job status.

Acceptance criteria:

- Audit job progresses through statuses.
- Progress can be polled.
- Failed audits store error message.
- Completed audits produce report.

---

### Phase 7 — API Handlers

Goal: expose audit endpoints.

Tasks:

1. Implement `POST /api/v1/audit`.
2. Implement `GET /api/v1/audit/:id`.
3. Implement `GET /api/v1/audit/:id/report`.
4. Implement `GET /api/v1/exploits`.
5. Add request validation.
6. Add error middleware.
7. Add CORS middleware.

Acceptance criteria:

- API accepts valid contract submission.
- API rejects invalid input with 400.
- API returns 404 for unknown audit id.
- API returns structured JSON errors.

---

### Phase 8 — Frontend

Goal: allow user to submit contract and view report.

Tasks:

1. Build landing page.
2. Add Monaco editor.
3. Add contract name input.
4. Add submit button.
5. Call `POST /audit`.
6. Redirect to audit page.
7. Poll audit status.
8. Show progress.
9. Fetch report.
10. Render findings.
11. Add severity badges.

Acceptance criteria:

- User can paste Solidity code.
- User can submit audit.
- User sees progress.
- User sees final report.
- Critical findings are visually distinct.

---

### Phase 9 — Testing and Polish

Goal: ensure MVP works end to end.

Tasks:

1. Add backend unit tests.
2. Add parser tests.
3. Add RAG retrieval tests.
4. Add LLM response parsing tests.
5. Add API integration tests.
6. Test vulnerable contracts.
7. Test safe contracts.
8. Add README.
9. Add demo instructions.
10. Add example report.

Acceptance criteria:

- Known reentrancy contract is detected.
- Known access control contract is detected.
- Safe contract produces zero critical/high findings.
- `make test` passes.
- README explains how to run project.

---

## 15. Test Contracts

### 15.1 Vulnerable Reentrancy Contract

Path:

```text
backend/testdata/vulnerable/reentrancy.sol
```

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract VulnerableVault {
    mapping(address => uint256) public balances;

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function withdraw() external {
        uint256 amount = balances[msg.sender];

        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "Transfer failed");

        balances[msg.sender] = 0;
    }
}
```

Expected finding:

```text
vulnerability_type: reentrancy
severity: critical
affected_function: withdraw
```

---

### 15.2 Vulnerable Access Control Contract

Path:

```text
backend/testdata/vulnerable/access_control.sol
```

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract UnprotectedAdmin {
    address public owner;
    uint256 public totalSupply;

    constructor() {
        owner = msg.sender;
    }

    function mint(uint256 amount) external {
        totalSupply += amount;
    }

    function destroy() external {
        selfdestruct(payable(msg.sender));
    }
}
```

Expected finding:

```text
vulnerability_type: access_control
severity: critical
affected_function: mint or destroy
```

---

### 15.3 Safe Contract

Path:

```text
backend/testdata/safe/safe_vault.sol
```

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SafeVault {
    mapping(address => uint256) public balances;

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function withdraw() external {
        uint256 amount = balances[msg.sender];
        require(amount > 0, "No balance");

        balances[msg.sender] = 0;

        (bool success, ) = msg.sender.call{value: amount}("");
        require(success, "Transfer failed");
    }
}
```

Expected result:

```text
No critical or high findings.
```

---

## 16. Error Handling Requirements

Backend must handle:

| Error Case | Behavior |
|---|---|
| Empty source code | Return 400 INVALID_INPUT |
| Invalid contract name | Return 400 INVALID_INPUT |
| Solidity parse failure | Return 422 PARSE_ERROR |
| Unknown audit id | Return 404 AUDIT_NOT_FOUND |
| LLM timeout | Retry once, then fallback provider |
| LLM invalid JSON | Mark chunk analysis failed, continue other chunks |
| Vector DB unavailable | Continue without RAG, add warning in metadata |
| Database unavailable | Return 503 SERVICE_UNAVAILABLE |

---

## 17. Logging Requirements

Use structured logging.

Required log fields:

```text
audit_id
stage
function_name
llm_provider
duration_ms
error
```

Example:

```text
INFO audit_started audit_id=xxx
INFO parsing_complete audit_id=xxx chunks=5
INFO rag_retrieval_complete audit_id=xxx patterns=5
INFO llm_analysis_complete audit_id=xxx function=withdraw findings=1
ERROR llm_analysis_failed audit_id=xxx error=timeout
INFO audit_complete audit_id=xxx duration_ms=28400
```

---

## 18. Definition of Done

The project is complete when:

1. `docker-compose up -d` starts database.
2. `make run-backend` starts backend.
3. `make run-frontend` starts frontend.
4. User can submit Solidity code from browser.
5. Backend parses contract into chunks.
6. RAG retrieves exploit patterns.
7. LLM returns structured findings.
8. Backend saves findings.
9. User can poll audit progress.
10. User can view final report.
11. Reentrancy test contract is detected.
12. Access control test contract is detected.
13. Safe contract returns no critical/high findings.
14. `make test` passes.
15. README contains setup instructions.
16. No API keys are hardcoded.

---

## 19. Build Order for AI Coding Assistant

Implement in this exact order:

```text
1. Create repository structure.
2. Create docker-compose.yml.
3. Create .env.example.
4. Create Makefile.
5. Initialize Go backend.
6. Add config loader.
7. Add health endpoint.
8. Add database connection.
9. Add migrations.
10. Add parser types.
11. Add Solidity chunker.
12. Add exploit seed data.
13. Add embedding service.
14. Add vector store.
15. Add RAG retriever.
16. Add LLM client interface.
17. Add Anthropic client.
18. Add OpenAI client.
19. Add prompt builder.
20. Add LLM response parser.
21. Add orchestrator.
22. Add report generator.
23. Add API handlers.
24. Add frontend.
25. Add tests.
26. Add README.
```

---

## 20. Final Note

This is an MVP.

The priority is:

```text
Working end-to-end pipeline > perfect parsing > fancy UI > advanced detection
```

Do not over-engineer.

Build a working ChainMind MVP first. 
