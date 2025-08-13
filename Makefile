.PHONY: install-reqs dev run-ledger run-treasury run-all migrate migrate-ledger migrate-treasury migrate-status migrate-new-ledger migrate-new-treasury health health-ledger health-treasury liveness liveness-ledger liveness-treasury run-tests run-integration-tests

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

# Infrastructure services are automatically started with devcontainer
# See .devcontainer/docker-compose.yml for service configuration

run-ledger: migrate-ledger
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
run-ledger-fast:
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

run-treasury: migrate-treasury
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

run-all: 
	@echo "Starting all services..."
	@make run-ledger &
	@make run-treasury &
	@wait

dev:
	@echo "Starting development environment..."
	@echo "Running all migrations..."
	@make migrate
	@echo "Starting all services..."
	@make run-all

# Health check commands
# Spec: docs/specs/003-health-check-liveness.md
health-ledger:
	@echo "Checking ledger service health..."
	@grpcurl -plaintext localhost:50051 ledger.Health/GetHealth

liveness-ledger:
	@echo "Checking ledger service liveness..."
	@grpcurl -plaintext localhost:50051 ledger.Health/GetLiveness

health-treasury:
	@echo "Checking treasury service health..."
	@grpcurl -plaintext localhost:50052 treasury.Health/GetHealth

liveness-treasury:
	@echo "Checking treasury service liveness..."
	@grpcurl -plaintext localhost:50052 treasury.Health/GetLiveness

# Combined health checks
health:
	@echo "==================================="
	@echo "   CHECKING ALL SERVICE HEALTH    "
	@echo "==================================="
	@make health-ledger || true
	@echo ""
	@make health-treasury || true

liveness:
	@echo "==================================="
	@echo "  CHECKING ALL SERVICE LIVENESS   "
	@echo "==================================="
	@make liveness-ledger || true
	@echo ""
	@make liveness-treasury || true

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

migrate-new-ledger:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-new-ledger NAME=description"; \
		exit 1; \
	fi
	@go run ./services/treasury-services/ledger-service/cmd/migrate create $(NAME) --migrations services/treasury-services/ledger-service/migrations

# Aggregate migration commands
migrate:
	@echo "===================================="
	@echo "           MIGRATE ALL              "
	@echo "===================================="
	@echo "Running all service migrations..."
	@make migrate-ledger
	@make migrate-treasury

migrate-status:
	@echo "===================================="
	@echo "   ALL SERVICE MIGRATION STATUS    "
	@echo "===================================="
	@make migrate-ledger-status || true
	@echo ""
	@make migrate-treasury-status || true

# Migration commands for treasury-service
# Spec: docs/specs/002-database-migrations.md#story-2-manual-migration-control

# Create a new migration
migrate-new-treasury:
	@read -p "Enter migration name (snake_case): " name; \
	migrate create -ext sql -dir services/treasury-services/treasury-service/migrations -seq $$name
	@echo "✓ Migration files created"

# Run migrations up (alias for consistency)
migrate-treasury: migrate-up-treasury

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

# Test commands
run-tests:
	@echo "====================================="
	@echo "        RUNNING ALL UNIT TESTS       "
	@echo "====================================="
	@find services -name "*_test.go" ! -name "*_integration_test.go" -exec dirname {} \; | sort | uniq | while read dir; do \
		echo "Running unit tests in $$dir..."; \
		go test -v ./$$dir -run "^Test[^I].*" || exit 1; \
	done
	@echo "✓ All unit tests completed"

run-integration-tests:
	@echo "====================================="
	@echo "    RUNNING ALL INTEGRATION TESTS    "
	@echo "====================================="
	@find services -name "*_integration_test.go" -exec dirname {} \; | sort | uniq | while read dir; do \
		echo "Running integration tests in $$dir..."; \
		go test -v ./$$dir -run ".*Integration.*" || exit 1; \
	done
	@echo "✓ All integration tests completed"