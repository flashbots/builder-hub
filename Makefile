# Heavily inspired by Lighthouse: https://github.com/sigp/lighthouse/blob/stable/Makefile
# and Reth: https://github.com/paradigmxyz/reth/blob/main/Makefile
.DEFAULT_GOAL := help

VERSION := $(shell git describe --tags --always --dirty="-dev")

# Admin API auth for curl examples (set ADMIN_AUTH="admin:secret" or similar. can be empty when server is run with --disable-admin-auth)
ADMIN_AUTH ?=
CURL ?= curl -v
CURL_AUTH := $(if $(ADMIN_AUTH),-u $(ADMIN_AUTH),)

# A few colors
RED:=\033[0;31m
BLUE:=\033[0;34m
GREEN:=\033[0;32m
NC:=\033[0m

##@ Help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: v
v: ## Show the version
	@echo "Version: ${VERSION}"

##@ Build

.PHONY: clean
clean: ## Clean the build directory
	rm -rf build/

.PHONY: build
build: ## Build the HTTP server
	@mkdir -p ./build
	go build -trimpath -ldflags "-X github.com/flashbots/builder-hub/common.Version=${VERSION}" -v -o ./build/builder-hub cmd/httpserver/main.go

##@ Test & Development

.PHONY: lt
lt: lint test ## Run linters and tests (always do this!)

.PHONY: fmt
fmt: ## Format the code (use this often)
	gofmt -s -w .
	gci write .
	gofumpt -w -extra .
	go mod tidy

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: test-race
test-race: ## Run tests with race detector
	go test -race ./...

.PHONY: lint
lint: ## Run linters
	gofmt -d -s .
	gofumpt -d -extra .
	go vet ./...
	staticcheck ./...
	golangci-lint run

.PHONY: gofumpt
gofumpt: ## Run gofumpt
	gofumpt -l -w -extra .

.PHONY: cover
cover: ## Run tests with coverage
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -func /tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

.PHONY: cover-html
cover-html: ## Run tests with coverage and open the HTML report
	go test -coverprofile=/tmp/go-sim-lb.cover.tmp ./...
	go tool cover -html=/tmp/go-sim-lb.cover.tmp
	unlink /tmp/go-sim-lb.cover.tmp

.PHONY: docker-httpserver
docker-httpserver: ## Build the HTTP server Docker image
	DOCKER_BUILDKIT=1 docker build \
		--file docker/httpserver/Dockerfile \
		--platform linux/amd64 \
		--build-arg VERSION=${VERSION} \
		--tag builder-hub \
	.

.PHONY: db-dump
db-dump: ## Dump the database contents to file 'database.dump'
	pg_dump -h localhost -U postgres --column-inserts --data-only postgres -f database.dump
	@printf "Database dumped to file: $(GREEN)database.dump$(NC) ✅\n"

.PHONY: dev-db-setup
dev-db-setup: ## Create the basic database entries for testing and development
	@printf "$(BLUE)Create the allow-all measurements $(NC)\n"
	$(CURL) $(CURL_AUTH) --request POST --url http://localhost:8081/api/admin/v1/measurements --data '{"measurement_id": "test1","attestation_type": "test","measurements": {}}'

	@printf "$(BLUE)Enable the measurements $(NC)\n"
	$(CURL) $(CURL_AUTH) --request POST --url http://localhost:8081/api/admin/v1/measurements/activation/test1 --data '{"enabled": true}'

	@printf "$(BLUE)Create the builder $(NC)\n"
	$(CURL) $(CURL_AUTH) --request POST --url http://localhost:8081/api/admin/v1/builders --data '{"name": "test_builder","ip_address": "1.2.3.4", "network": "production"}'

	@printf "$(BLUE)Create the builder configuration $(NC)\n"
	$(CURL) $(CURL_AUTH) --request POST --url http://localhost:8081/api/admin/v1/builders/configuration/test_builder --data '{"dns_name": "foobar-v1.a.b.c","rbuilder": {"extra_data": "FooBar"}}'

	@printf "$(BLUE)Enable the builder $(NC)\n"
	$(CURL) $(CURL_AUTH) --request POST --url http://localhost:8081/api/admin/v1/builders/activation/test_builder --data '{"enabled": true}'
