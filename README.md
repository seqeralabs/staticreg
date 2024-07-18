# :package: staticreg

A tool to generate a static website from a Docker V2 registry containing all the images and tags the provided user has access to.

## Development

Start a local github registry
```bash
docker run -d -p 5000:5000 --name registry registry
docker pull alpine
docker tag alpine localhost:5000/alpine
docker push localhost:5000/alpine
```

Build and start `staticreg`

```bash
go build .
./staticreg
```
