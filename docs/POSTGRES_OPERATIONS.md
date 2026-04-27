# Postgres Operations

Operational guidance for running StaticReg's postgres integration in production.
For full configuration reference see [CONFIGURATION.md](CONFIGURATION.md).

## Schema layout

StaticReg owns a single dedicated schema (default `staticreg`, configurable via
`STATICREG_DB_SCHEMA`). Two tables live there:

- `container_pull_metrics` — aggregated pull events.
- `goose_db_version` — migration bookkeeping (managed by goose; do not edit by
  hand).

The schema is created on startup with `CREATE SCHEMA IF NOT EXISTS`, and every
pooled connection has `search_path` set to it.

## Inspecting state

```sql
-- Confirm StaticReg's schema exists and is the active search_path
SHOW search_path;
\dn

-- List StaticReg's tables
\dt staticreg.*

-- Check the migration version
SELECT version_id, is_applied, tstamp
FROM staticreg.goose_db_version
ORDER BY id;
```

The same information is reachable via the HTTP API:

```bash
curl -k https://<host>:8093/healthz/db    # 200 if pool can ping
curl -k https://<host>:8093/metrics/db    # pgxpool stats as JSON
```

## Adding a new migration

1. Create `pkg/sql/migrations/NNNNN_<short_description>.sql` with the next
   sequential prefix (e.g. `00002_add_image_size_column.sql`).
2. Use goose's annotation markers:

   ```sql
   -- +goose Up
   -- +goose StatementBegin
   ALTER TABLE container_pull_metrics ADD COLUMN image_size BIGINT;
   -- +goose StatementEnd

   -- +goose Down
   -- +goose StatementBegin
   ALTER TABLE container_pull_metrics DROP COLUMN image_size;
   -- +goose StatementEnd
   ```

3. Build & run StaticReg locally against `docker compose up postgres`. The new
   migration runs automatically on startup; check the logs for
   `Database schema ready.`
4. Verify the change is visible in `staticreg.goose_db_version` and idempotent
   on a second startup.

**Notes:**

- Migrations run in numeric order. Never renumber or rewrite a migration that
  has been applied to a deployed database — write a new one instead.
- All DDL must be schema-unqualified inside migration files. The pool's
  `search_path` puts new objects in the dedicated schema automatically; using
  `public.foo` or hardcoding `staticreg.foo` defeats the schema-isolation guarantee.
- Wrap multi-statement migrations in `StatementBegin`/`StatementEnd` so goose
  treats them atomically. Single statements do not need the markers.

## Rolling back

StaticReg does not ship a goose CLI binary. To roll back a migration in
production:

1. Bring up an out-of-band container that has goose installed *and* mounts the
   `pkg/sql/migrations/` directory:

   ```bash
   docker run --rm -it \
     -v "$PWD/pkg/sql/migrations:/migrations:ro" \
     -e GOOSE_DRIVER=postgres \
     -e GOOSE_DBSTRING="host=$DB_HOST user=$DB_USER password=$DB_PASS dbname=$DB_NAME sslmode=require search_path=staticreg" \
     ghcr.io/pressly/goose:latest \
     -dir /migrations down
   ```

2. Or, equivalently, run a manual `DROP`/`ALTER` matching the `-- +goose Down`
   block of the migration you want to revert, then `DELETE FROM
   staticreg.goose_db_version WHERE version_id = <N>;`.

The `down` path is rarely used in practice — prefer rolling forward with a new
migration that undoes the change.

## Pool tuning guidance

| Workload | Suggested settings |
|---|---|
| Default | (leave defaults: `MAX_CONNS=25`, `MIN_CONNS=2`, `MAX_CONN_LIFETIME=1h`, `MAX_CONN_IDLE_TIME=30m`, `HEALTHCHECK_PERIOD=1m`) |
| High traffic (>1000 req/s) | `STATICREG_DB_MAX_CONNS=100`, `STATICREG_DB_MIN_CONNS=10`. Coordinate with postgres `max_connections`. |
| Many idle pods (k8s autoscaling) | `STATICREG_DB_MIN_CONNS=0`, `STATICREG_DB_MAX_CONN_IDLE_TIME=5m` so scaled-down pods do not hold open connections. |
| Postgres behind a connection pooler (PgBouncer) | `STATICREG_DB_MAX_CONN_LIFETIME=15m` to encourage rebalancing across pooler backends. |

Watch `/metrics/db` after tuning. If `empty_acquire_count` is climbing, the
pool is undersized for the workload — increase `MAX_CONNS` (and verify
postgres can accommodate the new ceiling).

## Schema migration when upgrading from v0.7

The bootstrap path in v0.7 created `container_pull_metrics` directly in
`public`. After upgrading, that data is **not** auto-migrated to the new
dedicated schema; the table simply stays where it is and StaticReg starts
writing to the new location.

To consolidate (run once, manually):

```sql
-- Inspect what's there first
SELECT count(*) FROM public.container_pull_metrics;

-- Copy across (preserves aggregation via the unique constraint)
INSERT INTO staticreg.container_pull_metrics
  (pull_date, repo_name, tag, digest, architecture, pull_count, created_at, updated_at)
SELECT pull_date, repo_name, tag, digest, architecture, pull_count, created_at, updated_at
FROM public.container_pull_metrics
ON CONFLICT (pull_date, repo_name, tag, digest, architecture) DO UPDATE
SET pull_count = staticreg.container_pull_metrics.pull_count + EXCLUDED.pull_count,
    updated_at = NOW();

-- Once verified
DROP TABLE public.container_pull_metrics;
```
