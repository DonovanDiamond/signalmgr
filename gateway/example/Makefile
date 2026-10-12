# Run from anywhere: make -C gateway/example <target>
DIR := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
DC := docker compose -f $(DIR)compose.yaml
FAKE_VOLUME := signal-gateway-example_gateway-fake-data

.PHONY: help up fake down logs password ps build reset fake-reset

help:
	@echo "make up          build and start the real stack (gateway on 8090)"
	@echo "make fake        start the fake-data profile (gateway on 8091)"
	@echo "make down        stop everything, including the fake profile; keep data"
	@echo "make logs        follow the gateway logs"
	@echo "make password    print the one-time admin password line from the logs"
	@echo "make ps          list running services"
	@echo "make build       rebuild the gateway image"
	@echo "make reset       delete all volumes, including Signal accounts (asks first; FORCE=1 skips)"
	@echo "make fake-reset  wipe only the fake gateway's volume"

up:
	$(DC) up -d --build
	@echo "Gateway: http://127.0.0.1:8090  (user: admin)"
	@echo "First-start password: make password"

fake:
	$(DC) --profile fake up -d --build
	@echo "Fake gateway: http://127.0.0.1:8091  (user: admin)"
	@echo "First-start password: $(DC) logs gateway-fake | grep password"

down:
	$(DC) --profile fake down

logs:
	$(DC) logs -f gateway

# The password is logged only on the first start with an empty database.
password:
	@if ! $(DC) logs gateway 2>&1 | grep -i password; then \
		echo "No password line in the gateway logs. It is logged only on the first start with an empty database; if you set SIGNALGW_ADMIN_PASSWORD in .env, use that."; \
	fi

ps:
	$(DC) --profile fake ps

build:
	$(DC) build gateway

reset:
	@if [ "$(FORCE)" != 1 ]; then \
		echo "This deletes all volumes, including registered Signal accounts."; \
		printf "Type yes to continue: "; read ans; \
		if [ "$$ans" != yes ]; then echo "Aborted."; exit 1; fi; \
	fi
	$(DC) --profile fake down -v

fake-reset:
	$(DC) --profile fake rm -sf gateway-fake fake-seed fakesignal
	docker volume rm $(FAKE_VOLUME)
