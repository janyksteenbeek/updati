# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /updati ./cmd/updati

# Runtime stage
FROM alpine:3.23

# Install runtime dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    nodejs \
    npm \
    php84 \
    php84-phar \
    php84-mbstring \
    php84-openssl \
    php84-curl \
    php84-json \
    php84-iconv \
    php84-zip \
    php84-dom \
    php84-xml \
    php84-xmlwriter \
    php84-tokenizer \
    composer

# Create non-root user
RUN adduser -D -h /home/updati updati
USER updati
WORKDIR /home/updati

# Copy binary
COPY --from=builder /updati /usr/local/bin/updati

ENTRYPOINT ["/usr/local/bin/updati"]

