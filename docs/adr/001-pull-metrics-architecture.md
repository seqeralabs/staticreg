# ADR-001: Pull Metrics Architecture

**Status:** Proposed
**Date:** 2026-01-07
**Deciders:** TBD
**Technical Story:** Review of Postgres analytics introduced in commit bbd659ad3816bfe4fdebe7896a030d44599df7be

## Context and Problem Statement

StaticReg needs to track container pull metrics from the Distribution registry. The current implementation uses Distribution's webhook notification system to send pull events to StaticReg, which then stores them in PostgreSQL.

This approach has several issues:

1. **No query pattern defined** - Indexes were created speculatively without defined query patterns or an API to read the data
2. **Manual migrations** - Schema is applied via raw SQL at startup with no versioning or migration path
3. **Schema mismatch** - Table columns (`id`, `created_at`) don't match the aggregation/counting intent
4. **Architectural coupling** - StaticReg (a read-only frontend) is receiving write traffic for metrics
5. **Data loss** - Valuable event data (client IP, full user-agent, actor) is discarded during aggregation

## Decision Drivers

- Reliability - all pulls should be captured
- Simplicity - minimize moving parts and coupling
- Separation of concerns - StaticReg is a UI, not a metrics pipeline
- Standard tooling - use battle-tested solutions
- Query flexibility - support future analytics needs

## Considered Options

### Option A: Access Log Parsing (Recommended)

Use Distribution's built-in access logging with a log processor (Vector/Fluentbit) to parse and store metrics.

```
Distribution (stdout) → Vector/Fluentbit → PostgreSQL/ClickHouse
```

**How it works:**

Distribution outputs Apache Combined Log Format by default:

```
192.168.1.100 - - [07/Jan/2026:10:00:00 +0000] "GET /v2/library/nginx/manifests/latest HTTP/1.1" 200 1573 "-" "docker/20.10.7 go/go1.16.4 os/linux arch/amd64"
```

A log processor filters for manifest GET requests with status 200 and extracts:

- Repository name from URL path
- Tag/digest from URL path
- Architecture from user-agent
- Client IP
- Timestamp

**Pros:**

- No Distribution configuration changes (access logging is enabled by default)
- No webhook complexity or delivery concerns
- Captures ALL requests, not just events Distribution decides to emit
- Richer data available (full user-agent, response size, exact timestamps)
- Decoupled from StaticReg entirely
- Standard tooling (Vector, Fluentbit, Loki) is production-grade
- Can replay logs if needed

**Cons:**

- Requires deploying a log processor
- Slightly delayed (batch processing vs real-time)
- Need to parse log format

### Option B: Distribution Webhook (Current Implementation)

Keep the current webhook-based approach but fix the implementation issues.

```
Distribution → webhook → StaticReg → PostgreSQL
```

**Pros:**

- Already partially implemented
- Real-time event delivery
- Structured JSON payload

**Cons:**

- Webhook delivery is not guaranteed (network issues, StaticReg downtime)
- Couples StaticReg to write path
- Limited data in webhook payload vs access logs
- Requires Distribution configuration
- StaticReg becomes a critical path component

### Option C: Harbor-Style Interception

Put StaticReg in the request path, intercepting manifest requests before proxying to Distribution.

```
Client → StaticReg → Distribution
           ↓
       PostgreSQL
```

**Pros:**

- 100% capture rate
- Full request context available
- How Harbor does it

**Cons:**

- Major architectural change
- StaticReg becomes critical infrastructure
- Adds latency to every pull
- Requires StaticReg to handle registry protocol

### Option D: Prometheus Metrics from Distribution

Enable Distribution's built-in Prometheus endpoint and scrape metrics.

```yaml
http:
  debug:
    addr: "0.0.0.0:5001"
    prometheus:
      enabled: true
```

**Pros:**

- Built into Distribution
- Standard Prometheus ecosystem

**Cons:**

- Does NOT expose per-repository pull counts
- Only operational metrics (storage latency, notification queue, cache hits)
- Cannot answer "how many times was image X pulled"

### Option E: OpenTelemetry Tracing

Use Distribution's built-in OpenTelemetry support to send traces to a collector.

**Pros:**

- Built into Distribution
- Rich request context in spans

**Cons:**

- Overkill for simple pull counts
- Requires trace aggregation infrastructure
- Complex query patterns

## Decision Outcome

**Chosen option: Option A (Access Log Parsing)**

This option provides the best balance of reliability, simplicity, and separation of concerns. It:

- Requires no changes to Distribution or StaticReg
- Uses standard, production-grade tooling
- Captures all requests with rich context
- Keeps StaticReg as a pure read-only frontend

## Implementation

### Distribution Configuration

Ensure access logging is enabled (this is the default):

```yaml
log:
  accesslog:
    disabled: false
  formatter: text # or json for easier parsing
```

### Log Processor (Vector Example)

```toml
[sources.registry_logs]
type = "file"
include = ["/var/log/distribution/*.log"]

[transforms.parse_pulls]
type = "remap"
inputs = ["registry_logs"]
source = '''
. = parse_apache_log!(.message, "combined")

# Only process successful manifest GETs
if .method != "GET" || .status != 200 { abort }
if !starts_with(.path, "/v2/") || !contains(.path, "/manifests/") { abort }

# Extract repo and reference from /v2/{repo}/manifests/{ref}
parts = split(.path, "/manifests/")
.repo = replace(parts[0], "/v2/", "")
.reference = parts[1]

# Extract architecture from user-agent (docker clients include arch/xxx)
arch_match = parse_regex(.user_agent, r'arch/(?P<arch>\w+)')
.arch = arch_match.arch ?? "unknown"

# Extract OS from user-agent
os_match = parse_regex(.user_agent, r'os/(?P<os>\w+)')
.os = os_match.os ?? "unknown"
'''

[sinks.postgres]
type = "postgresql"
inputs = ["parse_pulls"]
endpoint = "postgres://user:pass@localhost/metrics"
table = "pull_events"
```

### Database Schema

For raw event storage (enables flexible querying):

```sql
CREATE TABLE pull_events (
    id BIGSERIAL PRIMARY KEY,
    pulled_at TIMESTAMPTZ NOT NULL,
    repo TEXT NOT NULL,
    reference TEXT NOT NULL,
    client_ip INET,
    arch TEXT,
    os TEXT,
    user_agent TEXT,
    response_size INTEGER
);

CREATE INDEX idx_pull_events_repo_time ON pull_events (repo, pulled_at DESC);
CREATE INDEX idx_pull_events_time ON pull_events (pulled_at DESC);
```

For pre-aggregated counts (if raw events are too voluminous):

```sql
CREATE TABLE pull_metrics (
    pull_date DATE NOT NULL,
    repo TEXT NOT NULL,
    reference TEXT NOT NULL,
    arch TEXT NOT NULL,
    pull_count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (pull_date, repo, reference, arch)
);
```

### Migration Path

1. Deploy log processor alongside Distribution
2. Verify metrics are being captured correctly
3. Remove webhook configuration from Distribution
4. Remove webhook handling code from StaticReg
5. Delete `pkg/webhook/`, `pkg/db/`, `pkg/sql/` directories
6. Remove PostgreSQL dependency from StaticReg

### Kubernetes Deployment Example

Vector runs as a sidecar container alongside Distribution, reading logs from a shared emptyDir volume.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vector-config
data:
  vector.toml: |
    [sources.registry_logs]
    type = "file"
    include = ["/var/log/registry/*.log"]
    read_from = "beginning"

    [transforms.parse_pulls]
    type = "remap"
    inputs = ["registry_logs"]
    source = '''
    . = parse_apache_log!(.message, "combined")

    # Only process successful manifest GETs
    if .method != "GET" || .status != 200 { abort }
    if !starts_with(.path, "/v2/") || !contains(.path, "/manifests/") { abort }

    # Extract repo and reference from /v2/{repo}/manifests/{ref}
    parts = split(.path, "/manifests/")
    .repo = replace(parts[0], "/v2/", "")
    .reference = parts[1]

    # Extract architecture from user-agent
    arch_match = parse_regex(.user_agent, r'arch/(?P<arch>\w+)')
    .arch = arch_match.arch ?? "unknown"

    # Extract OS from user-agent
    os_match = parse_regex(.user_agent, r'os/(?P<os>\w+)')
    .os = os_match.os ?? "unknown"

    # Keep only fields we need
    del(.message)
    del(.user_agent)
    '''

    [sinks.postgres]
    type = "postgresql"
    inputs = ["parse_pulls"]
    endpoint = "postgresql://metrics:${POSTGRES_PASSWORD}@postgres:5432/metrics"
    table = "pull_events"

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: distribution-config
data:
  config.yml: |
    version: 0.1
    log:
      accesslog:
        disabled: false
      level: info
    storage:
      filesystem:
        rootdirectory: /var/lib/registry
    http:
      addr: :5000
      headers:
        X-Content-Type-Options: [nosniff]

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: distribution
  labels:
    app: distribution
spec:
  replicas: 1
  selector:
    matchLabels:
      app: distribution
  template:
    metadata:
      labels:
        app: distribution
    spec:
      containers:
        # Distribution registry
        - name: registry
          image: registry:2
          ports:
            - containerPort: 5000
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
            - name: registry-config
              mountPath: /etc/docker/registry
            - name: logs
              mountPath: /var/log/registry
          # Redirect stdout/stderr to log file for Vector to read
          command:
            - /bin/sh
            - -c
            - |
              /bin/registry serve /etc/docker/registry/config.yml 2>&1 | tee /var/log/registry/access.log

        # Vector sidecar for log processing
        - name: vector
          image: timberio/vector:0.34.1-alpine
          volumeMounts:
            - name: logs
              mountPath: /var/log/registry
              readOnly: true
            - name: vector-config
              mountPath: /etc/vector
          env:
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
          args:
            - --config-dir
            - /etc/vector

      volumes:
        - name: registry-data
          persistentVolumeClaim:
            claimName: registry-data
        - name: registry-config
          configMap:
            name: distribution-config
        - name: vector-config
          configMap:
            name: vector-config
        - name: logs
          emptyDir: {}

---
apiVersion: v1
kind: Service
metadata:
  name: distribution
spec:
  selector:
    app: distribution
  ports:
    - port: 5000
      targetPort: 5000

---
# PostgreSQL for metrics storage
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  labels:
    app: postgres
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
        - name: postgres
          image: postgres:16-alpine
          ports:
            - containerPort: 5432
          env:
            - name: POSTGRES_DB
              value: metrics
            - name: POSTGRES_USER
              value: metrics
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: password
          volumeMounts:
            - name: postgres-data
              mountPath: /var/lib/postgresql/data
            - name: init-scripts
              mountPath: /docker-entrypoint-initdb.d
      volumes:
        - name: postgres-data
          persistentVolumeClaim:
            claimName: postgres-data
        - name: init-scripts
          configMap:
            name: postgres-init

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-init
data:
  init.sql: |
    CREATE TABLE IF NOT EXISTS pull_events (
        id BIGSERIAL PRIMARY KEY,
        pulled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        repo TEXT NOT NULL,
        reference TEXT NOT NULL,
        client_ip INET,
        arch TEXT,
        os TEXT,
        response_size INTEGER
    );

    CREATE INDEX IF NOT EXISTS idx_pull_events_repo_time
        ON pull_events (repo, pulled_at DESC);
    CREATE INDEX IF NOT EXISTS idx_pull_events_time
        ON pull_events (pulled_at DESC);

    -- Aggregation view for common queries
    CREATE OR REPLACE VIEW pull_metrics_daily AS
    SELECT
        DATE(pulled_at) as pull_date,
        repo,
        reference,
        arch,
        COUNT(*) as pull_count
    FROM pull_events
    GROUP BY DATE(pulled_at), repo, reference, arch;

---
apiVersion: v1
kind: Service
metadata:
  name: postgres
spec:
  selector:
    app: postgres
  ports:
    - port: 5432
      targetPort: 5432

---
apiVersion: v1
kind: Secret
metadata:
  name: postgres-credentials
type: Opaque
stringData:
  password: "change-me-in-production"
```

#### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Distribution Pod                        │
│  ┌─────────────────────┐    ┌─────────────────────────────┐ │
│  │                     │    │                             │ │
│  │    Distribution     │───▶│  /var/log/registry/         │ │
│  │    (registry:2)     │    │  access.log                 │ │
│  │                     │    │                             │ │
│  └─────────────────────┘    └──────────────┬──────────────┘ │
│                                            │ (emptyDir)     │
│  ┌─────────────────────┐                   │                │
│  │                     │◀──────────────────┘                │
│  │    Vector Sidecar   │                                    │
│  │    (log processor)  │                                    │
│  │                     │                                    │
│  └──────────┬──────────┘                                    │
│             │                                               │
└─────────────┼───────────────────────────────────────────────┘
              │
              │ Parsed pull events
              ▼
┌─────────────────────────────────────────────────────────────┐
│                     PostgreSQL Pod                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  pull_events table                                      ││
│  │  - pulled_at, repo, reference, arch, client_ip, ...     ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

#### Alternative: DaemonSet with Host Logs

If Distribution writes logs to a host path or uses a different logging setup, Vector can run as a DaemonSet:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: vector
spec:
  selector:
    matchLabels:
      app: vector
  template:
    metadata:
      labels:
        app: vector
    spec:
      containers:
        - name: vector
          image: timberio/vector:0.34.1-alpine
          volumeMounts:
            - name: var-log
              mountPath: /var/log
              readOnly: true
            - name: vector-config
              mountPath: /etc/vector
      volumes:
        - name: var-log
          hostPath:
            path: /var/log
        - name: vector-config
          configMap:
            name: vector-config
```

## Consequences

### Positive

- StaticReg remains a simple, stateless frontend
- Metrics pipeline is independent and can be scaled/modified separately
- No risk of StaticReg downtime affecting pull metrics
- Richer data capture enables future analytics
- Standard tooling reduces maintenance burden

### Negative

- Requires deploying additional infrastructure (log processor)
- Metrics are not available in StaticReg UI without additional API work
- Slight delay between pull and metric availability

### Neutral

- Pull metrics become a separate concern from the StaticReg application
- Team needs familiarity with log processing tools (Vector/Fluentbit)

## References

### Distribution

- [Distribution Registry](https://github.com/distribution/distribution) - Source code
- [Distribution Access Logging](https://github.com/distribution/distribution/blob/main/registry/registry.go#L158-L160) - Access log implementation
- [Gorilla Handlers Combined Log Format](https://pkg.go.dev/github.com/gorilla/handlers#CombinedLoggingHandler) - Apache Combined Log Format

### Vector Documentation

- [Vector](https://vector.dev/) - High-performance observability data pipeline
- [Vector Quickstart](https://vector.dev/docs/setup/quickstart/) - Getting started guide
- [File Source](https://vector.dev/docs/reference/configuration/sources/file/) - Reading log files
- [Remap Transform](https://vector.dev/docs/reference/configuration/transforms/remap/) - VRL transformation language
- [parse_apache_log function](https://vector.dev/docs/reference/vrl/functions/#parse_apache_log) - Parsing Combined Log Format
- [parse_regex function](https://vector.dev/docs/reference/vrl/functions/#parse_regex) - Regex extraction
- [PostgreSQL Sink](https://vector.dev/docs/reference/configuration/sinks/postgresql/) - Writing to PostgreSQL
- [Vector Kubernetes](https://vector.dev/docs/setup/installation/platforms/kubernetes/) - Kubernetes deployment guide
- [Vector Helm Chart](https://github.com/vectordotdev/helm-charts) - Official Helm charts

### Alternatives

- [Harbor Pull Metrics](https://github.com/goharbor/harbor) - Application-level interception approach
- [Fluentbit](https://fluentbit.io/) - Alternative log processor
- [Fluent Bit Kubernetes](https://docs.fluentbit.io/manual/installation/kubernetes) - Fluentbit K8s deployment

## Appendix: Current Implementation Issues

The current webhook-based implementation (commit bbd659ad) has these specific issues:

### Schema Problems (`pkg/sql/event_schema.sql`)

```sql
CREATE TABLE container_pull_metrics (
    id BIGSERIAL PRIMARY KEY,        -- Unnecessary for counter table
    pull_date DATE NOT NULL,
    repo_name TEXT NOT NULL,
    tag TEXT NOT NULL,
    digest TEXT NOT NULL,
    architecture TEXT NOT NULL,
    pull_count INTEGER DEFAULT 1,
    created_at TIMESTAMP,            -- Confusing for aggregation
    updated_at TIMESTAMP,
    UNIQUE(pull_date, repo_name, tag, digest, architecture)
);
```

- `id` column is redundant (UNIQUE constraint already identifies rows)
- `created_at` is misleading (only records first pull of the day)
- Speculative indexes without query patterns

### Data Loss (`pkg/webhook/events.go`)

Available in webhook but discarded:

- `Request.Addr` - client IP address
- `Request.UserAgent` - full string (only `arch/` extracted)
- `Actor.Name` - user identity
- `Source` - registry node info

### Migration Issues (`pkg/db/postgres.go`)

- Raw SQL execution at startup
- No migration versioning
- No rollback capability
- `os.Exit(1)` on schema failure
