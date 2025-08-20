# --- Build stage ---
FROM golang:1.22-alpine AS builder

# Ensure we don’t need C dependencies (fully static)
ENV CGO_ENABLED=0 GOOS=linux

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source
COPY . .

# Build main.go inside cmd/
RUN go build -o /out/app ./cmd

# --- Runtime stage ---
FROM alpine:3.20

# Install certs & timezone info (for HTTPS + timezones)
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -g '' appuser

WORKDIR /app
COPY --from=builder /out/app .

USER appuser

EXPOSE 9080
ENTRYPOINT ["./app"]
