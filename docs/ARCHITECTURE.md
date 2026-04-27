# Staticreg Architecture

## Overview

Staticreg is a web application that provides a user-friendly interface for browsing OCI (Open Container Initiative) registries. It integrates with Docker Distribution registries to display container images and collect usage metrics.

## Core Components

### 1. Web Server (`pkg/server`)

The main HTTP server built with the Gin web framework. It provides:

- Static web interface for browsing container images
- Image listing and tag browsing endpoints
- Image inspection and security scanning integration
- Webhook endpoint for receiving registry events

**Key Files:**
- `server.go` - Main server initialization and routing

### 2. Registry Client (`pkg/registry`)

Handles communication with OCI-compliant registries:

- Fetches image catalogs via `/v2/_catalog` endpoint
- Retrieves image manifests and metadata
- Supports authentication with registry credentials
- Async registry operations with caching (`pkg/registry/async`)

### 3. Webhook Integration (`pkg/webhook`)

Processes Docker Distribution webhook events for real-time metrics:

**Event Types:**
- `events.go` - Defines event structures for Docker Distribution notifications
  - `DistributionEvent` - Core event structure
  - `DistributionEventTarget` - Image/artifact information (repository, tag, digest, media type)
  - `DistributionEventRequest` - Request metadata including user agent (for architecture detection)
  - `DistributionEventActor` - User/service that triggered the event

**Event Processing:**
- Receives webhook notifications from Docker Distribution
- Filters for manifest pull events (actual container pulls)
- Extracts architecture from Docker client user agent
- Excludes events from staticreg itself to prevent counting internal fetches
- Aggregates metrics in PostgreSQL using UPSERT pattern

**Architecture Pattern:**
- `WebhookService` interface defines the contract for webhook operations
- `ServiceAdapter` implements the interface, coordinating between database and event processing
- Gracefully handles database unavailability (continues operation without persistence)

### 4. Database Layer (`pkg/db`)

PostgreSQL integration for storing container pull/push metrics:

**Features:**
- Connection pool management with configurable limits
- Automatic schema initialization on startup
- Graceful degradation when database is unavailable
- Schema validation on startup

**Schema Isolation:**
- All StaticReg tables live in a dedicated postgres schema (default: `staticreg`),
  never `public`. The schema name is configurable via `STATICREG_DB_SCHEMA` and
  is set as `search_path` on every pooled connection, so unqualified queries
  resolve there transparently.
- This isolation is what makes a single postgres instance safely usable for
  multiple StaticReg deployments (e.g. `staticreg_prod`, `staticreg_staging`)
  and for shared databases owned by other applications.

**Schema Management (`pkg/sql`):**
- Migrations are versioned `.sql` files under `pkg/sql/migrations/`, embedded
  via `embed.FS` and applied with [pressly/goose](https://github.com/pressly/goose)
  on startup.
- The `goose_db_version` table lives inside the dedicated schema and tracks
  applied versions; reruns are no-ops once all migrations are up.
- See [`docs/POSTGRES_OPERATIONS.md`](POSTGRES_OPERATIONS.md) for adding new
  migrations and rolling back.

**Health & Observability:**
- `GET /healthz` — process liveness, no DB dependency.
- `GET /healthz/db` — readiness; pings the pool with a 750ms timeout, returns
  503 if the pool is unconfigured or the ping fails.
- `GET /metrics/db` — JSON snapshot of `pgxpool.Stat()` (acquired/idle/max
  conns, acquire counts, lifetime destroy counts).
- `GET /metrics/webhook` — JSON snapshot of the batched webhook adapter's
  counters (events received/dropped/flushed, batch count, queue size).

**Database Schema:**
- `container_pull_metrics` - Aggregated pull metrics table
  - Columns: `id`, `pull_date`, `repo_name`, `tag`, `digest`, `architecture`, `pull_count`, `created_at`, `updated_at`
  - Primary Key: `id` (BIGSERIAL)
  - Unique Constraint: `(pull_date, repo_name, tag, digest, architecture)`
  - Indexes:
    - `idx_pull_date` - Optimized for date-based queries
    - `idx_repo_date` - Optimized for repository + date queries
    - `idx_repo_arch_date` - Optimized for repository + architecture + date queries

**Data Aggregation:**
- Uses PostgreSQL UPSERT (INSERT ... ON CONFLICT DO UPDATE)
- Aggregates metrics by date, repository, tag, digest, and architecture
- Increments `pull_count` for duplicate entries
- Significantly reduces database size compared to storing individual events

### 5. Configuration (`pkg/cfg`)

Application configuration management:

- Registry connection details
- Database connection strings
- Server settings (port, cache duration)
- Wave server integration (for security scanning)

## Data Flow

### Image Browsing Flow

```
User Browser → Web Server → Registry Client → OCI Registry
                    ↓
              Template Rendering
                    ↓
              HTML Response
```

### Webhook Event Flow

```
Docker Distribution Registry
        ↓ (HTTP POST)
    Webhook Endpoint
        ↓
Event Validation & Filtering
        ↓
  WebhookService
        ↓
  Database (PostgreSQL)
```

## Event Processing

When a container is pulled from a Docker Distribution registry:

1. **Event Reception:** Docker Distribution sends a webhook notification with multiple events (blob pulls, manifest pulls)
2. **Event Filtering:** Staticreg webhook endpoint receives and filters events:
   - Identifies manifest pull events using `IsManifestPull()`
   - Excludes events from staticreg itself using `IsFromStaticReg()` (prevents counting internal fetches)
   - Only processes events with valid repository and tag information
3. **Architecture Detection:** Extracts CPU architecture from Docker client user agent
   - Parses user agent for `arch/xxx` pattern (e.g., `arch/amd64`, `arch/arm64`)
   - Defaults to `not-specified` if architecture cannot be determined
4. **Metrics Aggregation:** Events are aggregated in PostgreSQL:
   - Pull date (UTC)
   - Repository name
   - Image tag
   - Manifest digest (for tracking tag updates)
   - Architecture
   - Pull count (incremented via UPSERT)
5. **Database Handling:** Gracefully handles database unavailability
   - If database is not configured, events are received but not stored
   - Application continues to function for registry browsing

## Registry Webhook Configuration

Docker Distribution must be configured to send webhooks to staticreg:

```yaml
notifications:
  endpoints:
    - name: staticreg-webhook
      url: http://staticreg-host:8093/api/webhook/registry
      timeout: 5s
      threshold: 3
      backoff: 1s
```

## Environment Variables

### Required
- `STATICREG_REGISTRY_HOSTNAME` - OCI registry hostname

### Optional
- `STATICREG_DB_URL` - PostgreSQL connection string for metrics tracking
  - Format: `postgresql://[user[:password]@][host][:port][/dbname]`
  - If not set, application runs without metrics persistence
- `STATICREG_REGISTRY_USER` - Registry authentication username
- `STATICREG_REGISTRY_PASSWORD` - Registry authentication password
- `WAVE_SERVER_URL` - Wave server URL for security scanning
- `PORT` - HTTP server port (default: 8093)

## Technology Stack

- **Language:** Go 1.24.3+
- **Web Framework:** Gin
- **Database:** PostgreSQL 18+ (tested with 18.1, compatible with 15+)
- **Database Driver:** pgx/v5
- **Container Runtime:** Docker
- **Registry Protocol:** OCI Distribution Spec / Docker Registry HTTP API V2

## Design Decisions

### Database Schema Design

**Why Aggregated Metrics?**
- Original approach stored individual events, leading to rapid database growth
- Current approach aggregates metrics by (date, repo, tag, digest, architecture)
- Benefits:
  - Dramatically reduced storage requirements
  - Faster query performance
  - Simplified analytics queries
  - Efficient for time-series analysis

**Why Include Digest?**
- Tags can be updated to point to different image versions
- Digest provides immutable reference to specific image version
- Enables tracking of tag updates over time
- Example: Track when `latest` tag moved from one version to another

**Why Extract Architecture?**
- Enables platform-specific analytics
- Useful for understanding multi-arch image usage
- Helps identify platform adoption trends

### Graceful Degradation

**Database Optional Design:**
- Staticreg can run without database for registry browsing
- Webhook events are accepted but not persisted if DB unavailable
- Connection failures logged as warnings, not errors
- Application continues to provide value even without metrics

**Benefits:**
- Easier deployment and testing
- Reduced operational complexity
- Progressive enhancement approach

## Future Enhancements

- Metrics dashboard UI
- REST API for querying pull/push statistics
- Real-time metrics streaming
- Webhook authentication/verification (signature-based)
- Multi-registry support
- Data retention policies (automatic archiving/cleanup)
- Push event tracking
- Export to time-series databases (Prometheus, InfluxDB)