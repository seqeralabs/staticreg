# Contributing to staticreg

To contribute you need

- [goreleaser](https://goreleaser.com/install/): This is the tool we use to build and release staticreg, used by GNU Make to compile the project
- [Go](https://go.dev/): The Go compiler, used by goreleaser to build binaries (at least 1.21 so we can use [Go Toolchains](https://go.dev/doc/toolchain))
- [GNU Make](https://www.gnu.org/software/make/): we use make to hide the details of running multiple commands to get builds done
- Optional: [Docker](https://docs.docker.com/desktop/install/linux-install/): to build container images and for running the local development dependencies

Start the local development environment with Docker Compose (includes Registry and Prometheus):

```bash
docker-compose up -d
```

This starts:
- OCI Registry on port 5000
- Prometheus monitoring on port 9090

Alternatively, start a standalone registry and push an image to it:

```bash
docker run -d -p 5000:5000 --name registry registry:3
docker pull alpine
docker tag alpine:latest localhost:5000/alpine:latest
docker push localhost:5000/alpine:latest
```

Build staticreg

```bash
make deps
make
```

Start staticreg

```bash
./_output/dist/staticreg serve
```

By default, staticreg will:
- Serve the web UI on http://localhost:8093
- Connect to the registry at localhost:5000
- Expose Prometheus metrics at http://localhost:8093/metrics

## Build (without releasing)

```bash
make clean
make
```

## Testing Prometheus Metrics

To test Prometheus metrics integration locally:

1. Ensure docker-compose services are running (including Prometheus):
   ```bash
   docker-compose up -d
   ```

2. Start staticreg:
   ```bash
   go run main.go serve
   ```

3. Pull an image from the local registry to trigger metrics:
   ```bash
   docker pull localhost:5000/alpine:latest
   ```

4. Access Prometheus UI:
   ```
   http://localhost:9090
   ```

5. Query staticreg metrics in Prometheus:
   - `staticreg_container_pulls_total` - Detailed pull metrics with all labels
   - `staticreg_container_pulls_by_repo_total` - Pulls aggregated by repository
   - `staticreg_container_pulls_by_arch_total` - Pulls aggregated by architecture

6. View the raw metrics endpoint:
   ```bash
   curl http://localhost:8093/metrics
   ```

## Release
Releasing is done via GitHub actions.

To release you need to:

- Bump version in the `VERSION` file (this needs to be a [semver](https://semver.org/) numbers)
- Commit the version file
- Start the release process


**Bump version file**
```bash
git checkout master
echo "<new version>" > VERSION
```

**Commit the version file**
```bash
git commit -am "release: v$(cat VERSION)"
git push
```



**Start the release process**

```bash
make release
```

