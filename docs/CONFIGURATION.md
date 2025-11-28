# Configuration Guide

This document describes all configuration options available for the Static Registry application.

## Environment Variables

### Webhook Batching Configuration

Configuration for the asynchronous webhook event processing system. These settings control how container pull events are batched and written to the database.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `METRICS_BATCH_SIZE` | integer | `100` | Number of events to accumulate before flushing to the database. Higher values reduce database load but increase memory usage and potential data loss on crashes. Recommended range: 50-500. |
| `METRICS_FLUSH_INTERVAL` | duration | `5s` | Maximum time to wait before flushing a partial batch. Ensures events are written even during low traffic. Accepts Go duration format (e.g., `5s`, `1m`, `500ms`). Recommended range: 1s-30s. |
| `METRICS_BUFFER_SIZE` | integer | `10000` | Size of the event channel buffer. This is the maximum number of events that can be queued in memory. If the buffer fills up, new events will be dropped and logged. Increase this if you see dropped events. Recommended range: 1000-100000. |

#### Example Configurations

**High-Throughput Setup** (handles bursts, tolerates higher latency):
```bash
export METRICS_BATCH_SIZE=500
export METRICS_FLUSH_INTERVAL=10s
export METRICS_BUFFER_SIZE=50000
```

**Low-Latency Setup** (faster metric visibility, more DB operations):
```bash
export METRICS_BATCH_SIZE=50
export METRICS_FLUSH_INTERVAL=1s
export METRICS_BUFFER_SIZE=5000
```

**Default Setup** (balanced):
```bash
# No need to set - these are the defaults
export METRICS_BATCH_SIZE=100
export METRICS_FLUSH_INTERVAL=5s
export METRICS_BUFFER_SIZE=10000
```

### Database Configuration

Configuration for PostgreSQL connection and pooling.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DATABASE_URL` | string | *(none)* | PostgreSQL connection string. Format: `postgres://user:password@host:port/dbname?sslmode=disable`. If not set, the application runs without database metrics collection. |
| `PGHOST` | string | `localhost` | PostgreSQL host. Alternative to `DATABASE_URL`. |
| `PGPORT` | string | `5432` | PostgreSQL port. Alternative to `DATABASE_URL`. |
| `PGUSER` | string | `postgres` | PostgreSQL username. Alternative to `DATABASE_URL`. |
| `PGPASSWORD` | string | *(none)* | PostgreSQL password. Alternative to `DATABASE_URL`. |
| `PGDATABASE` | string | `staticreg` | PostgreSQL database name. Alternative to `DATABASE_URL`. |
| `PGSSLMODE` | string | `disable` | PostgreSQL SSL mode. Options: `disable`, `require`, `verify-ca`, `verify-full`. |

### Server Configuration

HTTP server and application behavior settings.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `BIND_ADDR` | string | `127.0.0.1:8093` | Address and port for the HTTP server to bind to. Use `0.0.0.0:8093` to listen on all interfaces. |
| `CACHE_DURATION` | duration | `1m` | How long to keep generated HTML pages in cache before expiring. Use `0` to never expire. Accepts Go duration format. |
| `REFRESH_INTERVAL` | duration | `15m` | How long to wait before fetching fresh data from the target registry. Controls how often the registry is polled for updates. |
| `REQUESTS_PER_SECOND` | integer | `1` | Rate limit for requests to the target registry. Prevents overwhelming the upstream registry. Minimum value: 1. |
| `IGNORED_USER_AGENTS` | string array | *(none)* | Comma-separated list of user agent substrings to ignore. Requests from matching user agents receive an empty 200 OK response. Useful for health checks and monitoring tools. |

### Registry Configuration

Target Docker registry settings.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `REGISTRY_HOSTNAME` | string | *(required)* | Hostname of the target Docker registry to proxy. Example: `registry.hub.docker.com` |
| `REGISTRY_USERNAME` | string | *(none)* | Username for authenticating with the target registry. Required if the registry requires authentication. |
| `REGISTRY_PASSWORD` | string | *(none)* | Password for authenticating with the target registry. Required if the registry requires authentication. |
| `REGISTRY_INSECURE` | boolean | `false` | Allow insecure HTTPS connections to the target registry. Set to `true` for self-signed certificates. |

### Logging Configuration

Application logging settings.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `LOG_LEVEL` | string | `info` | Minimum log level to output. Options: `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | string | `json` | Log output format. Options: `json`, `text`. JSON format is recommended for production. |

## Command-Line Flags

Most environment variables can also be set via command-line flags when using the `serve` command:

```bash
staticreg serve \
  --bind-addr 0.0.0.0:8093 \
  --cache-duration 5m \
  --refresh-interval 10m \
  --requests-per-second 5 \
  --ignored-user-agent "HealthCheck/1.0" \
  --ignored-user-agent "Prometheus/2.0"
```

Run `staticreg serve --help` for a complete list of available flags.

## Configuration Priority

Configuration values are resolved in the following order (highest to lowest priority):

1. Command-line flags
2. Environment variables
3. Configuration file (if implemented)
4. Default values

## Example: Complete Configuration

```bash
#!/bin/bash

# Database
export DATABASE_URL="postgres://staticreg:password@db.example.com:5432/staticreg?sslmode=require"

# Webhook Batching
export METRICS_BATCH_SIZE=200
export METRICS_FLUSH_INTERVAL=10s
export METRICS_BUFFER_SIZE=20000

# Server
export BIND_ADDR="0.0.0.0:8093"
export CACHE_DURATION="5m"
export REFRESH_INTERVAL="10m"
export REQUESTS_PER_SECOND=5

# Target Registry
export REGISTRY_HOSTNAME="registry.example.com"
export REGISTRY_USERNAME="readonly"
export REGISTRY_PASSWORD="secretpassword"

# Logging
export LOG_LEVEL="info"
export LOG_FORMAT="json"

# Run the server
./staticreg serve
```

## Docker Compose Example

```yaml
version: '3.8'

services:
  staticreg:
    image: staticreg:latest
    ports:
      - "8093:8093"
    environment:
      # Database
      DATABASE_URL: "postgres://staticreg:password@postgres:5432/staticreg"

      # Webhook Batching
      METRICS_BATCH_SIZE: "200"
      METRICS_FLUSH_INTERVAL: "10s"
      METRICS_BUFFER_SIZE: "20000"

      # Server
      BIND_ADDR: "0.0.0.0:8093"
      CACHE_DURATION: "5m"
      REFRESH_INTERVAL: "10m"
      REQUESTS_PER_SECOND: "5"

      # Target Registry
      REGISTRY_HOSTNAME: "registry.example.com"
      REGISTRY_USERNAME: "readonly"
      REGISTRY_PASSWORD: "secretpassword"

      # Logging
      LOG_LEVEL: "info"
      LOG_FORMAT: "json"

    depends_on:
      - postgres

  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: staticreg
      POSTGRES_PASSWORD: password
      POSTGRES_DB: staticreg
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: staticreg-config
data:
  # Webhook Batching
  METRICS_BATCH_SIZE: "200"
  METRICS_FLUSH_INTERVAL: "10s"
  METRICS_BUFFER_SIZE: "20000"

  # Server
  BIND_ADDR: "0.0.0.0:8093"
  CACHE_DURATION: "5m"
  REFRESH_INTERVAL: "10m"
  REQUESTS_PER_SECOND: "5"

  # Target Registry
  REGISTRY_HOSTNAME: "registry.example.com"
  REGISTRY_USERNAME: "readonly"

  # Logging
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"

---
apiVersion: v1
kind: Secret
metadata:
  name: staticreg-secrets
type: Opaque
stringData:
  DATABASE_URL: "postgres://staticreg:password@postgres:5432/staticreg"
  REGISTRY_PASSWORD: "secretpassword"
```

## Performance Tuning

### For High Traffic (>1000 req/s)

```bash
# Increase batching to reduce DB load
export METRICS_BATCH_SIZE=500
export METRICS_FLUSH_INTERVAL=10s
export METRICS_BUFFER_SIZE=50000

# Increase cache duration to reduce upstream load
export CACHE_DURATION="10m"
export REFRESH_INTERVAL="30m"
export REQUESTS_PER_SECOND=10
```

### For Low Latency (real-time metrics)

```bash
# Flush more frequently for faster visibility
export METRICS_BATCH_SIZE=50
export METRICS_FLUSH_INTERVAL=1s
export METRICS_BUFFER_SIZE=5000

# Shorter cache for fresher data
export CACHE_DURATION="30s"
export REFRESH_INTERVAL="5m"
```

### For Resource-Constrained Environments

```bash
# Reduce memory footprint
export METRICS_BATCH_SIZE=50
export METRICS_BUFFER_SIZE=1000

# Reduce cache size
export CACHE_DURATION="1m"
```

## Monitoring Configuration Health

Check the application logs on startup for configuration values:

```json
{
  "level": "info",
  "msg": "Started batched webhook processor",
  "batchSize": 100,
  "flushInterval": "5s",
  "bufferSize": 10000
}
```

Monitor these metrics to validate configuration effectiveness:
- `events_dropped` should be 0 (increase `METRICS_BUFFER_SIZE` if not)
- `current_queue_size` should stay well below `METRICS_BUFFER_SIZE`
- Batch flush operations should occur regularly

## Troubleshooting

### Events Being Dropped
**Symptom**: `events_dropped` metric is increasing

**Solution**: Increase `METRICS_BUFFER_SIZE` or increase database write capacity

```bash
export METRICS_BUFFER_SIZE=50000
```

### Metrics Lag Too High
**Symptom**: Metrics appear in database several seconds after events

**Solution**: Reduce `METRICS_FLUSH_INTERVAL`

```bash
export METRICS_FLUSH_INTERVAL=1s
```

### High Database Load
**Symptom**: Database CPU/IO is high

**Solution**: Increase `METRICS_BATCH_SIZE` to reduce flush frequency

```bash
export METRICS_BATCH_SIZE=500
```

### Memory Usage Too High
**Symptom**: Application using excessive memory

**Solution**: Reduce `METRICS_BUFFER_SIZE`

```bash
export METRICS_BUFFER_SIZE=5000
```

## Related Documentation
- [Webhook Batching Architecture](WEBHOOK_BATCHING.md)