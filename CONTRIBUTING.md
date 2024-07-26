# Contributing to staticreg

To contribute you need

- [goreleaser](https://goreleaser.com/install/): This is the tool we use to build and release staticreg, used by GNU Make to compile the project
- [Go >= 1.22.2](https://go.dev/): The Go compiler, used by goreleaser to build binaries
- [GNU Make](https://www.gnu.org/software/make/): we use make to hide the details of running multiple commands to get builds done
- Optional: [Docker](https://docs.docker.com/desktop/install/linux-install/): to build container images and for running the local development dependencies

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

## Build (without releasing)

```bash
make clean
make
```


## Manual Release to GitHub

**NB**: This is only done manually in case the GH action does not work.

Export a `GITHUB_TOKEN`, generate it from [here](https://github.com/settings/tokens/new?scopes=repo,write:packages) with `write:packages` permissions.



**Release**

Bump version in the `VERSION` file (this needs to be a [semver](https://semver.org/) numbers)

```bash
git checkout master
echo "<new version>" > VERSION
git commit -am "release: v$(cat VERSION)"
git push
```

Then you can either release:

```bash
make release
```

