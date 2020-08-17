# Build the pvc-autoresizer binary
FROM golang:1.13 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/
# Copy the go source
COPY main.go main.go
COPY runners/ runners/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -mod=vendor -a -o pvc-autoresizer main.go

# Use distroless as minimal base image to package the pvc-autoresizer binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/pvc-autoresizer .
USER nonroot:nonroot

ENTRYPOINT ["/pvc-autoresizer"]
