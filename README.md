# Healing Eval - Self-Improving AI Agent Evaluation Pipeline

A production-grade evaluation pipeline for AI agents with modular evaluators, multi-provider LLM support, and automatic improvement suggestions.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Server                               │
│  POST /api/v1/conversations  │  GET /api/v1/evaluations         │
│  GET /api/v1/suggestions     │  GET /api/v1/metrics             │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Redis Streams                             │
│                    (Message Queue)                               │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Worker Pool                                 │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────────┐ │
│  │  Heuristic  │ │  LLM Judge  │ │  Tool Call  │ │ Coherence  │ │
│  │  Evaluator  │ │  Evaluator  │ │  Evaluator  │ │ Evaluator  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └────────────┘ │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                       PostgreSQL                                 │
│   conversations │ evaluations │ suggestions │ annotators        │
└──────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16+ (or use Docker)
- Redis 7+ (or use Docker)

### Setup

1. Clone and configure:
```bash
cp .env.example .env
# Edit .env with your API keys
```

2. Start infrastructure:
```bash
make docker-up
make migrate
```

3. Run the server:
```bash
make run-server
```

4. Run the worker (in another terminal):
```bash
make run-worker
```

## API Reference

### Ingest Conversations

```bash
curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Content-Type: application/json" \
  -d '{
    "conversations": [
      {
        "conversation_id": "conv_001",
        "agent_version": "v2.3.1",
        "turns": [
          {"turn_id": 1, "role": "user", "content": "Book a flight to NYC", "timestamp": "2024-01-15T10:30:00Z"},
          {"turn_id": 2, "role": "assistant", "content": "I found several flights...", "timestamp": "2024-01-15T10:30:02Z"}
        ],
        "feedback": {
          "user_rating": 4
        }
      }
    ]
  }'
```

### Query Evaluations

```bash
curl -X POST http://localhost:8080/api/v1/evaluations/query \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_ids": ["conv_001"],
    "limit": 10
  }'
```

### Get Suggestions

```bash
curl http://localhost:8080/api/v1/suggestions
```

### Evaluator Metrics

```bash
curl http://localhost:8080/api/v1/metrics/evaluators
```

## Evaluators

| Type | Description | Weight |
|------|-------------|--------|
| Heuristic | Latency, format, tool execution checks | 0.20 |
| LLM Judge | Response quality, helpfulness, factuality | 0.40 |
| Tool Call | Tool selection, parameter accuracy, hallucination | 0.25 |
| Coherence | Multi-turn context, contradictions | 0.15 |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | API server port |
| `WORKER_CONCURRENCY` | 10 | Parallel evaluation workers |
| `LLM_DEFAULT_PROVIDER` | openai | Primary LLM provider |
| `DB_MAX_CONNS` | 25 | PostgreSQL connection pool |

## Deployment

### Free Hosting (No Credit Card)

Deploy for free on Render.com using the included `render.yaml`:

**Quick Deploy:**
1. Push repo to GitHub
2. Sign up at [render.com](https://render.com) (no credit card)
3. Create new Blueprint → Connect your repo
4. Add API keys in environment variables
5. Run database migrations

📖 **Full Guide**: See [DEPLOYMENT.md](DEPLOYMENT.md) for detailed instructions

🔀 **Alternatives**: See [DEPLOYMENT_ALTERNATIVES.md](DEPLOYMENT_ALTERNATIVES.md) for other free options (Railway, VPS, Split Stack, etc.)

🤖 **Ollama/Open-Source Models**: See [OLLAMA_DEPLOYMENT.md](OLLAMA_DEPLOYMENT.md) for running Llama, Mistral, etc.

**What's Included:**
- ✅ PostgreSQL database (free tier)
- ✅ Redis instance (free tier)
- ✅ API server + background worker
- ✅ Auto-deploy on git push
- ✅ Free SSL certificates
- ✅ Support for OpenAI, Anthropic, and Ollama

**Limitations:**
- Services sleep after 15 min inactivity (~30-50s cold start)
- 750 hours/month shared across services

### Using Open-Source Models (Ollama)

Want to use Llama, Mistral, or other open models? You have options:

1. **OpenRouter** (easiest): Use hosted open models via OpenAI-compatible API (~$0.10/1M tokens)
2. **Free VPS**: Run Ollama on a separate free VPS (Alavps, etc.)
3. **Modal Labs**: GPU-accelerated serverless inference
4. **Together AI**: Hosted open models, cheap pricing

See [OLLAMA_DEPLOYMENT.md](OLLAMA_DEPLOYMENT.md) for complete setup instructions.

## Development

```bash
make build      # Build binaries
make test       # Run tests
make lint       # Run linter
make clean      # Clean build artifacts
```

## Project Structure

```
├── cmd/
│   ├── server/         # API server
│   └── worker/         # Evaluation worker
├── internal/
│   ├── api/            # HTTP handlers
│   ├── config/         # Configuration
│   ├── domain/         # Domain models
│   ├── evaluator/      # Evaluation framework
│   ├── llm/            # LLM providers
│   ├── queue/          # Redis Streams
│   ├── storage/        # PostgreSQL repos
│   └── worker/         # Worker implementation
├── migrations/         # SQL migrations
└── docker-compose.yml
```

