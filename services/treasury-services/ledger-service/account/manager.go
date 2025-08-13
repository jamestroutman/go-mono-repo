package account

import (
	"context"
	"database/sql"
	"log"
	"strings"

	pb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Manager handles account business logic
// Spec: docs/specs/003-account-management.md
type Manager struct {
	repo      RepositoryInterface
	validator *Validator
}

// NewManager creates a new account manager
func NewManager(repo RepositoryInterface, validator *Validator) *Manager {
	return &Manager{
		repo:      repo,
		validator: validator,
	}
}

// CreateAccount creates a new account
// Spec: docs/specs/003-account-management.md#story-1-create-account
func (m *Manager) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.Account, error) {
	// Validate input
	if err := m.validator.ValidateCreateAccount(req); err != nil {
		return nil, err
	}

	// Validate currency code
	// Note: Currency validation via Treasury Service would be implemented here
	// For now, we'll do basic validation
	if err := m.validator.ValidateCurrencyCode(ctx, req.CurrencyCode); err != nil {
		return nil, err
	}

	// Convert proto to repository model
	accountRow := &AccountRow{
		Name:         req.Name,
		ExternalID:   req.ExternalId,
		CurrencyCode: req.CurrencyCode,
		AccountType:  accountTypeProtoToString(req.AccountType),
	}

	// Handle optional external group ID
	if req.ExternalGroupId != "" {
		accountRow.ExternalGroupID = sql.NullString{
			String: req.ExternalGroupId,
			Valid:  true,
		}
	}

	// Create account in database
	if err := m.repo.CreateAccount(ctx, accountRow); err != nil {
		log.Printf("Failed to create account: %v", err)
		return nil, err
	}

	// Convert back to proto
	return accountRowToProto(accountRow), nil
}

// GetAccount retrieves account by ID
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func (m *Manager) GetAccount(ctx context.Context, accountID string) (*pb.Account, error) {
	// Validate ID format
	if accountID == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}

	// Query database
	accountRow, err := m.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	return accountRowToProto(accountRow), nil
}

// GetAccountByExternalID retrieves account by external ID
// Spec: docs/specs/003-account-management.md#story-5-retrieve-account-by-external-id
func (m *Manager) GetAccountByExternalID(ctx context.Context, externalID string) (*pb.Account, error) {
	// Validate external ID
	if externalID == "" {
		return nil, status.Error(codes.InvalidArgument, "external_id is required")
	}

	// Query database
	accountRow, err := m.repo.GetAccountByExternalID(ctx, externalID)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	return accountRowToProto(accountRow), nil
}

// UpdateAccount updates account fields
// Spec: docs/specs/003-account-management.md#story-3-update-account
func (m *Manager) UpdateAccount(ctx context.Context, accountID string, account *pb.Account, updateMask *fieldmaskpb.FieldMask) (*pb.Account, error) {
	// Validate account ID
	if accountID == "" {
		return nil, status.Error(codes.InvalidArgument, "account_id is required")
	}

	// Get existing account to check version
	existingAccount, err := m.repo.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}

	// Build update map based on field mask
	updates := make(map[string]interface{})
	
	if updateMask == nil || len(updateMask.Paths) == 0 {
		// If no mask provided, update all mutable fields
		if account.Name != "" {
			updates["name"] = account.Name
		}
		if account.ExternalGroupId != "" {
			updates["external_group_id"] = account.ExternalGroupId
		}
		if account.AccountType != pb.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
			updates["account_type"] = accountTypeProtoToString(account.AccountType)
		}
	} else {
		// Apply field mask
		for _, path := range updateMask.Paths {
			switch path {
			case "name":
				if account.Name == "" {
					return nil, status.Error(codes.InvalidArgument, "name cannot be empty")
				}
				updates["name"] = account.Name
			case "external_group_id":
				// Allow empty to clear the field
				if account.ExternalGroupId == "" {
					updates["external_group_id"] = sql.NullString{Valid: false}
				} else {
					updates["external_group_id"] = account.ExternalGroupId
				}
			case "account_type":
				if account.AccountType == pb.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
					return nil, status.Error(codes.InvalidArgument, "invalid account type")
				}
				updates["account_type"] = accountTypeProtoToString(account.AccountType)
			case "external_id", "currency_code":
				return nil, status.Errorf(codes.InvalidArgument, "field %s is immutable", path)
			}
		}
	}

	// Validate updates
	for field, value := range updates {
		if field == "name" {
			name := value.(string)
			if err := m.validator.ValidateName(name); err != nil {
				return nil, err
			}
		}
		if field == "account_type" {
			accountType := value.(string)
			if err := m.validator.ValidateAccountType(accountType); err != nil {
				return nil, err
			}
		}
	}

	// Update account with optimistic locking
	updatedAccount, err := m.repo.UpdateAccount(ctx, accountID, updates, existingAccount.Version)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	return accountRowToProto(updatedAccount), nil
}

// ListAccounts lists accounts with filtering
// Spec: docs/specs/003-account-management.md#story-4-list-accounts
func (m *Manager) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	// Build filters
	filters := ListAccountFilters{
		PageSize:        req.PageSize,
		PageToken:       req.PageToken,
		AccountType:     req.AccountType.String(),
		CurrencyCode:    req.CurrencyCode,
		ExternalGroupID: req.ExternalGroupId,
		NameSearch:      req.NameSearch,
	}

	// Validate page size
	if filters.PageSize < 0 {
		return nil, status.Error(codes.InvalidArgument, "page_size cannot be negative")
	}
	if filters.PageSize == 0 {
		filters.PageSize = 50 // Default
	}
	if filters.PageSize > 200 {
		filters.PageSize = 200 // Max
	}

	// Query database
	accountRows, nextPageToken, totalCount, err := m.repo.ListAccounts(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Convert to proto
	accounts := make([]*pb.Account, len(accountRows))
	for i, row := range accountRows {
		accounts[i] = accountRowToProto(row)
	}

	return &pb.ListAccountsResponse{
		Accounts:       accounts,
		NextPageToken:  nextPageToken,
		TotalCount:     totalCount,
	}, nil
}

// Helper functions

// accountRowToProto converts database row to proto message
func accountRowToProto(row *AccountRow) *pb.Account {
	account := &pb.Account{
		Id:           row.ID,
		Name:         row.Name,
		ExternalId:   row.ExternalID,
		CurrencyCode: row.CurrencyCode,
		AccountType:  stringToAccountTypeProto(row.AccountType),
		CreatedAt:    timestamppb.New(row.CreatedAt),
		UpdatedAt:    timestamppb.New(row.UpdatedAt),
		Version:      row.Version,
	}

	// Handle optional external group ID
	if row.ExternalGroupID.Valid {
		account.ExternalGroupId = row.ExternalGroupID.String
	}

	return account
}

// accountTypeProtoToString converts proto enum to string
func accountTypeProtoToString(accountType pb.AccountType) string {
	switch accountType {
	case pb.AccountType_ACCOUNT_TYPE_ASSET:
		return "ASSET"
	case pb.AccountType_ACCOUNT_TYPE_LIABILITY:
		return "LIABILITY"
	case pb.AccountType_ACCOUNT_TYPE_REVENUE:
		return "REVENUE"
	case pb.AccountType_ACCOUNT_TYPE_EXPENSE:
		return "EXPENSE"
	case pb.AccountType_ACCOUNT_TYPE_EQUITY:
		return "EQUITY"
	default:
		return ""
	}
}

// stringToAccountTypeProto converts string to proto enum
func stringToAccountTypeProto(accountType string) pb.AccountType {
	switch strings.ToUpper(accountType) {
	case "ASSET":
		return pb.AccountType_ACCOUNT_TYPE_ASSET
	case "LIABILITY":
		return pb.AccountType_ACCOUNT_TYPE_LIABILITY
	case "REVENUE":
		return pb.AccountType_ACCOUNT_TYPE_REVENUE
	case "EXPENSE":
		return pb.AccountType_ACCOUNT_TYPE_EXPENSE
	case "EQUITY":
		return pb.AccountType_ACCOUNT_TYPE_EQUITY
	default:
		return pb.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	}
}