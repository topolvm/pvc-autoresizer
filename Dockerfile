# Stage1: Build the pvc-autoresizer binary
FROM golang:1.16 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
# Copy the go source
COPY main.go main.go
COPY metrics/ metrics/
COPY runners/ runners/
COPY cmd/ cmd/

# Build
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -a -o pvc-autoresizer main.go

# Stage2: setup runtime container
FROM scratch
WORKDIR /
COPY --from=builder /workspace/pvc-autoresizer .
EXPOSE 8080
USER 10000:10000

ENTRYPOINT ["/pvc-autoresizer"]
