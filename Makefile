.PHONY: install-reqs dev ledger-service treasury-service all-services

install-reqs:
	@echo "Checking prerequisites..."
	@if ! command -v brew >/dev/null 2>&1; then \
		echo "Error: Homebrew is not installed."; \
		echo "Please install Homebrew first: https://brew.sh"; \
		exit 1; \
	fi
	@echo "✓ Homebrew is installed"
	@if ! command -v go >/dev/null 2>&1; then \
		echo "Go is not installed. Installing with brew..."; \
		brew install go; \
	else \
		echo "✓ Go is installed"; \
	fi
	@if ! command -v protoc >/dev/null 2>&1; then \
		echo "protoc is not installed. Installing with brew..."; \
		brew install protobuf; \
	else \
		echo "✓ protoc is installed"; \
	fi
	@echo "Installing Go protobuf plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✓ Go protobuf plugins installed"
	@echo "All prerequisites are installed!"

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
	@make install-reqs &
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