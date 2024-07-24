ARG GO_IMAGE=docker.io/golang:1.22
FROM ${GO_IMAGE} AS builder
ADD . /project
WORKDIR /project

RUN make DEBUG=0

FROM scratch

COPY --from=builder /project/_output/bin/staticreg /staticreg
