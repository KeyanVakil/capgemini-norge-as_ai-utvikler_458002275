# PRD: Agentic AI Code Review Platform

## 1. Project Overview

### What It Does

An **agentic AI code review platform** built in Go that orchestrates multiple specialized AI agents to analyze, review, and improve code submissions. Users paste or upload code through a web UI, and the platform dispatches a chain of AI agents — each with a distinct role (security auditor, style reviewer, test generator, improvement suggester) — that collaborate to produce a comprehensive, multi-perspective code review.

### Why It's Relevant to Capgemini Norge AS

Capgemini's AI-Utvikler role centers on building solutions where **AI is an integrated part of the functionality** — specifically intelligent assistants, automated processes, and **agentic AI workflows**. This project is exactly that: a production-grade demonstration of agentic AI where multiple specialized agents coordinate autonomously, mimicking how Capgemini's teams would build AI-powered developer tools for clients.

The platform demonstrates:
- **Agentic AI in practice** — not a single LLM call, but an orchestrated multi-agent workflow with handoffs, parallel execution, and result synthesis
- **AI-integrated development** — the kind of tool Capgemini builds for clients who want AI woven into their SDLC
- **Consulting-ready architecture** — clean, containerized, extensible; the type of deliverable a Capgemini team would ship

### The Problem It Solves

Code reviews are time-consuming and inconsistent. Senior developers spend hours reviewing PRs, and coverage varies by reviewer expertise. This platform automates the tedious parts — catching security issues, style violations, missing tests, and suggesting improvements — so human reviewers can focus on architecture and design decisions.

---

## 2. Technical Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Web UI (Browser)                    │
│              HTML/CSS/JS + SSE streaming                 │
└──────────────────────┬──────────────────────────────────┘
                       │ HTTP/SSE
┌──────────────────────▼──────────────────────────────────┐
│                    Go HTTP Server                         │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │  REST API    │  │  SSE Stream  │  │  Static Files  │  │
│  │  Handlers    │  │  Handler     │  │  Server        │  │
│  └──────┬──────┘  └──────┬───────┘  └────────────────┘  │
│         │                │                               │
│  ┌──────▼────────────────▼───────┐                       │
│  │       Agent Orchestrator       │                       │
│  │  (workflow engine, DAG-based)  │                       │
│  └──┬────┬────┬────┬─────────────┘                       │
│     │    │    │    │                                      │
│  ┌──▼┐┌──▼┐┌──▼┐┌──▼──┐                                 │
│  │ A1 ││ A2 ││ A3 ││ A4  │  ← Specialized AI Agents     │
│  └──┬┘└──┬┘└──┬┘└──┬──┘                                 │
│     └────┴────┴────┘                                     │
│            │                                             │
│  ┌─────────▼─────────┐  ┌────────────────┐               │
│  │  Claude AI Client  │  │   PostgreSQL   │               │
│  │  (Anthropic API)   │  │   (reviews DB) │               │
│  └────────────────────┘  └────────────────┘               │
└──────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Responsibility |
|-----------|---------------|
| **Web UI** | Code submission form, real-time review streaming, review history browser |
| **REST API** | Accepts code submissions, serves review results and history |
| **SSE Stream Handler** | Streams agent progress and results to the UI in real-time |
| **Agent Orchestrator** | Defines and executes the agent workflow DAG — dispatches agents in parallel where possible, collects results, handles failures |
| **Security Agent** | Scans for vulnerabilities: injection, secrets exposure, unsafe patterns |
| **Style Agent** | Checks naming conventions, code structure, idiomatic patterns for the detected language |
| **Test Generator Agent** | Produces unit tests for the submitted code |
| **Improvement Agent** | Suggests refactoring, performance improvements, and cleaner abstractions; synthesizes all agent outputs into a final summary |
| **Claude AI Client** | Handles communication with the Anthropic API — prompt construction, streaming responses, rate limiting |
| **PostgreSQL** | Persists review submissions, agent results, and review history |

### Data Flow

1. User pastes code into the web UI and selects the programming language
2. UI sends a POST to `/api/reviews` with the code and language
3. API creates a review record in PostgreSQL (status: `pending`)
4. Orchestrator launches the agent workflow:
   - **Phase 1 (parallel):** Security Agent, Style Agent, and Test Generator Agent each receive the code and run concurrently
   - **Phase 2 (sequential):** Improvement Agent receives the original code + all Phase 1 results, produces a synthesis with prioritized recommendations
5. Each agent streams its progress via SSE to the UI (agent name, status, partial results)
6. Final combined review is saved to PostgreSQL (status: `completed`)
7. UI renders the full review with collapsible sections per agent

---

## 3. Tech Stack

| Technology | Role | Rationale |
|------------|------|-----------|
| **Go 1.22+** | Backend application | Listed in job requirements (Java/C#/.NET/Go). Go is ideal for concurrent agent orchestration with goroutines and channels |
| **Claude AI (Anthropic API)** | LLM powering all agents | Explicitly listed in job description. Each agent uses tailored system prompts |
| **PostgreSQL 16** | Persistence | Production-grade relational DB for review history, suitable for structured agent results |
| **HTML/CSS/JavaScript** | Web UI | Lightweight frontend — no framework needed for this scope. SSE for real-time streaming |
| **Docker + Docker Compose** | Containerization | Listed in job requirements. Single `docker compose up` to run everything |
| **GitHub Actions** | CI/CD pipeline | Listed in job requirements (CI/CD). Runs tests, linting, builds Docker image |

### What's NOT Included (and Why)

- **No cloud deployment scripts** — the job mentions cloud experience but the project demonstrates cloud-readiness through containerization. No cloud accounts needed to run it.
- **No frontend framework** — the UI is functional and clean with vanilla JS + SSE. This is a backend/AI role, not a frontend position.
- **No microservices** — a single Go binary is the right architecture for this scope. The agent pattern provides internal modularity without network overhead.

---

## 4. Features & Acceptance Criteria

### Feature 1: Code Submission

Submit code for AI-powered review through the web UI.

**Acceptance Criteria:**
- User can paste code (up to 500 lines) into a text area and select a language (Go, Java, C#, Python, JavaScript)
- Submission returns a review ID and redirects to the review page
- Code is syntax-highlighted in the UI
- Invalid submissions (empty code, unsupported language) return clear error messages

### Feature 2: Multi-Agent Orchestrated Review

An orchestrator dispatches specialized AI agents that run a coordinated review workflow.

**Acceptance Criteria:**
- Four agents run per review: Security, Style, Test Generator, Improvement
- Security, Style, and Test Generator run in parallel (Phase 1)
- Improvement agent runs after Phase 1 completes and receives all prior results
- Each agent produces structured output: findings list with severity, description, and code references
- If any agent fails (API error, timeout), the review completes with partial results and a clear failure indicator for the failed agent
- Total orchestration time is visible in the UI

### Feature 3: Real-Time Streaming Progress

Watch agents work in real-time as they analyze the code.

**Acceptance Criteria:**
- UI shows each agent's status: `waiting`, `running`, `completed`, `failed`
- Agent results stream incrementally via SSE as they are produced
- Visual indicator shows which agents are running in parallel vs. sequentially
- No page refresh needed — UI updates live

### Feature 4: Review History

Browse and revisit past code reviews.

**Acceptance Criteria:**
- Landing page shows a list of recent reviews (language, submission time, status)
- Clicking a review shows the full results with all agent outputs
- Reviews persist across container restarts (PostgreSQL volume)

### Feature 5: Test Generation Output

The Test Generator agent produces runnable unit tests for the submitted code.

**Acceptance Criteria:**
- Generated tests are displayed in a syntax-highlighted code block
- Tests are idiomatic for the detected language (e.g., `_test.go` for Go, JUnit for Java)
- Output includes both the test code and a brief explanation of what each test covers

### Feature 6: Configurable Agent Workflow

Agents can be enabled/disabled per review to customize the workflow.

**Acceptance Criteria:**
- UI provides checkboxes to toggle each agent on/off before submitting
- At least one agent must be selected
- Orchestrator respects the selection and only runs enabled agents
- Improvement agent adapts its synthesis based on which agents were active

---

## 5. Data Models

### Entity Relationship

```
Review (1) ──── (*) AgentResult
```

### Review

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `code` | TEXT | Submitted source code |
| `language` | VARCHAR(20) | Programming language |
| `status` | VARCHAR(20) | `pending`, `running`, `completed`, `failed` |
| `agents_config` | JSONB | Which agents were enabled for this review |
| `created_at` | TIMESTAMP | Submission time |
| `completed_at` | TIMESTAMP | When the last agent finished (nullable) |
| `duration_ms` | INTEGER | Total orchestration time in milliseconds (nullable) |

### AgentResult

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `review_id` | UUID | Foreign key to Review |
| `agent_name` | VARCHAR(30) | `security`, `style`, `test_generator`, `improvement` |
| `status` | VARCHAR(20) | `pending`, `running`, `completed`, `failed` |
| `findings` | JSONB | Structured output (array of findings) |
| `raw_output` | TEXT | Full LLM response text |
| `started_at` | TIMESTAMP | When the agent began (nullable) |
| `completed_at` | TIMESTAMP | When the agent finished (nullable) |
| `duration_ms` | INTEGER | Agent execution time in milliseconds (nullable) |
| `error` | TEXT | Error message if failed (nullable) |

### Finding (JSONB structure within `findings`)

```json
{
  "severity": "high | medium | low | info",
  "category": "security | style | test | improvement",
  "title": "SQL Injection via string concatenation",
  "description": "The query on line 42 concatenates user input directly...",
  "line_start": 42,
  "line_end": 42,
  "suggestion": "Use parameterized queries instead: db.Query(\"SELECT ... WHERE id = ?\", userID)"
}
```

### Database Schema (SQL)

```sql
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    language VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    agents_config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_ms INTEGER
);

CREATE TABLE agent_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    agent_name VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    findings JSONB DEFAULT '[]',
    raw_output TEXT DEFAULT '',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    UNIQUE(review_id, agent_name)
);

CREATE INDEX idx_agent_results_review_id ON agent_results(review_id);
CREATE INDEX idx_reviews_created_at ON reviews(created_at DESC);
```

---

## 6. API Design

### Base URL: `http://localhost:8080/api`

### POST /api/reviews

Create a new code review.

**Request:**
```json
{
  "code": "func main() {\n  fmt.Println(\"hello\")\n}",
  "language": "go",
  "agents": ["security", "style", "test_generator", "improvement"]
}
```

**Response (201 Created):**
```json
{
  "id": "a1b2c3d4-...",
  "status": "pending",
  "created_at": "2026-04-03T10:00:00Z"
}
```

**Errors:**
- `400` — empty code, unsupported language, no agents selected
- `413` — code exceeds 500 lines

### GET /api/reviews

List recent reviews.

**Query Parameters:**
- `limit` (int, default 20, max 100)
- `offset` (int, default 0)

**Response (200):**
```json
{
  "reviews": [
    {
      "id": "a1b2c3d4-...",
      "language": "go",
      "status": "completed",
      "created_at": "2026-04-03T10:00:00Z",
      "duration_ms": 12340
    }
  ],
  "total": 42
}
```

### GET /api/reviews/:id

Get full review details including all agent results.

**Response (200):**
```json
{
  "id": "a1b2c3d4-...",
  "code": "func main() {...}",
  "language": "go",
  "status": "completed",
  "agents_config": ["security", "style", "test_generator", "improvement"],
  "created_at": "2026-04-03T10:00:00Z",
  "completed_at": "2026-04-03T10:00:12Z",
  "duration_ms": 12340,
  "agent_results": [
    {
      "agent_name": "security",
      "status": "completed",
      "findings": [...],
      "duration_ms": 3200
    },
    {
      "agent_name": "style",
      "status": "completed",
      "findings": [...],
      "duration_ms": 2800
    },
    {
      "agent_name": "test_generator",
      "status": "completed",
      "findings": [...],
      "duration_ms": 4100
    },
    {
      "agent_name": "improvement",
      "status": "completed",
      "findings": [...],
      "duration_ms": 5400
    }
  ]
}
```

### GET /api/reviews/:id/stream

SSE endpoint for real-time review progress.

**Event Types:**
```
event: agent_status
data: {"agent_name": "security", "status": "running"}

event: agent_progress
data: {"agent_name": "security", "partial": "Analyzing for SQL injection..."}

event: agent_complete
data: {"agent_name": "security", "findings": [...], "duration_ms": 3200}

event: agent_error
data: {"agent_name": "test_generator", "error": "API timeout"}

event: review_complete
data: {"review_id": "a1b2c3d4-...", "status": "completed", "duration_ms": 12340}
```

### Authentication

None. This is a local development tool running in Docker. No auth needed.

---

## 7. Testing Strategy

### Unit Tests

**What to cover:**
- **Agent prompt construction** — each agent builds the correct system prompt and user message for its specialization
- **Orchestrator logic** — correct DAG execution order (Phase 1 parallel, Phase 2 sequential), partial failure handling, timeout behavior
- **Finding parser** — correctly extracts structured findings from LLM responses (including malformed responses)
- **API handlers** — request validation (empty code, bad language, line limit), correct HTTP status codes, response shapes
- **Database queries** — review CRUD operations, agent result upserts

**Approach:** Standard Go `testing` package with table-driven tests. Mock the Claude API client at the interface boundary so agent logic tests don't make real API calls.

### Integration Tests

**What to cover:**
- **Full review workflow** — submit code via API, verify agents run, check final review in database
- **SSE streaming** — connect to SSE endpoint, verify events arrive in correct order
- **Database persistence** — review survives container restart (via volume)
- **Concurrent reviews** — multiple reviews submitted simultaneously don't interfere

**Approach:** Use `testcontainers-go` for a real PostgreSQL instance. For Claude API, use a mock HTTP server that returns canned responses to keep tests deterministic and free of API keys.

### Target Coverage Areas

| Area | Priority |
|------|----------|
| Orchestrator (DAG execution, error handling) | High |
| Agent prompt construction | High |
| API request validation | High |
| Finding parser (LLM output → structured data) | High |
| Database repository layer | Medium |
| SSE event marshaling | Medium |

---

## 8. Infrastructure & Deployment

### Local Services (docker-compose)

| Service | Image | Port | Purpose |
|---------|-------|------|---------|
| `app` | Built from `Dockerfile` | 8080 | Go application (API + UI) |
| `db` | `postgres:16-alpine` | 5432 | Review storage |

### docker-compose.yml Overview

```yaml
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://reviewer:reviewer@db:5432/reviews?sslmode=disable
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    environment:
      - POSTGRES_USER=reviewer
      - POSTGRES_PASSWORD=reviewer
      - POSTGRES_DB=reviews
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U reviewer"]
      interval: 2s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `ANTHROPIC_API_KEY` | Yes | API key for Claude. Set in `.env` file or shell environment |
| `DATABASE_URL` | No | Auto-configured by docker-compose |

### CI/CD (GitHub Actions)

The `.github/workflows/ci.yml` pipeline:

1. **Lint** — `golangci-lint` for static analysis
2. **Test** — run unit tests with `go test ./...`
3. **Integration Test** — spin up PostgreSQL via service container, run integration tests
4. **Build** — compile Go binary, build Docker image
5. **Docker** — push image to GitHub Container Registry (on main branch only)

---

## 9. Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go              # Entry point: starts HTTP server
├── internal/
│   ├── agent/
│   │   ├── agent.go             # Agent interface and base implementation
│   │   ├── security.go          # Security review agent
│   │   ├── style.go             # Code style agent
│   │   ├── testgen.go           # Test generator agent
│   │   ├── improvement.go       # Improvement/synthesis agent
│   │   └── agent_test.go        # Agent unit tests
│   ├── orchestrator/
│   │   ├── orchestrator.go      # DAG-based workflow engine
│   │   ├── orchestrator_test.go # Orchestrator unit tests
│   │   └── workflow.go          # Workflow definition (phases, dependencies)
│   ├── api/
│   │   ├── handler.go           # HTTP handlers (REST + SSE)
│   │   ├── handler_test.go      # Handler unit tests
│   │   ├── middleware.go        # Request logging, recovery
│   │   └── router.go            # Route registration
│   ├── llm/
│   │   ├── client.go            # Claude API client (streaming support)
│   │   ├── client_test.go       # Client tests with mock server
│   │   └── mock.go              # Mock client for testing
│   ├── db/
│   │   ├── postgres.go          # PostgreSQL connection and migrations
│   │   ├── repository.go        # Review and AgentResult CRUD
│   │   └── repository_test.go   # Repository tests
│   └── model/
│       └── model.go             # Domain types: Review, AgentResult, Finding
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css        # Application styles
│   │   └── js/
│   │       └── app.js           # SSE client, UI logic
│   └── templates/
│       ├── index.html           # Landing page: submission form + review list
│       └── review.html          # Review detail page with streaming results
├── migrations/
│   └── 001_init.sql             # Database schema
├── .github/
│   └── workflows/
│       └── ci.yml               # CI/CD pipeline
├── Dockerfile                   # Multi-stage Go build
├── docker-compose.yml           # Local development stack
├── go.mod
├── go.sum
├── .env.example                 # ANTHROPIC_API_KEY=your-key-here
└── README.md                    # Setup and usage instructions
```

### Module Responsibilities

| Package | Responsibility |
|---------|---------------|
| `cmd/server` | Bootstrap: parse config, connect DB, register routes, start server |
| `internal/agent` | Each agent implements the `Agent` interface with its specialized prompt and response parser |
| `internal/orchestrator` | Manages the agent execution DAG — runs Phase 1 agents concurrently via goroutines, waits, then runs Phase 2 |
| `internal/api` | HTTP layer: request parsing, validation, response serialization, SSE streaming |
| `internal/llm` | Anthropic API client with streaming support. Defines a `Client` interface for testability |
| `internal/db` | PostgreSQL interactions: connection pooling, auto-migration, review/result CRUD |
| `internal/model` | Shared domain types used across all packages |
| `web/` | Static assets and HTML templates served by the Go application |
