# syntax=docker/dockerfile:1

# ============================================================
# Stage 1: Build
# Uses the official Go image to compile a static binary.
# ============================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency manifests first for Docker layer caching.
# The go mod download layer only re-runs when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a fully static binary.
# CGO_ENABLED=0 ensures no C runtime dependency (works in Alpine).
# Letting Go automatically detect the host architecture instead of forcing $TARGETARCH
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /fyers-trading .

# ============================================================
# Stage 2: Runtime
# Minimal Alpine image (~10MB total). Only the binary + certs.
# ============================================================
FROM alpine:3.19

# ca-certificates: required for HTTPS calls to Fyers REST API
# tzdata: required so time.LoadLocation("Asia/Kolkata") works
RUN apk add --no-cache ca-certificates tzdata

# Copy only the binary from the builder stage
COPY --from=builder /fyers-trading /usr/local/bin/fyers-trading

# CRITICAL FIX for Fyers SDK: The SDK uses runtime.Caller() to look for map.json
# at the exact absolute path where it was compiled (/go/pkg/mod/...).
# We must copy this specific file from the builder to the exact same absolute path
# in the runtime image, otherwise NewFyersDataSocket() returns nil and panics.
COPY --from=builder /go/pkg/mod/github.com/\!fyers\!dev/fyers-go-sdk@v1.1.0/websocket/map.json /go/pkg/mod/github.com/\!fyers\!dev/fyers-go-sdk@v1.1.0/websocket/map.json

# Default to IST timezone (matches Fyers exchange timestamps)
ENV TZ=Asia/Kolkata

# trading.json is mounted as a volume so it can be updated without rebuild
WORKDIR /app

ENTRYPOINT ["fyers-trading"]
