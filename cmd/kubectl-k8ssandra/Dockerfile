# Build the binary
FROM golang:1.21 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Go Modules
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/

# Copy the cached helm charts
RUN mkdir -p build/
COPY build/ build/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o kubectl-k8ssandra cmd/kubectl-k8ssandra/main.go

# Build the UBI image
FROM redhat/ubi8-minimal:latest

ARG VERSION

LABEL maintainer="DataStax, Inc <info@datastax.com>"
LABEL name="k8ssandra-client"
LABEL vendor="DataStax, Inc"
LABEL release="${VERSION}"
LABEL version="${VERSION}"
LABEL summary="Client for k8ssandra installations"
LABEL description="Part of the toolset for the DataStax Kubernetes Operator for Apache Cassandra®."

WORKDIR /
COPY --from=builder /workspace/kubectl-k8ssandra .
COPY --from=builder --chown=65532:65532 /workspace/build/ .cache/
COPY ./LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/kubectl-k8ssandra"]
