# syntax=docker/dockerfile:1.7

# ---------- build stage ----------
FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/pg-backup-service .

# ---------- supercronic stage ----------
FROM alpine:3.20 AS supercronic

ARG SUPERCRONIC_VERSION=v0.2.33
ARG TARGETARCH

RUN apk add --no-cache curl \
    && case "${TARGETARCH}" in \
         amd64) SC_ARCH=amd64 ;; \
         arm64) SC_ARCH=arm64 ;; \
         *) echo "unsupported arch: ${TARGETARCH}" && exit 1 ;; \
       esac \
    && curl -fsSLo /usr/local/bin/supercronic \
         "https://github.com/aptible/supercronic/releases/download/${SUPERCRONIC_VERSION}/supercronic-linux-${SC_ARCH}" \
    && chmod +x /usr/local/bin/supercronic

# ---------- runtime stage ----------
FROM alpine:3.20

RUN apk add --no-cache \
        postgresql16-client \
        ca-certificates \
        tzdata \
        bash \
    && addgroup -S app \
    && adduser -S -G app app

COPY --from=builder /out/pg-backup-service /usr/local/bin/pg-backup-service
COPY --from=supercronic /usr/local/bin/supercronic /usr/local/bin/supercronic
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

USER app
WORKDIR /home/app

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
