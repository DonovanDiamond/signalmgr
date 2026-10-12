# syntax=docker/dockerfile:1
# Build context is the repository root: gateway/go.mod replaces the library with ../.
#   docker build -f gateway/Dockerfile .

# --- web: build the SvelteKit UI (adapter-static) ---
FROM node:22-bookworm-slim AS web
WORKDIR /src/gateway/web
COPY gateway/web/package.json gateway/web/package-lock.json ./
RUN npm ci
COPY gateway/web/ ./
RUN npm run build

# --- go: static binaries, CGO disabled ---
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY *.go ./
COPY signaltypes/ ./signaltypes/
WORKDIR /src/gateway
COPY gateway/go.mod gateway/go.sum ./
RUN go mod download
COPY gateway/ ./
COPY --from=web /src/gateway/web/build/ ./web/build/
# fakesignal and devseed only serve the compose "fake" profile; they are tiny.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /out/ \
    ./cmd/gateway ./cmd/fakesignal ./cmd/devseed

# --- runtime: plain HTTP, TLS terminates at a reverse proxy ---
FROM debian:bookworm-slim AS runtime
# ca-certificates: LDAPS to Active Directory fails only at runtime without a trust store.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -r -f /var/lib/apt/lists/*
RUN groupadd --system gateway \
    && useradd --system --gid gateway --home /data --create-home gateway
COPY --from=build /out/gateway /out/fakesignal /out/devseed /usr/local/bin/
WORKDIR /data
VOLUME /data
EXPOSE 8080
USER gateway
# No curl in the image: bash's /dev/tcp is enough to see the server answer 200.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD bash -c 'exec 3<>/dev/tcp/127.0.0.1/8080 && printf "GET / HTTP/1.0\r\n\r\n" >&3 && head -n1 <&3 | grep -q " 200"'
ENTRYPOINT ["/usr/local/bin/gateway"]
CMD ["-config", "/etc/gateway/config.yaml"]
