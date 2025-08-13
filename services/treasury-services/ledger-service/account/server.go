package account

import (
	"context"
	"log"

	pb "example.com/go-mono-repo/proto/ledger"
	"github.com/codenotary/immudb/pkg/client"
)

// Server implements the AccountService gRPC interface
// Spec: docs/specs/003-account-management.md
type Server struct {
	pb.UnimplementedAccountServiceServer
	manager ManagerInterface
}

// NewServer creates a new account server
func NewServer(db client.ImmuClient) *Server {
	repo := NewAccountRepository(db)
	validator := NewValidator()
	manager := NewManager(repo, validator)
	
	return &Server{
		manager: manager,
	}
}

// CreateAccount creates a new account
// Spec: docs/specs/003-account-management.md#story-1-create-account
func (s *Server) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	log.Printf("Creating account: name=%s, external_id=%s, type=%s", req.Name, req.ExternalId, req.AccountType)
	
	account, err := s.manager.CreateAccount(ctx, req)
	if err != nil {
		log.Printf("Failed to create account: %v", err)
		return nil, err
	}
	
	log.Printf("Account created successfully: id=%s", account.Id)
	return &pb.CreateAccountResponse{
		Account: account,
	}, nil
}

// GetAccount retrieves account by ID
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func (s *Server) GetAccount(ctx context.Context, req *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	log.Printf("Getting account: id=%s", req.AccountId)
	
	account, err := s.manager.GetAccount(ctx, req.AccountId)
	if err != nil {
		log.Printf("Failed to get account: %v", err)
		return nil, err
	}
	
	return &pb.GetAccountResponse{
		Account: account,
	}, nil
}

// GetAccountByExternalId retrieves account by external ID
// Spec: docs/specs/003-account-management.md#story-5-retrieve-account-by-external-id
func (s *Server) GetAccountByExternalId(ctx context.Context, req *pb.GetAccountByExternalIdRequest) (*pb.GetAccountByExternalIdResponse, error) {
	log.Printf("Getting account by external ID: %s", req.ExternalId)
	
	account, err := s.manager.GetAccountByExternalID(ctx, req.ExternalId)
	if err != nil {
		log.Printf("Failed to get account by external ID: %v", err)
		return nil, err
	}
	
	return &pb.GetAccountByExternalIdResponse{
		Account: account,
	}, nil
}

// UpdateAccount updates account fields
// Spec: docs/specs/003-account-management.md#story-3-update-account
func (s *Server) UpdateAccount(ctx context.Context, req *pb.UpdateAccountRequest) (*pb.UpdateAccountResponse, error) {
	log.Printf("Updating account: id=%s", req.AccountId)
	
	account, err := s.manager.UpdateAccount(ctx, req.AccountId, req.Account, req.UpdateMask)
	if err != nil {
		log.Printf("Failed to update account: %v", err)
		return nil, err
	}
	
	log.Printf("Account updated successfully: id=%s", account.Id)
	return &pb.UpdateAccountResponse{
		Account: account,
	}, nil
}

// ListAccounts lists accounts with filtering
// Spec: docs/specs/003-account-management.md#story-4-list-accounts
func (s *Server) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	log.Printf("Listing accounts: page_size=%d, filters=%+v", req.PageSize, req)
	
	resp, err := s.manager.ListAccounts(ctx, req)
	if err != nil {
		log.Printf("Failed to list accounts: %v", err)
		return nil, err
	}
	
	log.Printf("Listed %d accounts, total=%d", len(resp.Accounts), resp.TotalCount)
	return resp, nil
}