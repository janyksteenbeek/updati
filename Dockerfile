# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" \
    -o /updati ./cmd/updati

# Runtime stage
FROM alpine:3.20

# Install runtime dependencies with all common PHP extensions
RUN apk add --no-cache \
    git \
    ca-certificates \
    nodejs \
    npm \
    php85 \
    php85-phar \
    php85-mbstring \
    php85-openssl \
    php85-curl \
    php85-iconv \
    php85-zip \
    php85-dom \
    php85-xml \
    php85-xmlwriter \
    php85-xmlreader \
    php85-tokenizer \
    php85-pdo \
    php85-pdo_mysql \
    php85-pdo_pgsql \
    php85-pdo_sqlite \
    php85-session \
    php85-ctype \
    php85-fileinfo \
    php85-simplexml \
    php85-bcmath \
    php85-gd \
    php85-intl \
    php85-sodium \
    php85-pcntl \
    php85-posix \
    composer

# Create non-root user
RUN adduser -D -h /home/updati updati
USER updati
WORKDIR /home/updati

COPY --from=builder /updati /usr/local/bin/updati

ENTRYPOINT ["/usr/local/bin/updati"]
