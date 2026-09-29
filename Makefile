SERVICES := auth-service video-api processing-worker notification-service

# Detecção de arquitetura para escolher o override de compose correto.
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
ARCH := amd64
else ifeq ($(UNAME_M),amd64)
ARCH := amd64
else ifeq ($(UNAME_M),aarch64)
ARCH := arm64
else ifeq ($(UNAME_M),arm64)
ARCH := arm64
else
ARCH := amd64
endif

COMPOSE_FILES := -f docker-compose.yml -f docker-compose.$(ARCH).yml

.PHONY: up down logs infra run build test test-integration cover cover-integration lint smoke arch help

arch:
	@echo "arquitetura detectada: $(ARCH) ($(UNAME_M))"

up:
	docker compose $(COMPOSE_FILES) up -d --build

down:
	docker compose $(COMPOSE_FILES) down

logs:
	docker compose $(COMPOSE_FILES) logs -f

infra:
	docker compose $(COMPOSE_FILES) up -d postgres redis rabbitmq mailhog s3 prometheus grafana

run:
	@trap 'kill 0' INT TERM; \
	(go -C services/auth-service run ./cmd) & \
	(go -C services/video-api run ./cmd) & \
	(go -C services/processing-worker run ./cmd) & \
	(go -C services/notification-service run ./cmd) & \
	wait

build: $(addprefix build-,$(SERVICES))

build-%:
	cd services/$* && go build ./...

test: $(addprefix test-,$(SERVICES))

test-%:
	@cd services/$* && set -o pipefail && go test ./... -cover -count=1 2>&1 | grep -vE 'coverage: 0\.0% of statements|\[no test files\]'

test-integration: $(addprefix integration-,$(SERVICES))

integration-%:
	@cd services/$* && go test ./... -tags integration -count=1 -v 2>&1 | grep -vE '^2026/|^[0-9]{4}/'

cover: $(addprefix cover-,$(SERVICES))

cover-%:
	@mkdir -p coverage
	@cd services/$* && go test ./... -coverpkg=./internal/... -coverprofile=$(CURDIR)/coverage/$*.out -count=1 > /dev/null
	@cd services/$* && go tool cover -html=$(CURDIR)/coverage/$*.out -o $(CURDIR)/coverage/$*.html
	@printf "  %-22s total: %s  ->  coverage/%s.html\n" "$*" "$$(cd services/$* && go tool cover -func=$(CURDIR)/coverage/$*.out | tail -1 | awk '{print $$NF}')" "$*"

cover-integration: $(addprefix coverint-,$(SERVICES))

coverint-%:
	@mkdir -p coverage
	@cd services/$* && go test ./... -tags integration -coverpkg=./internal/... -coverprofile=$(CURDIR)/coverage/$*-integration.out -count=1 > /dev/null
	@cd services/$* && go tool cover -html=$(CURDIR)/coverage/$*-integration.out -o $(CURDIR)/coverage/$*-integration.html
	@printf "  %-22s total: %s  ->  coverage/%s-integration.html\n" "$*" "$$(cd services/$* && go tool cover -func=$(CURDIR)/coverage/$*-integration.out | tail -1 | awk '{print $$NF}')" "$*"

smoke:
	bash scripts/smoke-test.sh

lint: $(addprefix lint-,$(SERVICES))

lint-%:
	cd services/$* && golangci-lint run ./...

help:
	@echo "Comandos disponíveis (arquitetura detectada: $(ARCH)):"
	@echo "  make up          - sobe tudo (docker compose, incl. serviços)"
	@echo "  make down        - derruba a infraestrutura"
	@echo "  make logs        - acompanha logs da infra"
	@echo "  make infra       - sobe só a infraestrutura (para rodar serviços via Go)"
	@echo "  make run         - roda os 4 serviços via 'go run' (precisa do make infra)"
	@echo "  make arch        - mostra a arquitetura detectada"
	@echo "  make build       - compila todos os serviços"
	@echo "  make test        - roda testes com cobertura (oculta pacotes sem teste)"
	@echo "  make test-integration - roda testes de integração (Testcontainers, exige Docker)"
	@echo "  make cover       - gera relatório HTML de cobertura em coverage/"
	@echo "  make cover-integration - cobertura incluindo testes de integração (exige Docker)"
	@echo "  make lint        - roda golangci-lint em todos os serviços"
	@echo "  make smoke       - smoke test end-to-end (exige 'make up' + curl/ffmpeg)"
