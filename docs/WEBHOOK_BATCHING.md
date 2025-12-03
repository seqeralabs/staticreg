# Webhook Event Batching

## Overview

The container registry webhook handler uses asynchronous batch processing to handle high-throughput pull events efficiently. This document describes the architecture, configuration, and operational considerations.

## Architecture

### Problem Statement

Container registries can experience thousands of concurrent pull requests from CI/CD pipelines, Nextflow workflows, and customer workloads. The previous synchronous implementation processed database inserts within the HTTP handler, causing:

- Webhook latency proportional to database latency
- Database connection pool exhaustion during traffic spikes
- Registry webhook delivery timeouts if the database is slow
- Backpressure from metrics collection affecting registry performance

### Solution: BatchServiceAdapter

The `BatchServiceAdapter` implements asynchronous event processing with batching:

```
Webhook Request → Queue Event → Return 200 OK immediately
                       ↓
                  Background Worker
                       ↓
            Batch Events (100 or 5s)
                       ↓
              pgx.Batch Insert → PostgreSQL
```

**Key Features:**

1. **Non-blocking webhook handler**: Returns immediately after queuing the event
2. **Batch aggregation**: Collects up to 100 events or flushes every 5 seconds
3. **Efficient DB operations**: Uses `pgx.Batch` for pipelined inserts
4. **Graceful degradation**: Drops events if buffer is full (logs warning)
5. **Graceful shutdown**: Flushes pending events on application shutdown
6. **Observability**: Exposes metrics for monitoring queue health

## Configuration

Configure the batching behavior via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `STATICREG_METRICS_BATCH_SIZE` | 100 | Number of events to batch before flushing |
| `STATICREG_METRICS_FLUSH_INTERVAL` | 5s | Maximum time to wait before flushing partial batch |
| `STATICREG_METRICS_BUFFER_SIZE` | 10000 | Channel buffer size (max queued events) |

### Example Configuration

```bash
# High-throughput configuration
export STATICREG_METRICS_BATCH_SIZE=500
export STATICREG_METRICS_FLUSH_INTERVAL=10s
export STATICREG_METRICS_BUFFER_SIZE=50000

# Low-latency configuration
export STATICREG_METRICS_BATCH_SIZE=50
export STATICREG_METRICS_FLUSH_INTERVAL=1s
export STATICREG_METRICS_BUFFER_SIZE=5000
```

## Operational Metrics

The `BatchServiceAdapter` exposes the following metrics via `GetMetrics()`:

| Metric | Description |
|--------|-------------|
| `events_received` | Total events received since startup |
| `events_dropped` | Events dropped due to buffer overflow |
| `events_flushed` | Events successfully written to database |
| `batches_flushed` | Number of batch operations completed |
| `current_queue_size` | Current number of events in the queue |

### Monitoring

**Critical alerts:**

- `events_dropped > 0`: Buffer overflow occurring, increase `STATICREG_METRICS_BUFFER_SIZE` or scale database
- `current_queue_size` approaching `STATICREG_METRICS_BUFFER_SIZE`: Backlog building up
- `events_flushed` stagnant while `events_received` increasing: Database write issues

## Performance Characteristics

### Throughput Improvements

| Configuration | Synchronous (before) | Batched (after) | Improvement |
|---------------|---------------------|-----------------|-------------|
| 1000 req/s | ~500 DB ops/s | ~10-20 batches/s | **25-50x fewer DB operations** |
| Webhook latency | 10-50ms (DB dependent) | <1ms (queue only) | **10-50x faster response** |

### Latency Trade-offs

- **Webhook response**: Reduced from ~20ms to <1ms
- **Metrics visibility**: Delayed by up to `STATICREG_METRICS_FLUSH_INTERVAL` (default 5s)
- **Event loss risk**: Possible if buffer overflows (logged and tracked)

## Graceful Shutdown

On application shutdown (SIGTERM/SIGINT), the system:

1. Stops accepting new events (closes channel)
2. Waits up to 10 seconds for the worker to flush pending events
3. Logs final metrics (received, flushed, dropped counts)

**Shutdown timeout**: If the worker cannot flush within 10 seconds, remaining events may be lost. This is logged with an error.

## Database Considerations

### UPSERT Behavior

The batched implementation uses the same UPSERT query as the synchronous version:

```sql
INSERT INTO container_pull_metrics (pull_date, repo_name, tag, digest, architecture, pull_count, updated_at)
VALUES ($1, $2, $3, $4, $5, 1, NOW())
ON CONFLICT (pull_date, repo_name, tag, digest, architecture)
DO UPDATE SET
    pull_count = container_pull_metrics.pull_count + 1,
    updated_at = NOW()
```

This is safe for batching because:
- Pull counts are aggregated atomically per-row
- Race conditions between batches are handled by the `ON CONFLICT` clause
- No ordering dependency between events

### Connection Pooling

With batching, the number of concurrent database connections is **decoupled** from webhook request rate:

- **Before**: Up to `max_connections` webhooks could hit DB simultaneously
- **After**: Single worker thread batches all requests

This reduces database connection churn and prevents pool exhaustion.

## Error Handling

### Worker Panic Recovery

If the background worker panics, it automatically restarts:

```go
defer func() {
    if r := recover(); r != nil {
        a.Logger.Error("Worker panicked, attempting recovery", "panic", r)
        go a.worker()  // Restart worker
    }
}()
```

### Batch Insert Errors

If individual events fail during a batch insert:
- First 3 errors are logged with full details
- Successful events in the batch are still committed
- Failed events are counted but not retried (at-most-once delivery)

### Buffer Overflow

When the channel buffer is full:
- New events are dropped (non-blocking)
- Warning logged with repository details
- `events_dropped` metric incremented

## Migration from Synchronous Adapter

The synchronous `ServiceAdapter` is still available for backwards compatibility. To switch back:

```go
// In pkg/server/server.go
whService := webhook.NewServiceAdapter(log, dbPool)  // Synchronous
// whService := webhook.NewBatchServiceAdapter(log, dbPool)  // Batched (default)
```

Both implement the `WebhookService` interface, so they are drop-in replacements.

## Testing

Run the webhook tests:

```bash
go test ./pkg/webhook/... -v
```

Key test scenarios:
- Non-blocking behavior verification
- Graceful shutdown with pending events
- Metrics tracking accuracy
- Buffer overflow handling
- Idempotent close operations

## Production Recommendations

1. **Start with defaults**: The default configuration (batch size 100, flush interval 5s, buffer 10K) is suitable for most workloads
2. **Monitor `events_dropped`**: Set up alerts if this metric is non-zero
3. **Tune buffer size**: If drops occur, increase `STATICREG_METRICS_BUFFER_SIZE` before scaling the database
4. **Database scaling**: If queue depth grows consistently, consider:
   - Increasing database write capacity
   - Reducing `STATICREG_METRICS_FLUSH_INTERVAL` for faster draining
   - Horizontal scaling with read replicas (if reading metrics)

## Future Enhancements

Potential improvements for even higher scale:

1. **Message queue integration**: Replace channel with Kafka/SQS for at-least-once delivery guarantees
2. **Multi-worker pool**: Parallel batch workers for higher throughput
3. **Adaptive batching**: Dynamic batch size based on queue depth
4. **Event deduplication**: Hash-based dedup within batches to reduce DB load
5. **Metrics endpoint**: HTTP endpoint exposing Prometheus-compatible metrics

## References

- Original proposal: See COMP-679 consideration document
- pgx.Batch documentation: https://pkg.go.dev/github.com/jackc/pgx/v5#Batch
- PostgreSQL UPSERT: https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT