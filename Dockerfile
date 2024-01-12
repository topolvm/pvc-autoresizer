# Stage1: Build the pvc-autoresizer binary
FROM golang:1.20 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# Copy the go source
COPY cmd/ cmd/
COPY hooks/ hooks/
COPY metrics/ metrics/
COPY runners/ runners/

# Build
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -a -o pvc-autoresizer cmd/*.go

# Stage2: setup runtime container
FROM scratch
WORKDIR /
COPY --from=builder /workspace/pvc-autoresizer .
EXPOSE 8080
USER 10000:10000

ENTRYPOINT ["/pvc-autoresizer"]
