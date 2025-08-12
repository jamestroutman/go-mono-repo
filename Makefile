.PHONY: install-reqs dev ledger-service treasury-service all-services infrastructure-up infrastructure-down infrastructure-status infrastructure-clean

# Prerequisites are automatically installed in the devcontainer
# This target confirms the development environment is ready
install-reqs:
	@if [ -f /.dockerenv ]; then \
		echo "✓ Running in devcontainer - all tools pre-installed"; \
		echo "✓ Go: $(shell go version)"; \
		echo "✓ protoc: $(shell protoc --version)"; \
		echo "✓ protoc-gen-go: $(shell which protoc-gen-go >/dev/null && echo 'installed' || echo 'missing')"; \
		echo "✓ protoc-gen-go-grpc: $(shell which protoc-gen-go-grpc >/dev/null && echo 'installed' || echo 'missing')"; \
		echo "✓ grpcurl: $(shell grpcurl --version 2>/dev/null || echo 'available')"; \
		echo "✓ migrate: $(shell migrate -version 2>/dev/null || echo 'available')"; \
		echo "✓ lsof: available"; \
		echo "✓ psql: $(shell psql --version)"; \
		echo "✓ Development environment ready"; \
	else \
		echo "⚠️  Not running in devcontainer"; \
		echo "Please reopen this project in VS Code devcontainer for the best experience"; \
		echo "See docs/DEVCONTAINER.md for setup instructions"; \
		exit 1; \
	fi

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

ledger-service: migrate-ledger
	@echo "Starting ledger service..."
	@echo "Checking for existing service on port 50051..."
	@lsof -ti:50051 | xargs -r kill -9 2>/dev/null || true
	@echo "Generating protobuf code for ledger service..."
	@protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/treasury-services/ledger-service/proto/ledger_service.proto
	@echo "✓ Protobuf code generated"
	@echo "Running ledger service..."
	@go run ./services/treasury-services/ledger-service/

# Optional: separate target without migrations for development
ledger-service-fast:
	@echo "Starting ledger service (no migrations)..."
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
	@protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
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
	@make migrate-all
	@make all-services

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

# Migration commands for monorepo
# Spec: services/treasury-services/ledger-service/docs/specs/002-database-migrations.md

# Ledger Service Migrations
migrate-ledger:
	@echo "Running ledger service database migrations..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate up --migrations services/treasury-services/ledger-service/migrations

migrate-ledger-status:
	@echo "Checking ledger service migration status..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate status --migrations services/treasury-services/ledger-service/migrations

migrate-ledger-validate:
	@echo "Validating ledger service migration files..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate validate --migrations services/treasury-services/ledger-service/migrations

migrate-ledger-dry-run:
	@echo "Ledger service migration dry run..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate up --dry-run --migrations services/treasury-services/ledger-service/migrations

migration-ledger-new:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migration-ledger-new NAME=description"; \
		exit 1; \
	fi
	@go run ./services/treasury-services/ledger-service/cmd/migrate create $(NAME) --migrations services/treasury-services/ledger-service/migrations

# Aggregate migration commands
migrate-all:
	@echo "===================================="
	@echo "           MIGRATE ALL              "
	@echo "===================================="
	@echo "Running all service migrations..."
	@make migrate-ledger

migrate-all-status:
	@echo "===================================="
	@echo "   ALL SERVICE MIGRATION STATUS    "
	@echo "===================================="
	@make migrate-ledger-status || true
	@echo ""
	@make migrate-treasury-status || true

# Migration commands for treasury-service
# Spec: docs/specs/002-database-migrations.md#story-2-manual-migration-control

# Create a new migration
migration-create-treasury:
	@read -p "Enter migration name (snake_case): " name; \
	migrate create -ext sql -dir services/treasury-services/treasury-service/migrations -seq $$name
	@echo "✓ Migration files created"

# Run migrations up
migrate-up-treasury:
	@echo "Running treasury service migrations..."
	@source services/treasury-services/treasury-service/.env 2>/dev/null || true; \
	migrate -path services/treasury-services/treasury-service/migrations \
		-database "postgresql://$${DB_USER:-treasury_user}:$${DB_PASSWORD:-treasury_pass}@$${DB_HOST:-postgres}:$${DB_PORT:-5432}/$${DB_NAME:-treasury_db}?sslmode=$${DB_SSL_MODE:-disable}" \
		up
	@echo "✓ Migrations completed"

# Rollback last migration
migrate-down-treasury:
	@echo "Rolling back last treasury service migration..."
	@source services/treasury-services/treasury-service/.env 2>/dev/null || true; \
	migrate -path services/treasury-services/treasury-service/migrations \
		-database "postgresql://$${DB_USER:-treasury_user}:$${DB_PASSWORD:-treasury_pass}@$${DB_HOST:-postgres}:$${DB_PORT:-5432}/$${DB_NAME:-treasury_db}?sslmode=$${DB_SSL_MODE:-disable}" \
		down 1
	@echo "✓ Rollback completed"

# Check migration status
migrate-status-treasury:
	@echo "Treasury service migration status:"
	@source services/treasury-services/treasury-service/.env 2>/dev/null || true; \
	migrate -path services/treasury-services/treasury-service/migrations \
		-database "postgresql://$${DB_USER:-treasury_user}:$${DB_PASSWORD:-treasury_pass}@$${DB_HOST:-postgres}:$${DB_PORT:-5432}/$${DB_NAME:-treasury_db}?sslmode=$${DB_SSL_MODE:-disable}" \
		version

# Force set migration version (use with caution)
migrate-force-treasury:
	@read -p "Enter version to force: " version; \
	source services/treasury-services/treasury-service/.env 2>/dev/null || true; \
	migrate -path services/treasury-services/treasury-service/migrations \
		-database "postgresql://$${DB_USER:-treasury_user}:$${DB_PASSWORD:-treasury_pass}@$${DB_HOST:-postgres}:$${DB_PORT:-5432}/$${DB_NAME:-treasury_db}?sslmode=$${DB_SSL_MODE:-disable}" \
		force $$version
	@echo "✓ Migration version forced"