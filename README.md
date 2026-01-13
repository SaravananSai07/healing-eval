# Healing Eval - Self-Improving AI Agent Evaluation Pipeline

**🌐 Live Demo**: [https://healing-eval-server-production.up.railway.app/](https://healing-eval-server-production.up.railway.app/)

An evaluation pipeline for AI agents with modular evaluators, multi-provider LLM support, and automatic improvement suggestions. The system continuously evaluates agent performance, detects failure patterns, and generates actionable improvement suggestions for prompts and tools.

**Test Coverage**: 21/21 PASS | **Token Savings**: 60-75% on long conversations

---

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Web UI Guide](#web-ui-guide)
- [Evaluation Framework](#evaluation-framework)
- [Self-Improvement System](#self-improvement-system)
- [API Reference](#api-reference)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Scaling Strategy](#scaling-strategy)

---

## Overview

Healing Eval is an automated evaluation pipeline designed to enable continuous improvement of AI agents in production. It addresses the core challenge of detecting regressions, aligning evaluation scores with user feedback, and automatically generating actionable suggestions for both prompts and tools.

### The Problem

Modern AI agents handle complex, multi-turn customer interactions. As they scale, they need robust infrastructure to continuously evaluate and improve based on:
- **User feedback**: Explicit ratings, implicit signals (rephrasing, early exits)
- **Internal ops feedback**: Quality reviews, escalations
- **Human annotations**: Labeled conversation samples

### The Solution

Healing Eval provides:
1. **Modular Evaluation Framework**: Multiple evaluator types (LLM-as-Judge, Tool Call, Coherence, Heuristic)
2. **Pattern Detection**: Automatically identifies recurring failure patterns
3. **Self-Updating Mechanism**: Generates improvement suggestions for prompts and tools
4. **Meta-Evaluation**: Continuously improves the evaluators themselves
5. **Human Review Integration**: Routes low-confidence evaluations for human review

---

## Key Features

### ✅ Data Ingestion Layer
- Ingest multi-turn conversation logs with full context
- Process feedback signals (user ratings, ops annotations, human labels)
- Handle high throughput (~1000+ conversations/minute)
- Support both batch and real-time processing

### ✅ Evaluation Framework
- **LLM-as-Judge**: Assesses response quality, helpfulness, factuality
- **Tool Call Evaluator**: Verifies correct tool selection and parameter accuracy
- **Multi-turn Coherence**: Checks context maintenance and consistency across turns
- **Heuristic Checks**: Format compliance, latency thresholds, required fields

### ✅ Feedback Integration
- Ingest annotations from multiple human annotators
- Handle annotator disagreement (Cohen's/Fleiss' Kappa)
- Weight evaluations by annotation quality/confidence
- Support confidence-based routing (auto-label vs. human review)

### ✅ Self-Updating Mechanism
- **For Prompts**: Identify failure patterns, suggest specific prompt modifications
- **For Tools**: Detect tool schema issues, suggest parameter description improvements
- Generate suggestions with rationale and expected impact

### ✅ Meta-Evaluation
- Calibrate LLM-as-Judge against human annotations
- Identify blind spots (failure categories evaluators miss)
- Track evaluator accuracy (precision/recall vs. human ground truth)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Server                               │
│  POST /api/v1/conversations  │  GET /api/v1/evaluations         │
│  GET /api/v1/suggestions     │  GET /api/v1/metrics             │
│  Web UI: Dashboard, Conversations, Suggestions, Reviews, Metrics│
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Redis Streams                             │
│                    (Message Queue)                               │
│              Consumer Groups for Load Balancing                  │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                      Worker Pool                                 │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌────────────┐ │
│  │  Heuristic  │ │  LLM Judge  │ │  Tool Call  │ │ Coherence  │ │
│  │  Evaluator  │ │  Evaluator  │ │  Evaluator  │ │ Evaluator  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └────────────┘ │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │         Pattern Detector → Suggester                        │ │
│  │    (Detects failures → Generates suggestions)               │ │
│  └─────────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               ▼
┌──────────────────────────────────────────────────────────────────┐
│                       PostgreSQL                                 │
│   conversations │ evaluations │ suggestions │ review_queue      │
│   annotators    │ meta_evaluation_metrics                       │
└──────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

- **Asynchronous Processing**: Queue-based architecture decouples API from evaluation processing
- **Modular Evaluators**: Each evaluator runs independently, enabling parallel execution
- **Intelligent Windowing**: Reduces token usage by 60-75% on long conversations
- **Graceful Degradation**: System continues even if individual evaluators fail

📖 **Detailed Design Rationale**: See [DESIGN_DECISIONS.md](DESIGN_DECISIONS.md)

---

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- PostgreSQL 16+ (or use Docker)
- Redis 7+ (or use Docker)

### Local Setup

1. **Clone and configure**:
```bash
git clone <repository-url>
cd healing-eval
cp .env.example .env
# Edit .env with your API keys (OpenAI, Anthropic, or OpenRouter)
```

2. **Start infrastructure**:
```bash
make docker-up
make migrate
```

3. **Run the server** (in one terminal):
```bash
make run-server
```

4. **Run the worker** (in another terminal):
```bash
make run-worker
```

5. **Access the Web UI**:
   - Open [http://localhost:8080](http://localhost:8080) in your browser
   - You'll see the Dashboard with system overview

### Using the Live Demo

Visit **[https://healing-eval-server-production.up.railway.app/](https://healing-eval-server-production.up.railway.app/)** to explore the system without local setup.

---

## Web UI Guide

The web interface provides a complete view of your evaluation pipeline. Here's how to use each section:

### 🏠 Dashboard

**URL**: `/` or `/dashboard`

**What you'll see**:
- **Overview Cards**: Total conversations, evaluations, suggestions, and average score
- **Recent Evaluations**: Latest evaluation results with scores and issue counts
- **Active Issues**: Conversations flagged for review with priority levels
- **Evaluator Performance**: Success rates and performance metrics for each evaluator

**How it works**: The dashboard aggregates data from all conversations and provides real-time insights into system health and agent performance.

### 💬 Conversations

**URL**: `/conversations`

**What you'll see**:
- **Upload Form**: JSON textarea to submit conversations for evaluation
- **Evaluated Conversations Table**: List of all processed conversations with:
  - Conversation ID
  - Number of turns
  - Overall score
  - Issue count
  - Evaluation timestamp
  - View button to see details

**How to use**:
1. Paste conversation JSON in the textarea (see [API Reference](#api-reference) for format)
2. Click "Submit for Evaluation"
3. The conversation is queued for processing
4. Once evaluated, it appears in the table below

**Example conversation format**:
```json
[{
  "conversation_id": "conv_001",
  "agent_version": "v2.3.1",
  "turns": [
    {"turn_id": 1, "role": "user", "content": "Book a flight to NYC", "timestamp": "2024-01-15T10:30:00Z"},
    {"turn_id": 2, "role": "assistant", "content": "I'll help you...", "timestamp": "2024-01-15T10:30:02Z"}
  ],
  "feedback": {"user_rating": 4}
}]
```

### 💡 Suggestions

**URL**: `/suggestions`

**What you'll see**:
- **Filter Buttons**: All, Pending, Approved
- **Suggestions Table**: Improvement suggestions with:
  - Type (Prompt or Tool)
  - Target (what to modify)
  - Suggestion text
  - Confidence score
  - Status (Pending/Approved/Rejected)
  - Approve/Reject buttons (for pending suggestions)

**How it works**:
1. System detects failure patterns (e.g., 5+ occurrences of same issue type)
2. Generates suggestions via LLM analysis
3. Suggestions appear in the table
4. Review and approve/reject suggestions
5. Approved suggestions can be applied to improve the agent

**Generating suggestions**:
- Suggestions are generated automatically when patterns are detected
- Or trigger manually via API: `POST /api/v1/suggestions/generate`

**Example suggestion**:
- **Type**: Prompt
- **Target**: flight_search tool usage
- **Suggestion**: "Add explicit date format instruction (YYYY-MM-DD) to reduce date inference errors"
- **Rationale**: "Date format errors detected in 8 conversations, causing tool execution failures"

### 📊 Metrics

**URL**: `/metrics`

**What you'll see**:
- **Evaluator Accuracy**: Precision, recall, F1 scores vs. human annotations
- **Calibration**: Correlation between evaluator scores and human judgments
- **Blind Spots**: Failure categories that evaluators miss
- **Evaluator Performance**: Success rates, token usage, cost per evaluation

**How it works**: The system continuously tracks how well evaluators align with human judgments, identifying areas for improvement in the evaluation framework itself.

### 👥 Reviews

**URL**: `/reviews`

**What you'll see**:
- **Pending Reviews Table**: Conversations flagged for human review with:
  - Conversation ID
  - Priority (1=high, 2=medium, 3=low)
  - Reason for review (low confidence, partial evaluation, etc.)
  - Assign and Complete buttons

**How it works**:
- Conversations are automatically routed to review queue when:
  - Evaluation confidence < 0.6
  - Overall score < 0.5
  - Partial evaluation failures
- Reviewers can assign, review, and complete items
- Completed reviews feed back into meta-evaluation metrics

---

## Evaluation Framework

The system uses four complementary evaluators, each focusing on different aspects of agent performance:

| Evaluator | Purpose | Weight | When It Runs | Windowing Strategy |
|-----------|---------|--------|--------------|-------------------|
| **Heuristic** | Latency, format, tool execution checks | 20% | Always | N/A (fast checks) |
| **LLM Judge** | Response quality, helpfulness, factuality | 40% | Always | Last 15 turns + summary (if >30 turns) |
| **Tool Call** | Tool selection, parameter accuracy, hallucination | 25% | Only if tool calls present | Last 20 turns + tool stats (if >40 turns) |
| **Coherence** | Multi-turn context, contradictions | 15% | Only if 3+ turns | Last 10 turns + LLM summary (if >20 turns) |

### Evaluator Details

#### Heuristic Evaluator
- **Checks**: Latency thresholds, format compliance, tool execution success
- **No LLM calls**: Pure rule-based checks
- **Confidence**: 0.95 (high - deterministic)
- **Example issues**: "Response latency 1200ms exceeds 1000ms target"

#### LLM-as-Judge Evaluator
- **Measures**: Response quality, helpfulness, factuality
- **Uses**: LLM to evaluate agent responses
- **Confidence**: 0.80
- **Windowing**: For conversations >30 turns, shows last 15 turns + summary of earlier context
- **Example issues**: "Response lacks specific details about flight options"

#### Tool Call Evaluator
- **Measures**: 
  - Was the correct tool selected?
  - Were parameters extracted accurately?
  - Did hallucinated parameters occur?
  - Did tool execution succeed?
- **Confidence**: 0.80
- **Windowing**: For conversations >40 turns, shows last 20 turns + tool usage statistics
- **Example issues**: "Wrong tool selected: used hotel_search instead of hotel_cancel"

#### Coherence Evaluator
- **Measures**:
  - Coherence across conversation turns
  - Consistency (no contradictions)
  - Proper handling of references and context
- **Confidence**: 0.80
- **Windowing**: For conversations >20 turns, shows last 10 turns + intelligent LLM-generated summary
- **Example issues**: "Agent lost context of vegetarian requirement mentioned in turn 1"

### Intelligent Windowing

For long conversations, evaluators use intelligent windowing to reduce token usage while maintaining evaluation quality:

**Token Savings**:
- 35-turn conversation: ~60% savings (3000 vs 7500 tokens)
- 50-turn conversation: ~65% savings (3500 vs 10000 tokens)
- 100-turn conversation: ~75% savings (4000 vs 16000 tokens)

**Windowing Activation**:
- **LLM Judge**: Activates at 30+ turns (window: last 15 turns + summary)
- **Tool Call**: Activates at 40+ turns (window: last 20 turns + tool stats)
- **Coherence**: Activates at 20+ turns (window: last 10 turns + LLM summary)

---

## Core Techniques

The system employs several techniques to handle production-scale evaluation efficiently:

### Conversation Windowing

**What it is**: Instead of sending entire long conversations to LLMs, we send only recent turns plus a summary of earlier context.

**Why it's used**: 
- Reduces token usage by 60-75% on long conversations
- Keeps evaluation costs manageable
- Maintains evaluation quality by preserving recent context

**How it works**: Each evaluator uses a different window size based on its needs:
- LLM Judge (15 turns): Recent context sufficient for quality assessment
- Tool Call (20 turns): More history needed to detect tool usage patterns
- Coherence (10 turns): Focuses on recent turns where contradictions are most likely

### Intelligent Summarization

**What it is**: For conversations exceeding window thresholds, earlier turns are summarized using LLM-generated summaries.

**Why it's used**: 
- Preserves key information (topics, entities, user preferences) without full context
- Enables evaluation of long conversations without token bloat
- Falls back to simple summarization if LLM summarization fails

**How it works**: The Coherence evaluator uses LLM to generate summaries that capture:
- Key topics discussed
- Important entities mentioned
- User commitments and preferences
- Contextual information needed for evaluation

### Token Tracking & Budgeting

**What it is**: System tracks token usage per evaluator and per conversation, with configurable budgets.

**Why it's used**:
- Cost visibility and control
- Early detection of expensive conversations
- Prevents runaway costs from malformed inputs
- Enables optimization based on actual usage patterns

**How it works**: 
- Estimates tokens before LLM calls
- Tracks actual usage per evaluator
- Calculates costs based on provider pricing
- Warns when approaching context limits (80% threshold)

### Message-Level Truncation

**What it is**: Individual messages/turns are truncated based on character limits, not just turn counts.

**Why it's used**:
- Handles extremely long messages within a single turn
- Prevents token bloat from verbose responses
- Works alongside turn-based windowing for comprehensive coverage

**How it works**:
- **Per-message limit**: 4,000 characters per turn
- **Truncation strategy**: Keeps first 60% + last 40% with ellipsis marker
- **Total conversation budget**: 15,000 characters across all messages
- Once total budget exceeded, remaining turns are excluded
- Applied before turn-based windowing, ensuring both protections work together

### Prompt Sanitization

**What it is**: Multi-layer sanitization of user-generated content before sending to LLMs.

**Why it's used**:
- Prevents prompt injection attacks
- Protects evaluators from manipulation
- Ensures evaluation integrity

**How it works**:
- Pattern-based detection of common injection attempts
- Replaces suspicious content with `[SANITIZED]` markers
- Applied in conjunction with message truncation

### Pattern Aggregation

**What it is**: Groups similar issues across multiple conversations to identify recurring problems.

**Why it's used**:
- Detects regressions before they impact many users
- Identifies systematic issues (not just one-off errors)
- Enables targeted improvements

**How it works**:
- Groups issues by type and severity
- Requires minimum 5 occurrences to form a pattern
- Tracks examples and conversation IDs
- Generates suggestions when patterns are detected

### Confidence-Based Routing

**What it is**: Automatically routes low-confidence evaluations to human review queue.

**Why it's used**:
- Ensures quality by having humans review uncertain cases
- Improves evaluator accuracy over time (feedback loop)
- Balances automation with human oversight

**How it works**:
- Evaluations with confidence < 0.6 are flagged
- Low scores (< 0.5) trigger review
- Partial evaluation failures are routed
- Priority levels (1=high, 2=medium, 3=low) guide review order

### Graceful Degradation

**What it is**: System continues operating even if individual evaluators fail.

**Why it's used**:
- High availability - one evaluator failure doesn't stop evaluation
- Partial evaluations are still valuable
- Enables independent scaling and maintenance

**How it works**:
- Each evaluator runs independently with timeouts (30s)
- Failures are recorded with reasons
- Overall score adjusts for missing evaluators
- Status tracked: success / partial / failed

---

## Self-Improvement System

The system automatically detects failure patterns and generates improvement suggestions.

### Pattern Detection

The system analyzes evaluations to find recurring issues:
- Groups issues by type and severity
- Requires minimum 5 occurrences to create a pattern
- Tracks examples and conversation IDs

**Example patterns**:
- "Date format errors detected in 8 conversations" → Tool parameter issue
- "Wrong tool selection in 6 conversations" → Prompt clarity issue
- "Context loss in 12 multi-turn conversations" → Context management issue

### Suggestion Generation

When patterns are detected, the system generates actionable suggestions:

**For Prompts**:
- Identifies failure patterns
- Suggests specific prompt modifications
- Provides rationale and expected impact

**For Tools**:
- Detects tool schema issues
- Suggests parameter description improvements
- Identifies missing validation rules

**Example suggestion**:
```json
{
  "type": "prompt",
  "target": "flight_search tool usage",
  "suggestion": "Add explicit date format instruction: 'Always use YYYY-MM-DD format for dates'",
  "rationale": "Date format errors detected in 8 conversations, causing 15% tool execution failures",
  "confidence": 0.85
}
```

### Meta-Evaluation

The system continuously improves its own evaluators:

**Calibration**:
- Compares evaluator scores against human annotations
- Calculates correlation (Pearson/Spearman)
- Identifies when evaluators diverge from human judgment

**Blind Spot Detection**:
- Identifies failure categories that evaluators miss
- Flags issues that humans catch but evaluators don't
- Helps improve evaluator coverage

**Accuracy Tracking**:
- Measures precision/recall of automated evaluators
- Tracks performance over time
- Identifies evaluator drift

**The Flywheel**:
```
Agent outputs → Evaluations → Human feedback → Better evaluators → Better evaluations
```

---

## API Reference

### Ingest Conversations

Submit conversations for evaluation:

```bash
curl -X POST https://healing-eval-server-production.up.railway.app/api/v1/conversations \
  -H "Content-Type: application/json" \
  -d '{
    "conversations": [
      {
        "conversation_id": "conv_001",
        "agent_version": "v2.3.1",
        "turns": [
          {
            "turn_id": 1,
            "role": "user",
            "content": "I need to book a flight to NYC next week",
            "timestamp": "2024-01-15T10:30:00Z"
          },
          {
            "turn_id": 2,
            "role": "assistant",
            "content": "I'll help you book a flight to NYC...",
            "tool_calls": [
              {
                "tool_name": "flight_search",
                "parameters": {
                  "destination": "NYC",
                  "departure_date": "2024-01-22"
                },
                "result": {"status": "success", "flights": ["..."]},
                "latency_ms": 450
              }
            ],
            "timestamp": "2024-01-15T10:30:02Z"
          }
        ],
        "feedback": {
          "user_rating": 4,
          "ops_review": {
            "quality": "good",
            "notes": "Correct tool usage"
          },
          "annotations": [
            {
              "type": "tool_accuracy",
              "label": "correct",
              "annotator_id": "ann_001"
            }
          ]
        },
        "metadata": {
          "total_latency_ms": 1200,
          "mission_completed": true
        }
      }
    ]
  }'
```

### Query Evaluations

Retrieve evaluation results:

```bash
curl -X POST https://healing-eval-server-production.up.railway.app/api/v1/evaluations/query \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_ids": ["conv_001"],
    "date_from": "2024-01-15T00:00:00Z",
    "date_to": "2024-01-16T00:00:00Z",
    "max_overall_score": 0.7,
    "limit": 100
  }'
```

**Response**:
```json
{
  "evaluations": [
    {
      "id": "eval_xyz789",
      "conversation_id": "conv_001",
      "evaluator_type": "llm_judge",
      "scores": {
        "overall": 0.87,
        "response_quality": 0.90,
        "helpfulness": 0.85,
        "factuality": 0.88
      },
      "issues": [
        {
          "type": "latency",
          "severity": "warning",
          "description": "Response latency 1200ms exceeds 1000ms target"
        }
      ],
      "confidence": 0.80,
      "created_at": "2024-01-15T10:30:05Z"
    }
  ],
  "total": 1
}
```

### Generate Suggestions

Trigger suggestion generation:

```bash
curl -X POST https://healing-eval-server-production.up.railway.app/api/v1/suggestions/generate \
  -H "Content-Type: application/json" \
  -d '{
    "hours_back": 24,
    "max_score": 0.7,
    "min_occurrences": 5
  }'
```

**Response**:
```json
{
  "patterns_detected": 3,
  "suggestions_generated": 3,
  "suggestions_stored": 3,
  "message": "Suggestion generation completed successfully"
}
```

### Get Suggestions

List all suggestions:

```bash
curl https://healing-eval-server-production.up.railway.app/api/v1/suggestions
```

Filter by status:

```bash
curl https://healing-eval-server-production.up.railway.app/api/v1/suggestions?status=pending
```

### Approve/Reject Suggestions

Approve a suggestion:

```bash
curl -X POST https://healing-eval-server-production.up.railway.app/api/v1/suggestions/{id}/approve \
  -H "Content-Type: application/json" \
  -d '{
    "apply_strategy": "immediate",
    "notes": "Looks good, will apply"
  }'
```

Reject a suggestion:

```bash
curl -X POST https://healing-eval-server-production.up.railway.app/api/v1/suggestions/{id}/reject \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Not applicable to our use case",
    "notes": "We handle dates differently"
  }'
```

### Get Metrics

Evaluator performance metrics:

```bash
curl https://healing-eval-server-production.up.railway.app/api/v1/metrics/evaluators
```

Calibration metrics:

```bash
curl https://healing-eval-server-production.up.railway.app/api/v1/metrics/calibration
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | API server port |
| `WORKER_CONCURRENCY` | 10 | Parallel evaluation workers |
| `WORKER_BATCH_SIZE` | 10 | Batch size for processing |
| `LLM_DEFAULT_PROVIDER` | openai | Primary LLM provider (openai/anthropic/ollama/openrouter) |
| `DB_MAX_CONNS` | 25 | PostgreSQL connection pool size |
| `REDIS_HOST` | localhost | Redis host (use service name in Docker) |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | - | Redis password (required in production) |
| `REDIS_URL` | - | Redis connection URL (alternative to individual vars) |

### LLM Providers

The system supports multiple LLM providers:

**OpenAI**:
```bash
OPENAI_API_KEY=sk-...
LLM_DEFAULT_PROVIDER=openai
```

**Anthropic**:
```bash
ANTHROPIC_API_KEY=sk-ant-...
LLM_DEFAULT_PROVIDER=anthropic
```

**OpenRouter** (Free open models):
```bash
OPENROUTER_API_KEY=sk-or-...
LLM_DEFAULT_PROVIDER=openrouter
OPENROUTER_MODEL=nvidia/nemotron-3-nano-30b-a3b:free
```

**Ollama** (Local models):
```bash
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.1:8b
LLM_DEFAULT_PROVIDER=ollama
```

📖 **LLM Setup Guide**: See [OLLAMA_DEPLOYMENT.md](OLLAMA_DEPLOYMENT.md) for detailed instructions

---

## Deployment

### Railway (Current Production)

The system is currently deployed on Railway at:
**https://healing-eval-server-production.up.railway.app/**

**Services**:
- API Server + Web UI
- Worker (background evaluation processing)
- PostgreSQL database
- Redis queue

### Docker Compose

```bash
# Start all services
docker-compose up -d

# Run migrations
docker-compose exec server /server migrate

# View logs
docker-compose logs -f worker
```

---

## Scaling Strategy

The system is designed to scale from current throughput to 10x and 100x loads.

### Current Baseline
- **Capacity**: ~1,000 conversations/minute
- **Components**: Single API server, 10 concurrent workers, single PostgreSQL/Redis

### 10x Scale (10,000 conversations/minute)

**Changes**:
- Horizontal worker scaling (10 worker instances)
- Database read replicas (2-3 replicas)
- Connection pooling (PgBouncer)
- Redis clustering

**Cost**: ~$50-100/month

### 100x Scale (100,000 conversations/minute)

**Changes**:
- API load balancing (multiple API servers)
- Database sharding by conversation_id
- Redis cluster with partitioning
- CDN for static assets
- Regional deployment

**Cost**: ~$500-1000/month

📖 **Detailed Scaling Plan**: See [SCALING_STRATEGY.md](SCALING_STRATEGY.md)

---

## Sample Scenarios

The pipeline handles real-world scenarios:

### Scenario 1: Tool Call Regression

**Problem**: After a prompt update, agent calls `flight_search` with incorrect date formats (DD/MM/YYYY instead of YYYY-MM-DD), causing 15% failures.

**Detection**: Tool Call Evaluator detects parameter format errors across multiple conversations.

**Suggestion Generated**: "Add explicit date format instruction: 'Always use YYYY-MM-DD format for dates in tool parameters'"

**Expected Outcome**: Pattern detected, alert generated, prompt fix suggested.

### Scenario 2: Context Loss

**Problem**: On conversations > 5 turns, agent forgets preferences from turn 1-2 (e.g., vegetarian requirement).

**Detection**: Coherence Evaluator flags context resolution failures.

**Suggestion Generated**: "Improve context retention: Summarize key user preferences at start of each turn"

**Expected Outcome**: Coherence evaluator flags context loss, suggestion generated for prompt improvement.

### Scenario 3: Annotator Disagreement

**Problem**: Two annotators disagree on response helpfulness (one says "helpful", other says "not helpful").

**Detection**: System calculates Cohen's Kappa, detects low agreement (<0.6).

**Expected Outcome**: Conversation routed to review queue for tiebreaker, disagreement metrics tracked.

---

## Project Structure

```
├── cmd/
│   ├── server/         # API server + Web UI
│   ├── worker/         # Evaluation worker
│   └── meta-eval/      # Meta-evaluation job
├── internal/
│   ├── api/            # HTTP handlers (REST + Web)
│   ├── config/         # Configuration management
│   ├── domain/         # Domain models
│   ├── evaluator/      # Evaluation framework
│   │   ├── heuristic.go
│   │   ├── llm_judge.go
│   │   ├── tool_call.go
│   │   └── coherence.go
│   ├── improvement/    # Pattern detection & suggestions
│   ├── llm/            # LLM providers (OpenAI, Anthropic, Ollama, OpenRouter)
│   ├── queue/          # Redis Streams integration
│   ├── storage/        # PostgreSQL repositories
│   └── worker/         # Worker implementation
├── web/
│   └── templates/      # HTML templates (Dashboard, Conversations, etc.)
├── migrations/         # SQL migrations
├── docker-compose.yml  # Local development setup
└── render.yaml         # Render.com deployment config
```

---

## Development

```bash
make build      # Build binaries
make test       # Run tests
make lint       # Run linter
make clean      # Clean build artifacts
make docker-up  # Start Docker services
make migrate    # Run database migrations
```

---

## Test Results

**Status**: ✅ System tested and validated

### Performance Results

| Conversation | Processing Time | Evaluations | Status |
|--------------|----------------|-------------|---------|
| 3 turns (short) | <1s | 2 | ✅ PASS |
| 7 turns (medium + tools) | <2s | 2 | ✅ PASS |
| 35 turns (long) | <3s | 2 | ✅ PASS |
| 50 turns (very long) | <3s | 2 | ✅ PASS |

### Evaluator Performance Metrics

| Evaluator | Precision | Recall | F1 Score | Sample Count |
|-----------|-----------|--------|----------|--------------|
| Heuristic | 0.98 | 0.95 | **0.96** | 2,100 |
| Tool Call | 0.94 | 0.88 | **0.91** | 890 |
| LLM Judge | 0.89 | 0.76 | **0.82** | 1,250 |
| Coherence | 0.85 | 0.72 | **0.78** | 654 |

📖 **Full Test Report**: See [TEST_RESULTS.md](TEST_RESULTS.md)

---

