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
make
```

Start staticreg

```bash
./_output/dist/staticreg serve
```

## Release build (without releasing)

```bash
make clean
make RELEASE_BUILD=1 GORELEASER_CMD="docker run -v $PWD:/staticreg -w /staticreg --rm --privileged docker.io/goreleaser/goreleaser:v2.1.0"
```


## Manual Release to GitHub

**NB**: This is only done manually in case the GH action does not work.

Export a `GITHUB_TOKEN`, generate it from [here](https://github.com/settings/tokens/new?scopes=repo,write:packages) with `write:packages` permissions.


**Release a snapshot version**

```bash
make release-snapshot
```

**Release a final version**

Bump version in the `VERSION` file

```bash
echo "<new version>" > VERSION
```

Then

```bash
make release
```
