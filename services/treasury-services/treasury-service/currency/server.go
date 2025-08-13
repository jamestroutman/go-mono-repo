package currency

import (
	"context"

	pb "example.com/go-mono-repo/proto/treasury"
)

// Server implements the CurrencyService gRPC interface
// Spec: docs/specs/003-currency-management.md
type Server struct {
	pb.UnimplementedCurrencyServiceServer
	manager *Manager
}

// NewServer creates a new currency server instance
// Spec: docs/specs/003-currency-management.md
func NewServer(manager *Manager) *Server {
	return &Server{
		manager: manager,
	}
}

// CreateCurrency creates a new currency
// Spec: docs/specs/003-currency-management.md#story-1-create-new-currency
func (s *Server) CreateCurrency(ctx context.Context, req *pb.CreateCurrencyRequest) (*pb.CreateCurrencyResponse, error) {
	currency, err := s.manager.CreateCurrency(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.CreateCurrencyResponse{Currency: currency}, nil
}

// GetCurrency retrieves currency information
// Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
func (s *Server) GetCurrency(ctx context.Context, req *pb.GetCurrencyRequest) (*pb.GetCurrencyResponse, error) {
	currency, err := s.manager.GetCurrency(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.GetCurrencyResponse{Currency: currency}, nil
}

// UpdateCurrency updates currency metadata
// Spec: docs/specs/003-currency-management.md#story-3-update-currency-metadata
func (s *Server) UpdateCurrency(ctx context.Context, req *pb.UpdateCurrencyRequest) (*pb.UpdateCurrencyResponse, error) {
	currency, err := s.manager.UpdateCurrency(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateCurrencyResponse{Currency: currency}, nil
}

// DeactivateCurrency deactivates a currency (soft delete)
// Spec: docs/specs/003-currency-management.md#story-4-deactivate-currency
func (s *Server) DeactivateCurrency(ctx context.Context, req *pb.DeactivateCurrencyRequest) (*pb.DeactivateCurrencyResponse, error) {
	currency, err := s.manager.DeactivateCurrency(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.DeactivateCurrencyResponse{
		Success:  true,
		Currency: currency,
	}, nil
}

// ListCurrencies lists currencies with optional filters
// Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
func (s *Server) ListCurrencies(ctx context.Context, req *pb.ListCurrenciesRequest) (*pb.ListCurrenciesResponse, error) {
	return s.manager.ListCurrencies(ctx, req)
}

// BulkCreateCurrencies creates multiple currencies in a single transaction
// Spec: docs/specs/003-currency-management.md#story-5-bulk-currency-operations
func (s *Server) BulkCreateCurrencies(ctx context.Context, req *pb.BulkCreateCurrenciesRequest) (*pb.BulkCreateCurrenciesResponse, error) {
	return s.manager.BulkCreateCurrencies(ctx, req)
}