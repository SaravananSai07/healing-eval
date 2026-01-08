# Scaling Strategy

This document outlines how the Healing Eval system scales from current throughput to 10x and 100x loads.

## Current Baseline

**Estimated Capacity**: ~1,000 conversations/minute

**Components**:
- Single API server
- 10 concurrent workers
- PostgreSQL (single instance)
- Redis (single instance)

**Bottlenecks**:
- Worker pool size
- Database writes
- LLM API rate limits

## 10x Scale: 10,000 Conversations/Minute

**Target**: Handle 10K conversations/minute (~600K/hour)

### Infrastructure Changes

#### 1. Horizontal Worker Scaling
```
Current: 1 worker instance (10 goroutines)
Target:  10 worker instances (100 total goroutines)
```

**Implementation**:
- Deploy multiple worker pods/containers
- All connect to same Redis queue (consumer groups handle load balancing)
- No code changes required
- Auto-scaling based on queue depth

**Cost**: 10x compute cost for workers, but still modest (~$50-100/month for moderate instances)

#### 2. Database Optimization

**Read Replicas**:
- Add 2-3 read replicas for dashboard queries
- Route evaluations queries to replicas
- Writes still go to primary

**Connection Pooling**:
- Increase `DB_MAX_CONNS` to 100-200
- Use PgBouncer for connection pooling
- Reduces connection overhead

**Partitioning**:
- Partition `evaluations` table by `created_at` (monthly)
- Keeps working set small
- Faster queries and indexes

**Cost**: Additional $20-50/month per replica

#### 3. Redis Clustering

**Setup**:
- Redis Cluster with 3-6 nodes
- Sharding across nodes
- Replication for HA

**Alternative**: Redis Sentinel for failover

**Cost**: Minimal increase (~$10-20/month)

#### 4. API Server Scaling

```
Current: 1 API server
Target:  3-5 API servers behind load balancer
```

**Load Balancer**: Nginx, HAProxy, or cloud LB

**Benefits**:
- Handle API spikes
- Zero-downtime deployments
- Fault tolerance

**Cost**: ~$20-30/month for LB + instances

### LLM Provider Optimization

#### 1. Rate Limit Management

**Issue**: Most LLM providers have rate limits (e.g., OpenAI: 3500 RPM for GPT-4)

**Solutions**:
- Implement backoff/retry with exponential delays
- Use multiple API keys (if provider allows)
- Distribute across multiple providers
- Queue evaluations to respect rate limits

#### 2. Model Selection

**Strategy**: Use faster/cheaper models where appropriate

| Evaluator | Current Model | Optimized Model | Speedup | Cost Savings |
|-----------|---------------|-----------------|---------|--------------|
| Heuristic | None | None | N/A | N/A |
| LLM Judge | gpt-4o-mini | gpt-4o-mini | 1x | $0 |
| Tool Call | gpt-4o-mini | gpt-3.5-turbo | 2x | 67% |
| Coherence | gpt-4o-mini | gpt-3.5-turbo | 2x | 67% |

**Result**: 30-40% faster, 40-50% cheaper

#### 3. Caching Layer

**Implementation**: Redis cache for evaluation results

```
Key: hash(conversation + evaluator_type + version)
TTL: 7 days
```

**Hit Rate**: ~20-30% for duplicate/similar conversations

**Benefit**: 20-30% reduction in LLM calls

### Monitoring & Alerts

**Key Metrics**:
- Queue depth (Redis)
- Evaluation latency (P50, P95, P99)
- Error rate by evaluator
- Token usage per hour
- Cost per evaluation

**Alerting**:
- Queue depth > 10,000 → Scale workers
- Error rate > 5% → Investigate
- Hourly cost > $X → Alert finance

**Tools**: Prometheus + Grafana, or cloud-native monitoring

### Estimated Costs (10x Scale)

| Component | Current | 10x Scale | Delta |
|-----------|---------|-----------|-------|
| Compute (workers) | $10/mo | $100/mo | +$90 |
| Database | $25/mo | $100/mo | +$75 |
| Redis | $10/mo | $30/mo | +$20 |
| LLM API | $50/mo | $400/mo | +$350 |
| Load Balancer | $0 | $20/mo | +$20 |
| **Total** | **$95/mo** | **$650/mo** | **+$555** |

**Cost per conversation**: ~$0.0004 (acceptable for most use cases)

## 100x Scale: 100,000 Conversations/Minute

**Target**: Handle 100K conversations/minute (~6M/hour, ~144M/day)

This is enterprise scale. Major architectural changes required.

### Infrastructure Changes

#### 1. Dedicated Evaluation Clusters

**Architecture**:
- Separate clusters per evaluator type
- Independent scaling for each evaluator
- Specialized hardware (GPU for ML models)

**Benefit**: 
- Optimize each evaluator independently
- Isolated failures don't cascade

#### 2. Database Sharding

**Strategy**: Shard by conversation ID hash

```
Shard 0: conversation_id hash % 8 == 0
Shard 1: conversation_id hash % 8 == 1
...
Shard 7: conversation_id hash % 8 == 7
```

**Queries**: 
- Point queries: Single shard
- Aggregations: MapReduce across shards

**Cost**: $500-1000/month for sharded DB cluster

#### 3. Multi-Region Deployment

**Setup**:
- API in multiple regions (US-East, US-West, EU, Asia)
- Workers colocated with LLM providers (reduce latency)
- Global Redis for queue, regional caches

**Benefit**:
- Lower latency
- Regulatory compliance (data locality)
- HA across regions

#### 4. Batch Inference

**LLM Optimization**: Batch multiple evaluations into single API call

**Example**: Evaluate 10 conversations simultaneously
```
Prompt: "Evaluate these 10 conversations: [batch of 10]"
```

**Benefit**:
- Amortize API overhead
- Throughput increase (though latency increases slightly)
- Some providers offer batch discounts

**Trade-off**: More complex prompt engineering

#### 5. Self-Hosted LLM Cluster

**Setup**: Deploy open-source models (Llama, Mistral) on GPU cluster

**Hardware**: 
- 10-20 A100 GPUs
- vLLM or TensorRT for inference optimization
- Load balancer across GPUs

**Cost**: $10K-20K/month (but $0 per call)

**Break-even**: ~20M evaluations/month

**Benefit**: Predictable costs, no rate limits

### Advanced Optimizations

#### 1. Adaptive Windowing

**Current**: Fixed window sizes per evaluator

**Optimized**: Dynamic windows based on conversation structure

**Algorithm**:
```
if conversation has repeated patterns:
    window_size = smaller
elif conversation has high information density:
    window_size = larger
```

**Benefit**: 10-20% additional token savings

#### 2. Semantic Summarization

**Current**: Simple truncation + basic summaries

**Optimized**: Semantic compression with embedding models

**Approach**:
- Extract key entities, intents, actions
- Cluster similar turns
- Preserve only novel information

**Benefit**: 30-40% additional token savings

#### 3. Evaluator Ensembles

**Approach**: Run multiple models, ensemble results

**Example**:
- LLM Judge: GPT-4, Claude-3, Llama-3 (vote/average)
- Confidence: Agreement between models

**Benefit**: Higher accuracy, detects model-specific blind spots

**Cost**: 3x LLM calls, but can use cheaper models

#### 4. Streaming Evaluations

**Architecture**: Evaluate as conversation progresses (not batch)

**Benefit**:
- Real-time feedback
- Catch issues early
- Incremental evaluation (cheaper)

**Trade-off**: More complex, webhooks/websockets required

### Database Architecture (100x Scale)

**Technology Shift**: Consider TimescaleDB or ClickHouse for time-series data

**Rationale**:
- Evaluations are append-only, time-series
- Optimized for analytical queries
- Better compression
- Faster aggregations

**Migration**: Dual-write to both PostgreSQL and TimescaleDB, then cutover

### Cost Optimization at Scale

#### 1. Spot Instances for Workers

**Savings**: 60-80% on compute costs

**Implementation**: 
- Graceful shutdown on termination signal
- Requeue in-flight jobs
- Mix of spot + on-demand (80/20)

#### 2. Reserved Instances for Database

**Savings**: 30-40% on database costs

**Commitment**: 1-3 year reservation

#### 3. LLM API Negotiation

**Approach**: Enterprise contracts with volume discounts

**Potential**: 20-50% discount at high volumes

#### 4. Storage Tiering

**Strategy**: Move old data to S3/glacier

- Hot data: Last 7 days (PostgreSQL)
- Warm data: 8-90 days (S3 + cache)
- Cold data: 90+ days (S3 Glacier)

**Savings**: 70-80% on storage costs

### Estimated Costs (100x Scale)

| Component | 10x Scale | 100x Scale | Delta |
|-----------|-----------|------------|-------|
| Compute | $100/mo | $2,000/mo | +$1,900 |
| Database | $100/mo | $1,000/mo | +$900 |
| Redis | $30/mo | $200/mo | +$170 |
| LLM API | $400/mo | $15,000/mo | +$14,600 |
| Self-hosted LLM | $0 | $10,000/mo | +$10,000 |
| Network | $0 | $500/mo | +$500 |
| **Total** | **$650/mo** | **$28,700/mo** | **+$28,050** |

**With optimizations** (caching, self-hosted models, spot instances):
**Optimized Total**: ~$15,000-20,000/month

**Cost per conversation**: ~$0.0003 (cheaper due to economies of scale)

## Key Takeaways

### Linear Scaling (10x)
- Horizontal scaling works out of the box
- Minimal architectural changes
- Cost scales roughly linearly

### Non-Linear Scaling (100x)
- Requires architectural changes
- Self-hosted infrastructure becomes cost-effective
- Database sharding essential
- Cost per unit decreases due to efficiencies

### Critical Path for Scaling
1. **First bottleneck**: Worker pool size → Scale workers
2. **Second bottleneck**: LLM rate limits → Caching + multiple providers
3. **Third bottleneck**: Database writes → Sharding + replicas
4. **Fourth bottleneck**: Cost → Self-hosted models + optimizations

## Monitoring for Scaling Decisions

### When to Scale Workers
```
if avg_queue_depth_5min > 1000:
    add_workers(count=5)
```

### When to Scale Database
```
if db_cpu > 80% for 10 minutes:
    add_read_replica()
```

### When to Self-Host Models
```
if monthly_llm_cost > $10,000:
    evaluate_self_hosting()
```

## Conclusion

The current architecture is designed for horizontal scalability. Reaching 10x scale requires infrastructure scaling but no code changes. Reaching 100x scale requires architectural evolution but the foundation is solid.

The modular design allows scaling different components independently, and the queue-based architecture provides natural backpressure during load spikes.

