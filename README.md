# Agentic Code Review Platform

An AI-powered code review platform that orchestrates multiple specialized agents to analyze code from different perspectives — security, style, test generation, and improvement synthesis — delivering comprehensive, multi-perspective reviews in real-time.

**Job Listing:** [Capgemini Norge AS — AI-Utvikler](https://www.finn.no/job/ad/458002275)

## Skills Demonstrated

| Skill | Where It's Used |
|-------|----------------|
| **Go / Concurrent programming** | Agent orchestrator uses goroutines and channels for parallel agent execution (DAG-based workflow engine) |
| **Agentic AI / LLM integration** | Four specialized AI agents with distinct system prompts, coordinated in a multi-phase pipeline with result handoff |
| **Claude API (Anthropic)** | Streaming API client with SSE parsing, structured output extraction, and error handling |
| **PostgreSQL** | Persistent storage with migrations, repository pattern, JSONB for flexible agent results |
| **Docker / Docker Compose** | Multi-stage build, health checks, service dependency ordering — one command to run everything |
| **CI/CD (GitHub Actions)** | Lint, test, integration test with PostgreSQL service container, Docker build |
| **REST API design** | Clean HTTP handlers with validation, SSE streaming for real-time progress, proper status codes |
| **Testing** | Table-driven unit tests, mock interfaces for testability, integration tests with real database |
| **Web UI** | Real-time SSE-powered interface showing agent workflow visualization, streaming progress, and structured findings |

## Architecture

```
Browser (HTML/JS + SSE)
    │
    ▼
Go HTTP Server (port 8080)
    ├── REST API (/api/reviews)
    ├── SSE Stream (/api/reviews/:id/stream)
    └── Static Files
          │
          ▼
    Agent Orchestrator (DAG-based)
    ┌─────────────────────────────────┐
    │ Phase 1 (parallel):             │
    │   Security ─┬─ Style ─┬─ Tests │
    │             └─────────┘        │
    │ Phase 2 (sequential):           │
    │   Improvement (synthesizes all) │
    └─────────────────────────────────┘
          │                    │
          ▼                    ▼
    Claude API           PostgreSQL
    (Anthropic)          (reviews DB)
```

**How the agents work:**
1. User submits code through the web UI
2. Three agents (Security, Style, Test Generator) analyze the code **in parallel**
3. The Improvement agent receives all prior results and produces a **prioritized synthesis**
4. All progress streams to the browser via Server-Sent Events in real-time

## Quick Start

```bash
# 1. Clone and configure
cp .env.example .env
# Edit .env and add your ANTHROPIC_API_KEY

# 2. Start everything
docker compose up --build

# 3. Open the UI
# http://localhost:8080
```

That's it. The app, database, and migrations all start automatically.

## Running Tests

```bash
# Unit tests (no external dependencies)
go test -v ./...

# Integration tests (requires PostgreSQL on port 5433, or set your own DATABASE_URL)
DATABASE_URL="postgres://reviewer:reviewer@localhost:5433/reviews?sslmode=disable" \
  go test -v -tags=integration ./...

# With Docker Compose (starts PostgreSQL automatically)
docker compose up db -d
DATABASE_URL="postgres://reviewer:reviewer@localhost:5433/reviews?sslmode=disable" \
  go test -v -tags=integration ./...
```

## Tech Stack

| Technology | Role | Rationale |
|------------|------|-----------|
| **Go 1.22** | Backend | Listed in job requirements. Goroutines and channels are ideal for concurrent agent orchestration |
| **Claude API** | LLM | Listed in job description. Each agent uses tailored system prompts for its specialization |
| **PostgreSQL 16** | Database | Production-grade persistence for reviews and agent results with JSONB flexibility |
| **HTML/CSS/JS + SSE** | Frontend | Lightweight real-time UI — no framework overhead for this scope |
| **Docker Compose** | Infrastructure | Listed in job requirements. Single command to run the entire stack |
| **GitHub Actions** | CI/CD | Listed in job requirements. Lint, test, integration test, build pipeline |

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/reviews` | POST | Submit code for review |
| `/api/reviews` | GET | List recent reviews |
| `/api/reviews/:id` | GET | Get full review with all agent results |
| `/api/reviews/:id/stream` | GET | SSE stream for real-time progress |

## Project Structure

```
cmd/server/         — Entry point
internal/
  agent/            — AI agent implementations (security, style, testgen, improvement)
  orchestrator/     — DAG-based workflow engine
  api/              — HTTP handlers, SSE streaming, middleware
  llm/              — Claude API client with streaming
  db/               — PostgreSQL connection, migrations, repository
  model/            — Domain types
web/                — HTML templates, CSS, JavaScript
migrations/         — Database schema
```
