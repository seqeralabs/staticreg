# :package: staticreg

A tool to generate a static website from an OCI registry.



Now you can either generate a static website or start a web server that updates automatically its content based  on the target registry.

## Generate a static website `staticreg`

```bash
./staticreg generate
```

## Serve the website directly

```bash
./staticreg serve
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
go generate ./...
make clean
make DEBUG=1
```


## Release build

```bash
make clean
ARCH=amd64
make GO="docker run -e GOARCH=$ARCH -v $PWD:/staticreg -w /staticreg --rm $(cat .go_image) go"
```
