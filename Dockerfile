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
FROM alpine:3.23

# Install runtime dependencies with multiple PHP versions
RUN apk add --no-cache \
    git \
    ca-certificates \
    nodejs \
    npm \
    # PHP 8.2
    php82 \
    php82-phar \
    php82-mbstring \
    php82-openssl \
    php82-curl \
    php82-iconv \
    php82-zip \
    php82-dom \
    php82-xml \
    php82-xmlwriter \
    php82-xmlreader \
    php82-pdo \
    php82-pdo_mysql \
    php82-pdo_pgsql \
    php82-pdo_sqlite \
    php82-session \
    php82-ctype \
    php82-fileinfo \
    php82-simplexml \
    php82-bcmath \
    php82-intl \
    php82-sodium \
    php82-pcntl \
    php82-posix \
    # PHP 8.3
    php83 \
    php83-phar \
    php83-mbstring \
    php83-openssl \
    php83-curl \
    php83-iconv \
    php83-zip \
    php83-dom \
    php83-xml \
    php83-xmlwriter \
    php83-xmlreader \
    php83-tokenizer \
    php83-pdo \
    php83-pdo_mysql \
    php83-pdo_pgsql \
    php83-pdo_sqlite \
    php83-session \
    php83-ctype \
    php83-fileinfo \
    php83-simplexml \
    php83-bcmath \
    php83-intl \
    php83-sodium \
    php83-pcntl \
    php83-posix \
    # PHP 8.4
    php84 \
    php84-phar \
    php84-mbstring \
    php84-openssl \
    php84-curl \
    php84-iconv \
    php84-zip \
    php84-dom \
    php84-xml \
    php84-xmlwriter \
    php84-xmlreader \
    php84-tokenizer \
    php84-pdo \
    php84-pdo_mysql \
    php84-pdo_pgsql \
    php84-pdo_sqlite \
    php84-session \
    php84-ctype \
    php84-fileinfo \
    php84-simplexml \
    php84-bcmath \
    php84-intl \
    php84-sodium \
    php84-pcntl \
    php84-posix \
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
    php85-intl \
    php85-sodium \
    php85-pcntl \
    php85-posix \
    composer \
    && ln -s /usr/bin/php85 /usr/bin/php

# Create non-root user
RUN adduser -D -h /home/updati updati
USER updati
WORKDIR /home/updati

COPY --from=builder /updati /usr/local/bin/updati

ENTRYPOINT ["/usr/local/bin/updati"]
