package account

import (
	"context"

	pb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RepositoryInterface defines the interface for account repository operations
type RepositoryInterface interface {
	CreateAccount(ctx context.Context, account *AccountRow) error
	GetAccountByID(ctx context.Context, accountID string) (*AccountRow, error)
	GetAccountByExternalID(ctx context.Context, externalID string) (*AccountRow, error)
	UpdateAccount(ctx context.Context, accountID string, updates map[string]interface{}, currentVersion int64) (*AccountRow, error)
	ListAccounts(ctx context.Context, filters ListAccountFilters) ([]*AccountRow, string, int32, error)
}

// ManagerInterface defines the interface for account manager operations
type ManagerInterface interface {
	CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.Account, error)
	GetAccount(ctx context.Context, accountID string) (*pb.Account, error)
	GetAccountByExternalID(ctx context.Context, externalID string) (*pb.Account, error)
	UpdateAccount(ctx context.Context, accountID string, account *pb.Account, updateMask *fieldmaskpb.FieldMask) (*pb.Account, error)
	ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error)
}