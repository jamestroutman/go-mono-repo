.PHONY: install-reqs dev ledger-service all-services

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

ledger-service: install-reqs
	@echo "Starting ledger service..."
	@echo "Generating protobuf code for ledger service..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/treasury-services/ledger-service/proto/ledger_service.proto
	@echo "✓ Protobuf code generated"
	@echo "Running ledger service..."
	@go run ./services/treasury-services/ledger-service/main.go

all-services: ledger-service

dev: ledger-service