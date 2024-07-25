# :package: staticreg

A tool to generate a static website from an OCI registry that supports the `/v2/_catalog` endpoint.

- [:package: staticreg](#package-staticreg)
  - [Features](#features)
  - [Install staticreg](#install-staticreg)
  - [Run staticreg](#run-staticreg)
    - [Generate a static website `staticreg`](#generate-a-static-website-staticreg)
    - [Serve the website directly](#serve-the-website-directly)
    - [Run with Docker](#run-with-docker)
  - [Install on Kubernetes](#install-on-kubernetes)
  - [Contributing](#contributing)
  - [Release build](#release-build)

## Features

:white_check_mark: Images list page<br>
:white_check_mark: Image tags list page<br>
:white_check_mark: Static website or HTTP webserver with cache 

<img alt="staticreg screenshot" src="docs/_static/screenshot.png">


## Install staticreg

You can install staticreg only via `go install` for now 

```
go install github.com/seqeralabs/staticreg
```


Alternatively, you can run staticreg in your **Container runtime** or **Kubernetes cluster**, please see the sections below.

## Run staticreg

### Generate a static website `staticreg`

```bash
./staticreg generate
```

### Serve the website directly

```bash
./staticreg serve
```

### Run with Docker

```bash
docker run --rm -d public.cr.seqera.io/seqeralabs/staticreg:master
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


## Contributing

Start a local Registry and push an image to it

```bash
docker run -d -p 5000:5000 --name registry registry
docker pull alpine
docker tag alpine:latest localhost:5000/alpine:latest
docker push localhost:5000/alpine:latest
```

Build staticreg

```bash
make deps
make clean
make DEBUG=1
```

Start staticreg

```bash
./_output/bin/staticreg serve
```

## Release build

```bash
make clean
ARCH=amd64
make DEBUG=0 GO="docker run -e GOARCH=$ARCH -v $PWD:/staticreg -w /staticreg --rm docker.io/golang:1.22 go"
```


