# One version for both modules: git tags vN (root) and gateway/vN (gateway).
# The image tag is derived from the gateway tag, e.g. v0.4.0 or v0.4.0-3-gabc123-dirty.
IMAGE ?= signal-gateway
DOCKER ?= sudo docker
VERSION ?= $(patsubst gateway/%,%,$(shell git describe --tags --match 'gateway/v*' --dirty))

.PHONY: help test vet build image version release

help:
	@echo "test      go test for the root module and the gateway"
	@echo "vet       go vet for the root module and the gateway"
	@echo "build     gateway binaries (linux, windows) stamped with the version"
	@echo "image     docker image $(IMAGE):<version>; also :latest on an exact release"
	@echo "version   print the version that build and image would use"
	@echo "release   VERSION=vX.Y.Z: tag the root, pin the gateway to it, tag and push both"

test:
	go test ./...
	cd gateway && go test ./...

vet:
	go vet ./...
	cd gateway && go vet ./...

build:
	$(MAKE) -C gateway build-linux build-windows VERSION=$(VERSION)

image:
	$(DOCKER) build -f gateway/Dockerfile --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .
	$(if $(findstring -,$(VERSION)),,$(DOCKER) tag $(IMAGE):$(VERSION) $(IMAGE):latest)

version:
	@echo $(VERSION)

# The root module zip excludes gateway/, so tagging the root first does not change
# its hash when only gateway/ changes afterwards.
release:
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "usage: make release VERSION=vX.Y.Z"; exit 1; }
	test "$$(git branch --show-current)" = main
	test -z "$$(git status --porcelain)"
	! git rev-parse -q --verify refs/tags/$(VERSION) >/dev/null
	test "$$(printf '%s\n%s\n' "$$(git tag --list 'v[0-9]*' | sort -V | tail -n1)" $(VERSION) | sort -V | tail -n1)" = "$(VERSION)"
	git tag $(VERSION)
	git push origin $(VERSION)
	cd gateway && GOWORK=off GOPROXY=direct GONOSUMDB=github.com/DonovanDiamond/signalmgr go get github.com/DonovanDiamond/signalmgr@$(VERSION) && GOWORK=off go mod tidy
	git add gateway/go.mod gateway/go.sum
	git commit -m "gateway: require signalmgr $(VERSION)"
	git tag gateway/$(VERSION)
	git push origin main gateway/$(VERSION)
