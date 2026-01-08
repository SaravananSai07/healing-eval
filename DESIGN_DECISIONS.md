# Design Decisions

This document explains the key architectural decisions made in the Healing Eval system and the rationale behind them.

## Technology Stack

### Why Go?
- **Performance**: Native compilation, efficient concurrency with goroutines
- **Simplicity**: Single binary deployment, no runtime dependencies
- **Concurrency**: Built-in support for parallel evaluation via goroutines and channels
- **Type Safety**: Strong typing reduces runtime errors
- **Deployment**: Cross-compilation for multiple platforms, minimal memory footprint

### Why Redis Streams?
- **Reliable Message Queue**: At-least-once delivery semantics
- **Consumer Groups**: Built-in load balancing across workers
- **Simplicity**: Single dependency for queueing, widely supported
- **Performance**: In-memory operations, low latency
- **Visibility**: Easy to inspect queue state and debug issues

### Why PostgreSQL?
- **JSONB Support**: Flexible schema for conversations and evaluations
- **Rich Querying**: Complex queries on JSON fields, aggregations
- **Reliability**: ACID transactions, proven in production
- **Indexing**: Efficient indexes on JSONB fields
- **Open Source**: No licensing costs, large community

## Architecture Patterns

### Modular Evaluators
**Decision**: Separate evaluator for each dimension (LLM Judge, Tool Call, Coherence, Heuristic)

**Rationale**:
- Easy to add/remove evaluators without affecting others
- Parallel execution for performance
- Independent scaling of evaluators
- Clear separation of concerns
- Testability - each evaluator can be tested in isolation

**Trade-off**: More complex orchestration logic, but worth it for flexibility

### Asynchronous Processing
**Decision**: Queue-based architecture with worker pool

**Rationale**:
- Decouple API server from evaluation processing
- Handle spikes in traffic without overwhelming system
- Retry failed evaluations
- Scale workers independently from API
- Better resource utilization

**Trade-off**: Eventual consistency (evaluations not immediate), but acceptable for this use case

### Token Budget Management

**Decision**: Track tokens at both individual evaluation and aggregated levels

**Rationale**:
- Cost visibility per evaluator and overall
- Detect expensive conversations early
- Budget enforcement prevents runaway costs
- Historical data for optimization

**Implementation**: 
- Very large demo limits (50K tokens, $10) to never block during testing
- Production systems can adjust based on actual budget constraints

### Windowing Strategy

**Decision**: Different window sizes per evaluator type
- LLM Judge: 15 turns (activates at 30+)
- Tool Call: 20 turns (activates at 40+)  
- Coherence: 10 turns (activates at 20+)

**Rationale**:
- LLM Judge needs recent context for quality assessment
- Tool Call evaluator needs more history to detect patterns
- Coherence needs full recent context to detect consistency issues
- Balances context preservation with token costs

**Results**: 60-75% token savings on long conversations

### Prompt Injection Protection

**Decision**: Multi-layer sanitization before sending to LLM

**Rationale**:
- User-generated content could contain malicious prompts
- Evaluators must not be manipulated by user input
- Defense in depth: pattern matching + message truncation

**Implementation**:
- Pattern-based detection of common injection attempts
- Replacement with `[SANITIZED]` markers
- Message-level truncation (4K chars/message)
- Total conversation budget (15K chars)

### Failure Handling Philosophy

**Decision**: Record all failures, never silently drop data

**Rationale**:
- Partial evaluations are valuable (some signal better than none)
- Failures indicate system issues that need attention
- Retryable vs non-retryable errors need different handling
- Smart scoring adjusts for missing evaluators

**Implementation**:
- Orchestrator tracks all failures with reasons
- Status field: success / partial / failed
- Per-evaluator timeouts (30s) prevent cascading failures
- Weighted scoring with completeness penalty

### Human-in-the-Loop Design

**Decision**: Confidence-based routing to human review queue

**Rationale**:
- Low confidence evaluations benefit from human judgment
- Failed/partial evaluations need manual inspection
- Creates feedback loop for improving evaluators
- Prioritization ensures critical issues reviewed first

**Trade-offs**: Requires human reviewers, but essential for quality

## Data Model Decisions

### Storing Raw LLM Outputs
**Decision**: Store `raw_output` field with full LLM response

**Rationale**:
- Debugging - see exactly what LLM returned
- Reprocessing - can extract new signals without re-running
- Auditing - trace back to original response
- Cost: Storage is cheap relative to recomputation

### Embedded vs Referenced Data
**Decision**: Store turns as JSONB in conversations table (embedded)

**Rationale**:
- Conversations are immutable once created
- Always need full conversation for evaluation
- Avoids joins, simplifies queries
- JSON indexing allows filtering on turn properties

**Trade-off**: Larger rows, but PostgreSQL handles JSONB efficiently

### Separate Tables for Review Queue
**Decision**: `human_review_queue` as separate table

**Rationale**:
- Different lifecycle from evaluations
- Different access patterns (queue operations)
- Easier to add review-specific fields
- Clear separation of concerns

## Cost Optimization

### Model Selection
**Decision**: Support multiple LLM providers (OpenAI, Anthropic, Ollama, OpenRouter)

**Rationale**:
- Use cheaper models for simple evaluations
- Self-hosted option (Ollama) for zero marginal cost
- Provider competition drives down costs
- Flexibility to switch based on pricing changes

### Cost Calculation
**Decision**: Calculate cost only for paid API models, return $0 for self-hosted

**Rationale**:
- Self-hosted cost is GPU time + electricity (fixed costs)
- Per-token pricing only relevant for API providers
- Avoids confusion with estimated costs for self-hosted

## Security Considerations

### Input Validation
**Decision**: Sanitize all user input before evaluation

**Rationale**:
- Prevent prompt injection attacks
- Protect evaluator integrity
- Ensure consistent evaluation quality

### API Authentication
**Decision**: Currently no authentication (demo system)

**Production Recommendation**: 
- Add API key authentication
- Rate limiting per user
- Budget limits per organization

## Performance Optimization

### Parallel Evaluation
**Decision**: Run all evaluators in parallel with goroutines

**Rationale**:
- Evaluators are independent
- LLM calls are I/O bound
- 3-4x speedup over sequential
- Worker pool prevents resource exhaustion

### Database Indexing
**Decision**: Indexes on conversation_id, evaluator_type, status, created_at

**Rationale**:
- Common query patterns
- Dashboard aggregations
- Time-based filtering
- Status filtering for retries

## Future Extensibility

### Plugin Architecture
**Potential**: Evaluator interface allows custom evaluators

**Benefit**: Organizations can add domain-specific evaluators

### Multi-Model Evaluation
**Potential**: Run same evaluator with different models, compare results

**Benefit**: Ensemble evaluation for higher accuracy

### Real-time Evaluation
**Potential**: Stream-based evaluation for immediate feedback

**Trade-off**: More complex, but valuable for certain use cases

## Lessons Learned

### What Worked Well
- Modular evaluator design enables rapid iteration
- Windowing strategy balances quality and cost effectively
- Failure tracking catches issues early
- Token tracking provides actionable insights

### What Could Be Improved
- More sophisticated summarization (semantic compression)
- Evaluator ensembling for robustness
- Adaptive window sizes based on conversation structure
- ML-based confidence calibration

## References

- [Eval Reqs.md](Eval%20Reqs.md) - Original requirements
- [README.md](README.md) - System overview and usage
- [SCALING_STRATEGY.md](SCALING_STRATEGY.md) - Scaling approach

