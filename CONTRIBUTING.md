# Contributing to staticreg

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


