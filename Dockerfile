# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY backup.go backup.go
COPY go.mod go.mod
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o backup-binary backup.go

# Minimal final image with CA certs and timezone support
FROM gcr.io/distroless/static-debian12

# Add a layer to set the timezone if TZ is provided
ARG TZ
ENV TZ=${TZ}

WORKDIR /app/backup
COPY --from=builder /src/backup-binary /app/backup/backup-binary

# Install tzdata and set timezone if TZ is set
# (distroless does not support RUN, so this must be handled at runtime or by switching to a minimal debian base)
# For distroless, the Go app should use TZ env var for time.Local

ENTRYPOINT ["/app/backup/backup-binary"] 