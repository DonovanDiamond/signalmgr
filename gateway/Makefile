# Release builds pass VERSION (see the root Makefile); a plain build reports dev.
VERSION ?= dev
LDFLAGS = -X main.version=$(VERSION)

.PHONY: help web build-linux build-windows test fakesignal run clean

help:
	@echo "web            npm ci and build the UI into web/build (embedded by the next Go build)"
	@echo "build-linux    web, then cross-compile dist/gateway-linux-amd64"
	@echo "build-windows  web, then cross-compile dist/gateway-windows-amd64.exe"
	@echo "test           go vet and go test ./..."
	@echo "fakesignal     run the in-memory fake signal-cli-rest-api on 127.0.0.1:8081"
	@echo "run            run the gateway with ./config.yaml (all defaults if absent)"
	@echo "clean          remove dist/"

web:
	cd web && npm ci && npm run build

# The UI is embedded at compile time, so a stale web/build ships stale screens.
build-linux: web
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gateway-linux-amd64 ./cmd/gateway

build-windows: web
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/gateway-windows-amd64.exe ./cmd/gateway

test:
	go vet ./...
	go test ./...

fakesignal:
	go run ./cmd/fakesignal

run:
	go run ./cmd/gateway -config config.yaml

clean:
	$(RM) -r dist
