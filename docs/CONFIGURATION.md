# Configuration Guide

This document describes all configuration options available for the Static Registry application.

## Environment Variables

### Webhook Batching Configuration

Configuration for the asynchronous webhook event processing system. These settings control how container pull events are batched and written to the database.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `STATICREG_METRICS_BATCH_SIZE` | integer | `100` | Number of events to accumulate before flushing to the database. Higher values reduce database load but increase memory usage and potential data loss on crashes. Recommended range: 50-500. |
| `STATICREG_METRICS_FLUSH_INTERVAL` | duration | `5s` | Maximum time to wait before flushing a partial batch. Ensures events are written even during low traffic. Accepts Go duration format (e.g., `5s`, `1m`, `500ms`). Recommended range: 1s-30s. |
| `STATICREG_METRICS_BUFFER_SIZE` | integer | `10000` | Size of the event channel buffer. This is the maximum number of events that can be queued in memory. If the buffer fills up, new events will be dropped and logged. Increase this if you see dropped events. Recommended range: 1000-100000. |

#### Example Configurations

**High-Throughput Setup** (handles bursts, tolerates higher latency):
```bash
export STATICREG_METRICS_BATCH_SIZE=500
export STATICREG_METRICS_FLUSH_INTERVAL=10s
export STATICREG_METRICS_BUFFER_SIZE=50000
```

**Low-Latency Setup** (faster metric visibility, more DB operations):
```bash
export STATICREG_METRICS_BATCH_SIZE=50
export STATICREG_METRICS_FLUSH_INTERVAL=1s
export STATICREG_METRICS_BUFFER_SIZE=5000
```

**Default Setup** (balanced):
```bash
# No need to set - these are the defaults
export STATICREG_METRICS_BATCH_SIZE=100
export STATICREG_METRICS_FLUSH_INTERVAL=5s
export STATICREG_METRICS_BUFFER_SIZE=10000
```

### Database Configuration

Configuration for PostgreSQL connection and pooling.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `STATICREG_DB_URL` | string | *(none)* | PostgreSQL connection string. Format: `postgres://user:password@host:port/dbname?sslmode=disable`. If not set, the application runs without database metrics collection. Takes priority over individual `STATICREG_DB_*` variables. |
| `STATICREG_DB_HOST` | string | *(required)* | PostgreSQL host. Alternative to `STATICREG_DB_URL`. Required if using individual variables. |
| `STATICREG_DB_PORT` | string | *(optional)* | PostgreSQL port. Alternative to `STATICREG_DB_URL`. Defaults to PostgreSQL's default port if not specified. |
| `STATICREG_DB_USER` | string | *(required)* | PostgreSQL username. Alternative to `STATICREG_DB_URL`. Required if using individual variables. |
| `STATICREG_DB_PASSWORD` | string | *(optional)* | PostgreSQL password. Alternative to `STATICREG_DB_URL`. |
| `STATICREG_DB_NAME` | string | *(required)* | PostgreSQL database name. Alternative to `STATICREG_DB_URL`. Required if using individual variables. |
| `STATICREG_DB_SSLMODE` | string | *(optional)* | PostgreSQL SSL mode. Options: `disable`, `require`, `verify-ca`, `verify-full`. Alternative to `STATICREG_DB_URL`. |


### Registry Configuration

Target Docker registry settings.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `STATICREG_REGISTRY_HOSTNAME` | string | *(required)* | Hostname of the target Docker registry to proxy. Example: `registry.hub.docker.com` |


## Configuration Priority

Configuration values are resolved in the following order (highest to lowest priority):

1. Command-line flags
2. Environment variables
3. Configuration file (if implemented)
4. Default values

## Example: Complete Configuration

### Using Connection URL

```bash
#!/bin/bash

# Database (using connection URL)
export STATICREG_DB_URL="postgres://staticreg:password@db.example.com:5432/staticreg?sslmode=require"

# Webhook Batching
export STATICREG_METRICS_BATCH_SIZE=200
export STATICREG_METRICS_FLUSH_INTERVAL=10s
export STATICREG_METRICS_BUFFER_SIZE=20000

# Target Registry
export STATICREG_REGISTRY_HOSTNAME="registry.example.com"

# Run the server
./staticreg serve
```

### Using Individual Variables

```bash
#!/bin/bash

# Database (using individual variables)
export STATICREG_DB_HOST="localhost"
export STATICREG_DB_PORT="5432"
export STATICREG_DB_USER="staticreg"
export STATICREG_DB_PASSWORD="password"
export STATICREG_DB_NAME="staticreg"
export STATICREG_DB_SSLMODE="require"

# Webhook Batching
export STATICREG_METRICS_BATCH_SIZE=200
export STATICREG_METRICS_FLUSH_INTERVAL=10s
export STATICREG_METRICS_BUFFER_SIZE=20000

# Target Registry
export STATICREG_REGISTRY_HOSTNAME="registry.example.com"

# Run the server
./staticreg serve
```

## Docker Compose Example

### Using Connection URL

```yaml
version: '3.8'

services:
  staticreg:
    image: staticreg:latest
    ports:
      - "8093:8093"
    environment:
      # Database (using connection URL)
      STATICREG_DB_URL: "postgres://staticreg:password@postgres:5432/staticreg"

      # Webhook Batching
      STATICREG_METRICS_BATCH_SIZE: "200"
      STATICREG_METRICS_FLUSH_INTERVAL: "10s"
      STATICREG_METRICS_BUFFER_SIZE: "20000"

      # Target Registry
      STATICREG_REGISTRY_HOSTNAME: "registry.example.com"

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

### Using Individual Variables

```yaml
version: '3.8'

services:
  staticreg:
    image: staticreg:latest
    ports:
      - "8093:8093"
    environment:
      # Database (using individual variables)
      STATICREG_DB_HOST: "postgres"
      STATICREG_DB_PORT: "5432"
      STATICREG_DB_USER: "staticreg"
      STATICREG_DB_PASSWORD: "password"
      STATICREG_DB_NAME: "staticreg"
      STATICREG_DB_SSLMODE: "disable"

      # Webhook Batching
      STATICREG_METRICS_BATCH_SIZE: "200"
      STATICREG_METRICS_FLUSH_INTERVAL: "10s"
      STATICREG_METRICS_BUFFER_SIZE: "20000"

      # Target Registry
      STATICREG_REGISTRY_HOSTNAME: "registry.example.com"

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
  STATICREG_METRICS_BATCH_SIZE: "200"
  STATICREG_METRICS_FLUSH_INTERVAL: "10s"
  STATICREG_METRICS_BUFFER_SIZE: "20000"

  # Target Registry
  STATICREG_REGISTRY_HOSTNAME: "registry.example.com"

---
apiVersion: v1
kind: Secret
metadata:
  name: staticreg-secrets
type: Opaque
stringData:
  STATICREG_DB_URL: "postgres://staticreg:password@postgres:5432/staticreg"
```

## Performance Tuning

### For High Traffic (>1000 req/s)

```bash
# Increase batching to reduce DB load
export STATICREG_METRICS_BATCH_SIZE=500
export STATICREG_METRICS_FLUSH_INTERVAL=10s
export STATICREG_METRICS_BUFFER_SIZE=50000

```

### For Low Latency (real-time metrics)

```bash
# Flush more frequently for faster visibility
export STATICREG_METRICS_BATCH_SIZE=50
export STATICREG_METRICS_FLUSH_INTERVAL=1s
export STATICREG_METRICS_BUFFER_SIZE=5000

```

### For Resource-Constrained Environments

```bash
# Reduce memory footprint
export STATICREG_METRICS_BATCH_SIZE=50
export STATICREG_METRICS_BUFFER_SIZE=1000

```

## Related Documentation
- [Webhook Batching Architecture](WEBHOOK_BATCHING.md)