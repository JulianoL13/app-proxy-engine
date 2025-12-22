FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Create non-root user
RUN adduser -D -u 10001 appuser

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/scheduler ./cmd/scheduler

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/JulianoL13/app-proxy-engine"
LABEL org.opencontainers.image.description="Proxy Engine Scheduler"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/bin/scheduler /app/scheduler

USER appuser

CMD ["/app/scheduler"]
