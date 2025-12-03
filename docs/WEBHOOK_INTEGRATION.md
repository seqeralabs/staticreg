# Docker Distribution Webhook Integration

This guide explains how to configure Docker Distribution to send webhook events to staticreg for metrics tracking.

## Overview

Staticreg can receive webhook notifications from Docker Distribution registries to track container pull and push events in real-time. These events are stored in PostgreSQL for analysis and reporting.

## Prerequisites

1. Docker Distribution registry (version 2.0+)
2. PostgreSQL database (version 18+ recommended) - **Optional**: Required only if you want to persist webhook metrics
3. Network connectivity between the registry and staticreg
4. Staticreg running and accessible to the registry

**Note:** Staticreg can run without a database for basic registry browsing. Webhook events will be accepted but not persisted if `STATICREG_DB_URL` is not configured.

## Setup Steps

### 1. Database Setup

First, create and configure the PostgreSQL database:

```bash
# Create database
createdb staticreg
```

**Note:** The database schema is automatically initialized when staticreg starts. The schema creates:
- `container_pull_metrics` table for storing aggregated pull metrics
- Indexes for efficient querying by date, repository, and architecture
- Unique constraint on (pull_date, repo_name, tag, digest, architecture)

If you prefer to manually initialize the schema, you can run:
```bash
# Apply the schema manually (optional)
psql -d staticreg -f sql/event_schema.sql
```

### 2. Configure Environment Variables

Set the database connection string for staticreg:

```bash
export STATICREG_DB_URL="postgresql://username:password@localhost:5432/staticreg"
```

Full connection string format:
```
postgresql://[user[:password]@][host][:port][/dbname][?param1=value1&...]
```

**Note:** The `STATICREG_DB_URL` environment variable is optional. If not set, staticreg will start successfully but webhook events will not be stored. This allows running staticreg without a database for basic registry browsing functionality.

### 3. Configure Docker Distribution Registry

Add webhook notification endpoint to your registry configuration file (typically `config.yml`):

```yaml
version: 0.1
log:
  level: info
  fields:
    service: registry

storage:
  filesystem:
    rootdirectory: /var/lib/registry

http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]

notifications:
  endpoints:
    - name: staticreg-webhook
      url: http://staticreg-host:8093/api/webhook/registry
      timeout: 5s
      threshold: 3
      backoff: 1s
```

**Configuration Parameters:**

- `name` - Friendly name for the webhook endpoint
- `url` - Full URL to staticreg webhook endpoint
  - Format: `http(s)://[host]:[port]/api/webhook/registry`
  - Default staticreg port is 8093
- `timeout` - Maximum time to wait for webhook response
- `threshold` - Number of failures before giving up
- `backoff` - Delay between retry attempts

### 4. Restart Docker Distribution

After updating the configuration, restart your registry:

```bash
docker restart <registry-container-name>
```

Or if running as a service:
```bash
systemctl restart docker-distribution
```

## Docker Compose Setup

For local development or testing, you can use docker-compose:

```yaml
services:
  postgres:
    image: postgres:18.1
    environment:
      POSTGRES_DB: staticreg
      POSTGRES_USER: staticreg
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"

  oci-registry:
    image: registry:3
    ports:
      - "5000:5000"
    volumes:
      - ./registry.yml:/etc/distribution/config.yml:ro
    depends_on:
      - staticreg

  staticreg:
    image: staticreg:latest
    ports:
      - "8093:8093"
    environment:
      STATICREG_DB_URL: postgresql://staticreg:password@postgres:5432/staticreg
      STATICREG_REGISTRY_HOSTNAME: oci-registry:5000
    depends_on:
      - postgres
```

## Event Types

Staticreg processes the following Docker Distribution events:

### Manifest Pull Events

These events indicate a complete container image pull:

- **Media Types Tracked:**
  - `application/vnd.docker.distribution.manifest.v2+json`
  - `application/vnd.docker.distribution.manifest.list.v2+json`
  - `application/vnd.oci.image.manifest.v1+json`
  - `application/vnd.oci.image.index.v1+json`

### Manifest Push Events

These events indicate a complete container image push (future feature).

### Event Filtering

Not all events from Docker Distribution are stored. Staticreg filters events to track only significant actions:

- Only manifest-level events (not individual blob pulls)
- Events with repository and tag information
- Events with valid timestamps
- **Events from staticreg itself are excluded** to prevent counting internal manifest fetches when staticreg browses the registry

## Metrics Data Schema

Pull metrics are aggregated and stored by date, repository, tag, digest, and architecture:

| Column | Type | Description |
|--------|------|-------------|
| id | BIGSERIAL | Primary key |
| pull_date | DATE | Date of the pull event (UTC) |
| repo_name | TEXT | Repository name (e.g., `library/nginx`) |
| tag | TEXT | Image tag (e.g., `latest`, `1.21`) |
| digest | TEXT | Image manifest digest (e.g., `sha256:abc123...`) |
| architecture | TEXT | CPU architecture (e.g., `amd64`, `arm64`, `not-specified`) |
| pull_count | INTEGER | Aggregated count of pulls for this combination |
| created_at | TIMESTAMP | When this record was first created |
| updated_at | TIMESTAMP | When this record was last updated |

**Unique Constraint:** `(pull_date, repo_name, tag, digest, architecture)`

When a pull event is received, the system:
1. Extracts architecture from the Docker client's user agent (defaults to `not-specified` if not present)
2. Uses UPSERT logic to increment `pull_count` if the combination exists
3. Creates a new record with `pull_count = 1` if it's a new combination

This approach significantly reduces database size and query complexity compared to storing individual events.

**Note:** The `digest` field was added to uniquely identify different image versions that may share the same tag. This ensures accurate tracking when tags are updated to point to new image versions.

## Verification

### Check Registry Logs

After configuration, verify the registry is attempting to send webhooks:

```bash
docker logs <registry-container> | grep webhook
```

You should see log entries indicating webhook delivery attempts.

### Check Staticreg Logs

Monitor staticreg logs for incoming webhook events:

```bash
# If running in Docker
docker logs <staticreg-container>

# If running as a binary
# Check stdout/stderr or log files
```

Successful webhook processing will show log entries like:
```
INFO Webhook processing completed totalEvents=1 eventsProcessed=1 eventsSkipped=0
INFO Pull metrics updated repository=library/nginx tag=latest digest=sha256:abc... architecture=amd64
```

If the database is not configured, you'll see:
```
DEBUG Skipping pull event save: Database pool is not active.
```

### Query the Database

Verify metrics are being stored:

```sql
-- Check recent metrics
SELECT pull_date, repo_name, tag, architecture, pull_count
FROM container_pull_metrics
ORDER BY pull_date DESC, updated_at DESC
LIMIT 10;

-- Total pulls by repository
SELECT repo_name, SUM(pull_count) as total_pulls
FROM container_pull_metrics
GROUP BY repo_name
ORDER BY total_pulls DESC;

-- Pulls today
SELECT SUM(pull_count) as pulls_today
FROM container_pull_metrics
WHERE pull_date = CURRENT_DATE;
```

## Troubleshooting

### Webhooks Not Being Received

1. **Check network connectivity:**
   ```bash
   # From registry container/host
   curl http://staticreg-host:8093/api/webhook/registry
   ```

2. **Verify registry configuration:**
   - Check the `notifications` section in registry config
   - Ensure URL is correct and accessible
   - Verify registry was restarted after config changes

3. **Check firewall rules:**
   - Ensure port 8093 is open on staticreg host
   - Verify no firewall blocking between registry and staticreg

### Events Not Being Stored

1. **Check if STATICREG_DB_URL is set:**
   ```bash
   echo $STATICREG_DB_URL
   ```
   If not set, staticreg will accept webhooks but not store them. Check startup logs for:
   ```
   WARNING: STATICREG_DB_URL environment variable is not set. Database functions will be disabled.
   ```

2. **Verify database connection:**
   ```bash
   psql $STATICREG_DB_URL -c "SELECT 1"
   ```

3. **Check database schema:**
   The schema is automatically created on startup. Check logs for:
   ```
   PostgreSQL connection pool successfully initialized.
   Database schema initialized successfully.
   ```

   To verify manually:
   ```sql
   \dt container_pull_metrics
   \d container_pull_metrics
   ```

4. **Check database permissions:**
   ```sql
   SELECT has_table_privilege('container_pull_metrics', 'INSERT');
   SELECT has_table_privilege('container_pull_metrics', 'UPDATE');
   ```

5. **Check staticreg startup logs:**
   Look for warnings about database initialization failures:
   ```
   WARNING: Unable to create connection pool: <error>. Database functions will be disabled.
   ```

### Registry Performance Issues

If webhook delivery impacts registry performance:

1. Increase timeout value
2. Reduce threshold (fail faster)
3. Consider asynchronous webhook processing
4. Monitor registry logs for webhook failures

## Security Considerations

### Network Security

- Use HTTPS for webhook URL in production
- Consider using a VPN or private network between registry and staticreg
- Implement IP allowlisting on staticreg webhook endpoint

### Authentication (Future Enhancement)

Future versions may support:
- Webhook signature verification
- Shared secret authentication
- OAuth/token-based authentication

### Database Security

- Use strong database passwords
- Limit database user permissions to required tables
- Enable SSL/TLS for database connections
- Regular backups of metrics data

## Performance Optimization

### Database

- Indexes are automatically created on `pull_date`, `(repo_name, pull_date)`, and `(repo_name, architecture, pull_date)`
- UPSERT operations efficiently aggregate metrics, reducing row count
- Consider partitioning by `pull_date` for very large datasets
- Implement data retention policies (e.g., keep 90 days of daily metrics)

### Registry

- Monitor webhook endpoint latency
- Set appropriate timeouts to avoid blocking registry operations
- Consider webhook endpoint caching/buffering

## Example Queries

### Pull Activity by Date

```sql
-- Pulls per day for the last 30 days
SELECT
  pull_date,
  SUM(pull_count) as daily_pulls
FROM container_pull_metrics
WHERE pull_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY pull_date
ORDER BY pull_date DESC;
```

### Most Popular Images

```sql
-- Top 10 most pulled images in the last 7 days
SELECT
  repo_name,
  tag,
  SUM(pull_count) as total_pulls
FROM container_pull_metrics
WHERE pull_date >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY repo_name, tag
ORDER BY total_pulls DESC
LIMIT 10;
```

### Architecture Distribution

```sql
-- Pull counts by architecture for the last 30 days
SELECT
  architecture,
  SUM(pull_count) as total_pulls,
  COUNT(DISTINCT repo_name) as unique_images
FROM container_pull_metrics
WHERE pull_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY architecture
ORDER BY total_pulls DESC;
```

### Image Pulls by Architecture

```sql
-- Breakdown of pulls by architecture for a specific image
SELECT
  tag,
  architecture,
  SUM(pull_count) as pulls
FROM container_pull_metrics
WHERE repo_name = 'library/nginx'
  AND pull_date >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY tag, architecture
ORDER BY tag, pulls DESC;
```

### Daily Trends

```sql
-- Daily pull trends with architecture breakdown
SELECT
  pull_date,
  architecture,
  SUM(pull_count) as pulls
FROM container_pull_metrics
WHERE pull_date >= CURRENT_DATE - INTERVAL '14 days'
GROUP BY pull_date, architecture
ORDER BY pull_date DESC, architecture;
```

### Tag Updates and Version Tracking

```sql
-- Track different image versions (digests) for the same tag
-- Useful for seeing when tags were updated to point to new images
SELECT
  repo_name,
  tag,
  digest,
  SUM(pull_count) as total_pulls,
  MAX(updated_at) as last_pulled
FROM container_pull_metrics
WHERE repo_name = 'library/nginx'
  AND tag = 'latest'
  AND pull_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY repo_name, tag, digest
ORDER BY last_pulled DESC;
```

## Next Steps

- Set up monitoring and alerting for webhook failures
- Create dashboards for visualizing pull metrics
- Implement data retention policies
- Consider adding webhook authentication