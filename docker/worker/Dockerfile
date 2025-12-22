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
    -o /app/bin/worker ./cmd/worker

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/JulianoL13/app-proxy-engine"
LABEL org.opencontainers.image.description="Proxy Engine Worker"
LABEL org.opencontainers.image.licenses="MIT"

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/bin/worker /app/worker

USER appuser

CMD ["/app/worker"]
