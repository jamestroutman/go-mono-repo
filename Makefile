.PHONY: install-reqs dev ledger-service treasury-service all-services infrastructure-up infrastructure-down infrastructure-status infrastructure-clean

# Note: Prerequisites are now automatically installed in the devcontainer
# This target is kept for backwards compatibility but is no longer needed
install-reqs:
	@echo "Prerequisites check..."
	@if [ -f /.dockerenv ]; then \
		echo "✓ Running in devcontainer - all tools pre-installed"; \
	else \
		echo "⚠️  Not running in devcontainer"; \
		echo "Please reopen this project in VS Code devcontainer for the best experience"; \
		echo "See docs/DEVCONTAINER.md for setup instructions"; \
	fi
	@echo "Checking tool versions..."
	@go version || echo "Go not found"
	@protoc --version || echo "protoc not found"
	@echo "Done!"

# Infrastructure commands - These should be run from the host machine
# Note: Infrastructure is automatically started when using devcontainer
infrastructure-up:
	@echo "Note: Infrastructure is automatically started with devcontainer"
	@echo "Starting infrastructure services manually..."
	@docker compose -f .devcontainer/docker-compose.yml up -d postgres immudb
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in $$(seq 1 30); do \
		docker compose -f .devcontainer/docker-compose.yml exec -T postgres pg_isready -U postgres >/dev/null 2>&1 && break || \
		(echo "Waiting for PostgreSQL... ($$i/30)" && sleep 2); \
	done
	@echo "✓ Infrastructure services are running"

infrastructure-down:
	@echo "Stopping infrastructure services..."
	@docker compose -f .devcontainer/docker-compose.yml down
	@echo "✓ Infrastructure services stopped"

infrastructure-status:
	@echo "Infrastructure service status:"
	@docker compose -f .devcontainer/docker-compose.yml ps

infrastructure-clean:
	@echo "Cleaning infrastructure (removing volumes)..."
	@docker compose -f .devcontainer/docker-compose.yml down -v
	@echo "✓ Infrastructure cleaned"

ledger-service:
	@echo "Starting ledger service..."
	@echo "Checking for existing service on port 50051..."
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
	@echo "Generating protobuf code for ledger service..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/treasury-services/ledger-service/proto/ledger_service.proto
	@echo "✓ Protobuf code generated"
	@echo "Running ledger service..."
	@go run ./services/treasury-services/ledger-service/

treasury-service:
	@echo "Starting treasury service..."
	@echo "Checking for existing service on port 50052..."
	@lsof -ti:50052 | xargs -r kill -9 2>/dev/null || true
	@echo "Generating protobuf code for treasury service..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/treasury-services/treasury-service/proto/treasury_service.proto
	@echo "✓ Protobuf code generated"
	@echo "Running treasury service..."
	@go run ./services/treasury-services/treasury-service/

all-services: 
	@echo "Starting all services..."
	@make ledger-service &
	@make treasury-service &
	@wait

dev:
	@echo "Starting development services..."
	@make ledger-service

# Health check commands
# Spec: docs/specs/003-health-check-liveness.md
health-check-ledger:
	@echo "Checking ledger service health..."
	@grpcurl -plaintext localhost:50051 ledger.Health/GetHealth

liveness-check-ledger:
	@echo "Checking ledger service liveness..."
	@grpcurl -plaintext localhost:50051 ledger.Health/GetLiveness

health-check-treasury:
	@echo "Checking treasury service health..."
	@grpcurl -plaintext localhost:50052 treasury.Health/GetHealth

liveness-check-treasury:
	@echo "Checking treasury service liveness..."
	@grpcurl -plaintext localhost:50052 treasury.Health/GetLiveness

# Combined health checks
health-check-all:
	@echo "==================================="
	@echo "   CHECKING ALL SERVICE HEALTH    "
	@echo "==================================="
	@make health-check-ledger || true
	@echo ""
	@make health-check-treasury || true

liveness-check-all:
	@echo "==================================="
	@echo "  CHECKING ALL SERVICE LIVENESS   "
	@echo "==================================="
	@make liveness-check-ledger || true
	@echo ""
	@make liveness-check-treasury || true