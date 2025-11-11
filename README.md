# :package: staticreg

A tool to serve a website from an OCI registry that supports the `/v2/_catalog` endpoint.

- [:package: staticreg](#package-staticreg)
  - [Features](#features)
  - [Install staticreg](#install-staticreg)
  - [Run staticreg](#run-staticreg)
    - [Serve the website](#serve-the-website)
    - [Run with Docker](#run-with-docker)
  - [Install on Kubernetes](#install-on-kubernetes)
  - [Container Registry Metrics](#container-registry-metrics)
    - [Database Setup](#database-setup)
    - [Webhook Configuration](#webhook-configuration)
    - [Example Queries](#example-queries)
  - [Contributing](#contributing)

## Features

:white_check_mark: Images list page<br>
:white_check_mark: Image tags list page<br>
:white_check_mark: Static website<br>
:white_check_mark: Image Security Scan<br>
:white_check_mark: Image Inspect<br>
:white_check_mark: Docker Distribution webhook integration<br>
:white_check_mark: Container pull/push metrics tracking<br>

<img alt="staticreg screenshot" src="docs/_static/staticreg.png">

## Install staticreg

If you need, you can run staticreg in your **Container runtime** or **Kubernetes cluster**, please see the sections below.

However, we also release pre-built binaries for Windows, Linux and MacOS for i386, x86_64 and arm64. Download them from [here](https://github.com/seqeralabs/staticreg/releases/latest).

## Run staticreg

### Serve the website

```bash
staticreg serve
```

### Run with Docker

```bash
docker run --rm -d cr.seqera.io/public/staticreg:0.4.7 serve --registry <registry-url-here> --wave-server-url <wave-server-url-here>
```

## Install on Kubernetes

Create a secret with the registry details (the registry you want to list images for)

```bash
kubectl create secret generic registry-credentials \
  --from-literal=REGISTRY_USER=<username> \
  --from-literal=REGISTRY_PASSWORD=<password> \
  --from-literal=REGISTRY_HOSTNAME=<hostname>
```

Create the staticreg deployment

```
kubectl apply -f manifests/deployment.yml
```

## Container Registry Metrics

Staticreg can collect and store container pull metrics from Docker Distribution registries via webhook integration. Metrics are aggregated by:

- **Date** - Daily pull counts
- **Repository & Tag** - Which images are being pulled
- **Architecture** - CPU architecture (amd64, arm64, etc.)

This provides efficient storage and fast queries for analyzing container usage patterns.

### Database Setup

Metrics are stored in PostgreSQL. To enable metrics collection:

1. Set up a PostgreSQL database
2. Apply the schema from `sql/event_schema.sql`
3. Configure the database connection:
   ```bash
   export DATABASE_URL="postgresql://user:password@localhost:5432/staticreg"
   ```

### Webhook Configuration

Configure your Docker Distribution registry to send webhook events to staticreg:

```yaml
notifications:
  endpoints:
    - name: staticreg-webhook
      url: http://your-staticreg-host:8093/api/webhook/registry
      timeout: 5s
      threshold: 3
      backoff: 1s
```

### Example Queries

```sql
-- Total pulls by repository
SELECT repo_name, SUM(pull_count) as total_pulls
FROM container_pull_metrics
GROUP BY repo_name
ORDER BY total_pulls DESC;

-- Pulls by architecture
SELECT architecture, SUM(pull_count) as total_pulls
FROM container_pull_metrics
WHERE pull_date >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY architecture;
```

For detailed setup instructions, see [docs/WEBHOOK_INTEGRATION.md](docs/WEBHOOK_INTEGRATION.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
