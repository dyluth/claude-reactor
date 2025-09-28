# Claude-Reactor Makefile
# Comprehensive build and test automation for the Docker containerization system

# Configuration
DOCKER_REGISTRY ?= ghcr.io/dyluth
IMAGE_PREFIX ?= claude-reactor
REMOTE_IMAGE_PREFIX = $(DOCKER_REGISTRY)/$(IMAGE_PREFIX)
VARIANTS = base go full cloud k8s
PROJECT_NAME = claude-reactor

# Architecture detection
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),x86_64)
    ARCHITECTURE := amd64
    DOCKER_PLATFORM := linux/amd64
else ifeq ($(UNAME_M),amd64)
    ARCHITECTURE := amd64
    DOCKER_PLATFORM := linux/amd64
else ifeq ($(UNAME_M),arm64)
    ARCHITECTURE := arm64
    DOCKER_PLATFORM := linux/arm64
else ifeq ($(UNAME_M),aarch64)
    ARCHITECTURE := arm64
    DOCKER_PLATFORM := linux/arm64
else
    $(error Unsupported architecture: $(UNAME_M). Supported: x86_64, amd64, arm64, aarch64)
endif

# Build settings
DOCKER_BUILD_ARGS ?= --no-cache=false
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION ?= latest

# Colors for output
BLUE = \033[0;34m
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help message
	@echo "$(BLUE)Claude-Reactor Build System$(NC)"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(YELLOW)<target>$(NC)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(BLUE)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(GREEN)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(YELLOW)Typical Workflows:$(NC)"
	@echo "  $(BLUE)Development:$(NC)"
	@echo "    make run              # Start container (auto-detects variant)"
	@echo "    make config           # Check current configuration"
	@echo "    make run-go           # Force Go variant"
	@echo ""
	@echo "  $(BLUE)Reactor-Fabric:$(NC)"
	@echo "    make build-fabric     # Build reactor-fabric binaries"
	@echo "    make fabric-config    # Generate default configuration"
	@echo "    make run-fabric       # Start orchestrator"
	@echo "    make fabric-validate  # Validate configuration"
	@echo ""
	@echo "  $(BLUE)Testing:$(NC)"
	@echo "    make test             # Run complete test suite"
	@echo "    make demo             # Interactive feature demo"
	@echo "    make test-unit        # Quick unit tests only"
	@echo ""
	@echo "  $(BLUE)Building:$(NC)"
	@echo "    make build            # Build binaries for all platforms"
	@echo "    make build-local      # Build binary for current platform only"
	@echo "    make build-all        # Build Docker container variants"
	@echo ""
	@echo "  $(BLUE)Installation:$(NC)"
	@echo "    make install          # Install to system PATH"
	@echo "    make uninstall        # Remove from system PATH"
	@echo ""
	@echo "  $(BLUE)Maintenance:$(NC)"
	@echo "    make clean-all        # Complete cleanup"
	@echo "    make benchmark        # Container size analysis"

.PHONY: info
info: ## Show project information
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "  Name:           $(PROJECT_NAME)"
	@echo "  Version:        $(VERSION)"
	@echo "  Git Commit:     $(GIT_COMMIT)"
	@echo "  Build Date:     $(BUILD_DATE)"
	@echo "  Architecture:   $(ARCHITECTURE)"
	@echo "  Docker Platform: $(DOCKER_PLATFORM)"
	@echo "  Variants:       $(VARIANTS)"
	@echo ""

##@ Container Building

.PHONY: build-base
build-base: ## Build base container variant
	@echo "$(BLUE)Building base variant...$(NC)"
	docker build $(DOCKER_BUILD_ARGS) \
		--platform $(DOCKER_PLATFORM) \
		--target base \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t $(IMAGE_PREFIX)-base-$(ARCHITECTURE):$(VERSION) \
		-t $(IMAGE_PREFIX)-base-$(ARCHITECTURE):latest .
	@echo "$(GREEN)✓ Base variant built successfully$(NC)"

.PHONY: build-go
build-go: ## Build Go container variant
	@echo "$(BLUE)Building Go variant...$(NC)"
	docker build $(DOCKER_BUILD_ARGS) \
		--platform $(DOCKER_PLATFORM) \
		--target go \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t $(IMAGE_PREFIX)-go-$(ARCHITECTURE):$(VERSION) \
		-t $(IMAGE_PREFIX)-go-$(ARCHITECTURE):latest .
	@echo "$(GREEN)✓ Go variant built successfully$(NC)"

.PHONY: build-full
build-full: ## Build full container variant (Go + Rust + Java + DBs)
	@echo "$(BLUE)Building full variant...$(NC)"
	docker build $(DOCKER_BUILD_ARGS) \
		--platform $(DOCKER_PLATFORM) \
		--target full \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t $(IMAGE_PREFIX)-full-$(ARCHITECTURE):$(VERSION) \
		-t $(IMAGE_PREFIX)-full-$(ARCHITECTURE):latest .
	@echo "$(GREEN)✓ Full variant built successfully$(NC)"

.PHONY: build-cloud
build-cloud: ## Build cloud container variant (Full + cloud CLIs)
	@echo "$(BLUE)Building cloud variant...$(NC)"
	docker build $(DOCKER_BUILD_ARGS) \
		--platform $(DOCKER_PLATFORM) \
		--target cloud \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t $(IMAGE_PREFIX)-cloud-$(ARCHITECTURE):$(VERSION) \
		-t $(IMAGE_PREFIX)-cloud-$(ARCHITECTURE):latest .
	@echo "$(GREEN)✓ Cloud variant built successfully$(NC)"

.PHONY: build-k8s
build-k8s: ## Build Kubernetes container variant (Full + K8s tools)
	@echo "$(BLUE)Building Kubernetes variant...$(NC)"
	docker build $(DOCKER_BUILD_ARGS) \
		--platform $(DOCKER_PLATFORM) \
		--target k8s \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg GIT_COMMIT="$(GIT_COMMIT)" \
		-t $(IMAGE_PREFIX)-k8s-$(ARCHITECTURE):$(VERSION) \
		-t $(IMAGE_PREFIX)-k8s-$(ARCHITECTURE):latest .
	@echo "$(GREEN)✓ Kubernetes variant built successfully$(NC)"

.PHONY: build-all
build-all: build-base build-go build-full ## Build core container variants (base, go, full)
	@echo "$(GREEN)✓ All core variants built successfully$(NC)"

.PHONY: build-extended
build-extended: build-all build-cloud build-k8s ## Build all container variants including cloud and k8s
	@echo "$(GREEN)✓ All variants built successfully$(NC)"

##@ Testing

.PHONY: test-go-unit
test-go-unit: ## Run Go unit tests
	@echo "$(BLUE)Running Go unit tests...$(NC)"
	@go test -v ./internal/... ./pkg/... ./cmd/... || (echo "$(RED)✗ Go unit tests failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Go unit tests completed$(NC)"

.PHONY: test-go-integration
test-go-integration: ## Run Go integration tests (requires Docker)
	@echo "$(BLUE)Running Go integration tests...$(NC)"
	@go test -v -tags=integration ./internal/... ./pkg/... ./cmd/...
	@echo "$(GREEN)✓ Go integration tests completed$(NC)"

.PHONY: test-go-all
test-go-all: test-go-unit test-go-integration ## Run all Go tests
	@echo "$(GREEN)✓ All Go tests completed$(NC)"

.PHONY: test-unit
test-unit: test-go-unit ## Run unit tests (fast) - includes both bash and Go tests
	@echo "$(BLUE)Running bash unit tests...$(NC)"
	@./tests/test-runner.sh --unit
	@echo "$(GREEN)✓ Unit tests completed$(NC)"

.PHONY: test-phase0
test-phase0: build ## Run Phase 0 feature validation tests
	@echo "$(BLUE)Running Phase 0 feature validation tests...$(NC)"
	@./tests/phase0/test-phase0-quick.sh
	@echo "$(GREEN)✓ Phase 0 tests completed$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./tests/test-runner.sh --integration && echo "$(GREEN)✓ Integration tests completed$(NC)"

.PHONY: test-integration-quick
test-integration-quick: ## Run integration tests without Docker builds
	@echo "$(BLUE)Running integration tests (quick mode)...$(NC)"
	@./tests/test-runner.sh --integration --quick && echo "$(GREEN)✓ Quick integration tests completed$(NC)"

.PHONY: test
test: build test-unit test-integration-quick ## Run complete test suite (unit + quick integration)
	@echo "$(GREEN)✓ Complete test suite passed$(NC)"

.PHONY: test-full
test-full: build-all test-unit test-integration ## Build containers and run full test suite
	@echo "$(GREEN)✓ Full test suite with builds completed$(NC)"

.PHONY: demo
demo: ## Run interactive feature demonstration
	@echo "$(BLUE)Starting interactive demo...$(NC)"
	./tests/demo.sh

.PHONY: demo-auto
demo-auto: ## Run automated feature demonstration
	@echo "$(BLUE)Running automated demo...$(NC)"
	./tests/demo.sh --auto

.PHONY: demo-quick
demo-quick: ## Run quick demo without Docker builds
	@echo "$(BLUE)Running quick demo...$(NC)"
	@./tests/demo.sh --quick --auto && echo "$(GREEN)✓ Demo completed$(NC)"

##@ Go Development

.PHONY: build
build: go-mod-tidy build-apps ## Build all applications (claude-reactor and reactor-fabric)

.PHONY: build-local
build-local: go-mod-tidy ## Build claude-reactor binary for current platform only (faster)
	@echo "$(BLUE)Building claude-reactor for current platform...$(NC)"
	@go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o claude-reactor ./cmd/claude-reactor
	@echo "$(GREEN)✓ Local binary built: claude-reactor$(NC)"

.PHONY: go-build
go-build: build ## Build Go binary (alias for build)
	@echo "$(GREEN)✓ Go binaries built in dist/$(NC)"

.PHONY: build-reactor
build-reactor: go-mod-tidy ## Build claude-reactor binaries for all major architectures
	@echo "$(BLUE)Building claude-reactor binaries for all platforms...$(NC)"
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-linux-amd64 ./cmd/claude-reactor
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-linux-arm64 ./cmd/claude-reactor
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-darwin-amd64 ./cmd/claude-reactor
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-darwin-arm64 ./cmd/claude-reactor
	@echo "$(GREEN)✓ Claude-reactor binaries built in dist/$(NC)"

.PHONY: build-fabric
build-fabric: go-mod-tidy ## Build reactor-fabric binaries for all major architectures
	@echo "$(BLUE)Building reactor-fabric binaries for all platforms...$(NC)"
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/reactor-fabric-linux-amd64 ./cmd/reactor-fabric
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/reactor-fabric-linux-arm64 ./cmd/reactor-fabric
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/reactor-fabric-darwin-amd64 ./cmd/reactor-fabric
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/reactor-fabric-darwin-arm64 ./cmd/reactor-fabric
	@echo "$(GREEN)✓ Reactor-fabric binaries built in dist/$(NC)"

.PHONY: build-apps
build-apps: build-reactor build-fabric build-test-client ## Build all applications

.PHONY: build-test-client
build-test-client: go-mod-tidy ## Build test client for reactor-fabric
	@echo "$(BLUE)Building test-client binary...$(NC)"
	@mkdir -p dist
	@go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/test-client ./cmd/test-client
	@echo "$(GREEN)✓ Test client built: dist/test-client$(NC)"

.PHONY: install
install: ## Install claude-reactor to system PATH using INSTALL script
	@echo "$(BLUE)Installing claude-reactor to system PATH...$(NC)"
	@./INSTALL

.PHONY: uninstall 
uninstall: ## Remove claude-reactor from system PATH using INSTALL script
	@echo "$(BLUE)Removing claude-reactor from system PATH...$(NC)"
	@./INSTALL --uninstall

.PHONY: go-build-all
go-build-all: ## Build Go binaries for multiple platforms
	@echo "$(BLUE)Building Go binaries for multiple platforms...$(NC)"
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-linux-amd64 ./cmd/claude-reactor
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-linux-arm64 ./cmd/claude-reactor
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-darwin-amd64 ./cmd/claude-reactor
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)" \
		-o dist/claude-reactor-darwin-arm64 ./cmd/claude-reactor
	@echo "$(GREEN)✓ Multi-platform binaries built in dist/$(NC)"

.PHONY: go-lint
go-lint: ## Run Go linting
	@echo "$(BLUE)Running Go linting...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./... ; \
		echo "$(GREEN)✓ Go linting completed$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not found, running basic go vet$(NC)" ; \
		go vet ./... ; \
	fi

.PHONY: go-fmt
go-fmt: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Go code formatted$(NC)"

.PHONY: go-mod-tidy
go-mod-tidy: ## Tidy Go modules
	@echo "$(BLUE)Tidying Go modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Go modules tidied$(NC)"

.PHONY: go-deps
go-deps: ## Download Go dependencies
	@echo "$(BLUE)Downloading Go dependencies...$(NC)"
	@go mod download
	@echo "$(GREEN)✓ Go dependencies downloaded$(NC)"

##@ Development

.PHONY: dev-setup
dev-setup: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@chmod +x tests/*.sh
	@chmod +x tests/*/*.sh
	@make go-deps
	@echo "$(GREEN)✓ Development environment ready$(NC)"

.PHONY: lint
lint: ## Run shell script linting (requires shellcheck)
	@echo "$(BLUE)Running shell script linting...$(NC)"
	@if command -v shellcheck >/dev/null 2>&1; then \
		find . -name "*.sh" -not -path "./tests/fixtures/*" -exec shellcheck {} \; ; \
		shellcheck claude-reactor ; \
		echo "$(GREEN)✓ Linting completed$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ shellcheck not found, skipping linting$(NC)" ; \
	fi

.PHONY: format
format: ## Format shell scripts (requires shfmt)
	@echo "$(BLUE)Formatting shell scripts...$(NC)"
	@if command -v shfmt >/dev/null 2>&1; then \
		find . -name "*.sh" -not -path "./tests/fixtures/*" -exec shfmt -w -i 4 {} \; ; \
		shfmt -w -i 4 claude-reactor ; \
		echo "$(GREEN)✓ Formatting completed$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ shfmt not found, skipping formatting$(NC)" ; \
	fi

.PHONY: validate
validate: dev-setup lint test-unit ## Validate code quality and basic functionality
	@echo "$(GREEN)✓ Code validation completed$(NC)"

.PHONY: test-persistence
test-persistence: ## Test Claude CLI configuration persistence across container restarts
	@echo "$(BLUE)Testing Claude CLI configuration persistence...$(NC)"
	@./tests/integration/test-persistence.sh

##@ Container Management

.PHONY: run-fabric
run-fabric: ## Start reactor-fabric orchestrator
	@echo "$(BLUE)Starting Reactor-Fabric orchestrator...$(NC)"
	@if [ -f "./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE)" ]; then \
		./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE) start ; \
	elif [ -f "./reactor-fabric" ]; then \
		./reactor-fabric start ; \
	else \
		echo "$(RED)Error: reactor-fabric binary not found. Run 'make build-fabric' first.$(NC)" ; \
		exit 1 ; \
	fi

.PHONY: fabric-config
fabric-config: ## Generate default reactor-fabric configuration
	@echo "$(BLUE)Generating default reactor-fabric configuration...$(NC)"
	@if [ -f "./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE)" ]; then \
		./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE) init ; \
	elif [ -f "./reactor-fabric" ]; then \
		./reactor-fabric init ; \
	else \
		echo "$(RED)Error: reactor-fabric binary not found. Run 'make build-fabric' first.$(NC)" ; \
		exit 1 ; \
	fi

.PHONY: fabric-validate
fabric-validate: ## Validate reactor-fabric configuration
	@echo "$(BLUE)Validating reactor-fabric configuration...$(NC)"
	@if [ -f "./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE)" ]; then \
		./dist/reactor-fabric-$(shell uname -s | tr '[:upper:]' '[:lower:]')-$(ARCHITECTURE) validate ; \
	elif [ -f "./reactor-fabric" ]; then \
		./reactor-fabric validate ; \
	else \
		echo "$(RED)Error: reactor-fabric binary not found. Run 'make build-fabric' first.$(NC)" ; \
		exit 1 ; \
	fi

.PHONY: run-base
run-base: ## Run base variant container (delegates to claude-reactor)
	@echo "$(BLUE)Starting base variant container...$(NC)"
	./claude-reactor run --image base

.PHONY: run-go
run-go: ## Run Go variant container (delegates to claude-reactor)
	@echo "$(BLUE)Starting Go variant container...$(NC)"
	./claude-reactor run --image go

.PHONY: run-full
run-full: ## Run full variant container (delegates to claude-reactor)
	@echo "$(BLUE)Starting full variant container...$(NC)"
	./claude-reactor run --image full

.PHONY: run-cloud
run-cloud: ## Run cloud variant container (delegates to claude-reactor)
	@echo "$(BLUE)Starting cloud variant container...$(NC)"
	./claude-reactor run --image cloud

.PHONY: run-k8s
run-k8s: ## Run Kubernetes variant container (delegates to claude-reactor)
	@echo "$(BLUE)Starting Kubernetes variant container...$(NC)"
	./claude-reactor run --image k8s

.PHONY: run
run: ## Run container with auto-detected or saved variant (delegates to claude-reactor)
	@echo "$(BLUE)Starting container with smart variant detection...$(NC)"
	./claude-reactor

.PHONY: stop-all
stop-all: ## Stop all running claude-agent containers (delegates to claude-reactor cleanup)
	@echo "$(BLUE)Stopping all claude-agent containers...$(NC)"
	@./claude-reactor clean > /dev/null 2>&1 || docker ps --format '{{.Names}}' | grep '^claude-agent' | xargs -r docker stop
	@echo "$(GREEN)✓ All containers stopped$(NC)"

.PHONY: logs
logs: ## Show logs from most recent claude-agent container
	@CONTAINER=$$(docker ps -a --format '{{.Names}}' | grep '^claude-agent' | head -1) ; \
	if [ -n "$$CONTAINER" ]; then \
		echo "$(BLUE)Showing logs for: $$CONTAINER$(NC)" ; \
		docker logs $$CONTAINER ; \
	else \
		echo "$(YELLOW)No claude-agent containers found$(NC)" ; \
	fi

.PHONY: config
config: ## Show current claude-reactor configuration
	@./claude-reactor config show

.PHONY: variants
variants: ## List available container variants
	@./claude-reactor debug info

##@ Cleanup

.PHONY: clean-containers
clean-containers: ## Remove all claude-agent containers (delegates to claude-reactor)
	@echo "$(BLUE)Removing claude-agent containers...$(NC)"
	@./claude-reactor clean > /dev/null 2>&1 || docker ps -a --format '{{.Names}}' | grep '^claude-agent' | xargs -r docker rm -f
	@echo "$(GREEN)✓ Containers cleaned$(NC)"

.PHONY: clean-images
clean-images: ## Remove all claude-reactor images
	@echo "$(BLUE)Removing claude-reactor images...$(NC)"
	@docker images --format '{{.Repository}}:{{.Tag}}' | grep '^claude-reactor' | xargs -r docker rmi -f
	@echo "$(GREEN)✓ Images cleaned$(NC)"

.PHONY: clean-test
clean-test: ## Clean test artifacts (delegates to test-runner)
	@echo "$(BLUE)Cleaning test artifacts...$(NC)"
	@./tests/test-runner.sh --clean > /dev/null 2>&1 || ( \
		rm -rf tests/fixtures && \
		find tests -name ".claude-reactor" -delete 2>/dev/null || true && \
		find tests -name "go.mod" -delete 2>/dev/null || true && \
		find tests -name "*.tmp" -delete 2>/dev/null || true \
	)
	@echo "$(GREEN)✓ Test artifacts cleaned$(NC)"

.PHONY: clean-config
clean-config: ## Remove local claude-reactor configuration
	@echo "$(BLUE)Removing local configuration...$(NC)"
	@rm -f .claude-reactor
	@echo "$(GREEN)✓ Configuration cleaned$(NC)"

.PHONY: clean-all
clean-all: clean-containers clean-images clean-test clean-config ## Complete cleanup (containers, images, test artifacts, config)
	@echo "$(GREEN)✓ Complete cleanup finished$(NC)"

##@ CI/CD

.PHONY: ci-test
ci-test: dev-setup test-unit test-integration-quick ## Run CI-appropriate tests
	@echo "$(GREEN)✓ CI tests completed$(NC)"

.PHONY: ci-build
ci-build: go-mod-tidy build-all ## Build core variants for CI
	@echo "$(GREEN)✓ CI build completed$(NC)"

.PHONY: ci-full
ci-full: ci-build ci-test ## Complete CI pipeline
	@echo "$(GREEN)✓ CI pipeline completed$(NC)"

##@ Registry Management

.PHONY: registry-login
registry-login: ## Log in to the Docker registry
	@echo "$(BLUE)Logging in to $(DOCKER_REGISTRY)...$(NC)"
	@docker login $(DOCKER_REGISTRY)
	@echo "$(GREEN)✓ Registry login completed$(NC)"

.PHONY: push-base
push-base: build-base registry-login ## Build and push base variant to registry
	@echo "$(BLUE)Pushing base variant to registry...$(NC)"
	@docker tag $(IMAGE_PREFIX)-base-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-base:latest
	@docker tag $(IMAGE_PREFIX)-base-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-base:$(VERSION)
	@docker push $(REMOTE_IMAGE_PREFIX)-base:latest
	@docker push $(REMOTE_IMAGE_PREFIX)-base:$(VERSION)
	@echo "$(GREEN)✓ Base variant pushed to registry$(NC)"

.PHONY: push-go
push-go: build-go registry-login ## Build and push Go variant to registry
	@echo "$(BLUE)Pushing Go variant to registry...$(NC)"
	@docker tag $(IMAGE_PREFIX)-go-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-go:latest
	@docker tag $(IMAGE_PREFIX)-go-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-go:$(VERSION)
	@docker push $(REMOTE_IMAGE_PREFIX)-go:latest
	@docker push $(REMOTE_IMAGE_PREFIX)-go:$(VERSION)
	@echo "$(GREEN)✓ Go variant pushed to registry$(NC)"

.PHONY: push-full
push-full: build-full registry-login ## Build and push full variant to registry
	@echo "$(BLUE)Pushing full variant to registry...$(NC)"
	@docker tag $(IMAGE_PREFIX)-full-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-full:latest
	@docker tag $(IMAGE_PREFIX)-full-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-full:$(VERSION)
	@docker push $(REMOTE_IMAGE_PREFIX)-full:latest
	@docker push $(REMOTE_IMAGE_PREFIX)-full:$(VERSION)
	@echo "$(GREEN)✓ Full variant pushed to registry$(NC)"

.PHONY: push-cloud
push-cloud: build-cloud registry-login ## Build and push cloud variant to registry
	@echo "$(BLUE)Pushing cloud variant to registry...$(NC)"
	@docker tag $(IMAGE_PREFIX)-cloud-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-cloud:latest
	@docker tag $(IMAGE_PREFIX)-cloud-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-cloud:$(VERSION)
	@docker push $(REMOTE_IMAGE_PREFIX)-cloud:latest
	@docker push $(REMOTE_IMAGE_PREFIX)-cloud:$(VERSION)
	@echo "$(GREEN)✓ Cloud variant pushed to registry$(NC)"

.PHONY: push-k8s
push-k8s: build-k8s registry-login ## Build and push Kubernetes variant to registry
	@echo "$(BLUE)Pushing Kubernetes variant to registry...$(NC)"
	@docker tag $(IMAGE_PREFIX)-k8s-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-k8s:latest
	@docker tag $(IMAGE_PREFIX)-k8s-$(ARCHITECTURE):latest $(REMOTE_IMAGE_PREFIX)-k8s:$(VERSION)
	@docker push $(REMOTE_IMAGE_PREFIX)-k8s:latest
	@docker push $(REMOTE_IMAGE_PREFIX)-k8s:$(VERSION)
	@echo "$(GREEN)✓ Kubernetes variant pushed to registry$(NC)"

.PHONY: push-all
push-all: push-base push-go push-full ## Build and push core variants to registry
	@echo "$(GREEN)✓ All core variants pushed to registry$(NC)"

.PHONY: push-extended
push-extended: push-all push-cloud push-k8s ## Build and push all variants to registry
	@echo "$(GREEN)✓ All variants pushed to registry$(NC)"

.PHONY: pull-base
pull-base: ## Pull base variant from registry
	@echo "$(BLUE)Pulling base variant from registry...$(NC)"
	@docker pull $(REMOTE_IMAGE_PREFIX)-base:latest
	@docker tag $(REMOTE_IMAGE_PREFIX)-base:latest $(IMAGE_PREFIX)-base-$(ARCHITECTURE):latest
	@echo "$(GREEN)✓ Base variant pulled from registry$(NC)"

.PHONY: pull-go
pull-go: ## Pull Go variant from registry
	@echo "$(BLUE)Pulling Go variant from registry...$(NC)"
	@docker pull $(REMOTE_IMAGE_PREFIX)-go:latest
	@docker tag $(REMOTE_IMAGE_PREFIX)-go:latest $(IMAGE_PREFIX)-go-$(ARCHITECTURE):latest
	@echo "$(GREEN)✓ Go variant pulled from registry$(NC)"

.PHONY: pull-full
pull-full: ## Pull full variant from registry
	@echo "$(BLUE)Pulling full variant from registry...$(NC)"
	@docker pull $(REMOTE_IMAGE_PREFIX)-full:latest
	@docker tag $(REMOTE_IMAGE_PREFIX)-full:latest $(IMAGE_PREFIX)-full-$(ARCHITECTURE):latest
	@echo "$(GREEN)✓ Full variant pulled from registry$(NC)"

.PHONY: pull-cloud
pull-cloud: ## Pull cloud variant from registry
	@echo "$(BLUE)Pulling cloud variant from registry...$(NC)"
	@docker pull $(REMOTE_IMAGE_PREFIX)-cloud:latest
	@docker tag $(REMOTE_IMAGE_PREFIX)-cloud:latest $(IMAGE_PREFIX)-cloud-$(ARCHITECTURE):latest
	@echo "$(GREEN)✓ Cloud variant pulled from registry$(NC)"

.PHONY: pull-k8s
pull-k8s: ## Pull Kubernetes variant from registry
	@echo "$(BLUE)Pulling Kubernetes variant from registry...$(NC)"
	@docker pull $(REMOTE_IMAGE_PREFIX)-k8s:latest
	@docker tag $(REMOTE_IMAGE_PREFIX)-k8s:latest $(IMAGE_PREFIX)-k8s-$(ARCHITECTURE):latest
	@echo "$(GREEN)✓ Kubernetes variant pulled from registry$(NC)"

.PHONY: pull-all
pull-all: pull-base pull-go pull-full ## Pull core variants from registry
	@echo "$(GREEN)✓ All core variants pulled from registry$(NC)"

.PHONY: pull-extended
pull-extended: pull-all pull-cloud pull-k8s ## Pull all variants from registry
	@echo "$(GREEN)✓ All variants pulled from registry$(NC)"

##@ Advanced

.PHONY: benchmark
benchmark: build-all ## Run container size and performance benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@echo "Container sizes:"
	@for variant in $(VARIANTS); do \
		if docker image inspect $(IMAGE_PREFIX)-$$variant-$(ARCHITECTURE):latest >/dev/null 2>&1; then \
			size=$$(docker image inspect $(IMAGE_PREFIX)-$$variant-$(ARCHITECTURE):latest --format='{{.Size}}' | numfmt --to=iec) ; \
			printf "  %-8s %s\n" "$$variant:" "$$size" ; \
		fi \
	done
	@echo ""
	@echo "$(GREEN)✓ Benchmarks completed$(NC)"

.PHONY: security-scan
security-scan: ## Run security scans on containers (requires trivy)
	@echo "$(BLUE)Running security scans...$(NC)"
	@if command -v trivy >/dev/null 2>&1; then \
		for variant in base go full; do \
			if docker image inspect $(IMAGE_PREFIX)-$$variant-$(ARCHITECTURE):latest >/dev/null 2>&1; then \
				echo "Scanning $$variant variant..." ; \
				trivy image --exit-code 0 $(IMAGE_PREFIX)-$$variant-$(ARCHITECTURE):latest ; \
			fi ; \
		done ; \
		echo "$(GREEN)✓ Security scans completed$(NC)" ; \
	else \
		echo "$(YELLOW)⚠ trivy not found, skipping security scans$(NC)" ; \
	fi

# Include local overrides if they exist
-include Makefile.local