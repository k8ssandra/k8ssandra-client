# Build the binary
FROM golang:1.24 AS builder
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

FROM redhat/ubi9-minimal:latest AS ubi-builder

ARG YQ_VERSION
ARG KUBE_VERSION

# Install kubectl
RUN cat <<EOF | tee /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://pkgs.k8s.io/core:/stable:/v${KUBE_VERSION:-1.32}/rpm/
enabled=1
gpgcheck=1
gpgkey=https://pkgs.k8s.io/core:/stable:/v${KUBE_VERSION:-1.32}/rpm/repodata/repomd.xml.key
EOF

RUN microdnf install -y kubectl

WORKDIR /workspace

# Install yq
RUN curl -L "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION:-4.45.4}/yq_${TARGETOS:-linux}_${TARGETARCH:-amd64}" -o yq

# Build the UBI image
FROM redhat/ubi9-micro:latest

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
COPY --from=ubi-builder --chown=65532:65532 --chmod=755 /workspace/yq /usr/bin/yq
COPY --from=ubi-builder /usr/bin/kubectl /usr/bin/kubectl
COPY ./LICENSE /licenses/

USER 65532:65532

ENTRYPOINT ["/kubectl-k8ssandra"]
