package account

import (
	"context"
	"database/sql"
	"testing"

	pb "example.com/go-mono-repo/proto/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// MockRepository is a mock implementation of AccountRepository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateAccount(ctx context.Context, account *AccountRow) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *MockRepository) GetAccountByID(ctx context.Context, accountID string) (*AccountRow, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountRow), args.Error(1)
}

func (m *MockRepository) GetAccountByExternalID(ctx context.Context, externalID string) (*AccountRow, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountRow), args.Error(1)
}

func (m *MockRepository) UpdateAccount(ctx context.Context, accountID string, updates map[string]interface{}, currentVersion int64) (*AccountRow, error) {
	args := m.Called(ctx, accountID, updates, currentVersion)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AccountRow), args.Error(1)
}

func (m *MockRepository) ListAccounts(ctx context.Context, filters ListAccountFilters) ([]*AccountRow, string, int32, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.String(1), args.Get(2).(int32), args.Error(3)
	}
	return args.Get(0).([]*AccountRow), args.String(1), args.Get(2).(int32), args.Error(3)
}

// TestCreateAccount tests the CreateAccount method
// Spec: docs/specs/003-account-management.md#story-1-create-account
func TestCreateAccount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	validator := NewValidator()
	manager := &Manager{
		repo:      mockRepo,
		validator: validator,
	}

	t.Run("successful creation", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "Test Account",
			ExternalId:   "EXT-TEST-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		expectedRow := &AccountRow{
			ID:           "generated-uuid",
			Name:         req.Name,
			ExternalID:   req.ExternalId,
			CurrencyCode: req.CurrencyCode,
			AccountType:  "ASSET",
			Version:      1,
		}

		mockRepo.On("CreateAccount", ctx, mock.AnythingOfType("*account.AccountRow")).
			Run(func(args mock.Arguments) {
				row := args.Get(1).(*AccountRow)
				row.ID = expectedRow.ID
			}).
			Return(nil).Once()

		result, err := manager.CreateAccount(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, req.Name, result.Name)
		assert.Equal(t, req.ExternalId, result.ExternalId)
		mockRepo.AssertExpectations(t)
	})

	t.Run("duplicate external ID", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "Test Account",
			ExternalId:   "EXT-DUP-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		mockRepo.On("CreateAccount", ctx, mock.AnythingOfType("*account.AccountRow")).
			Return(status.Error(codes.AlreadyExists, "account with external_id EXT-DUP-001 already exists")).Once()

		result, err := manager.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, st.Code())
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid input", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "", // Empty name
			ExternalId:   "EXT-TEST-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		result, err := manager.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		// Should not call repository if validation fails
		mockRepo.AssertNotCalled(t, "CreateAccount")
	})
}

// TestGetAccount tests the GetAccount method
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func TestGetAccount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	validator := NewValidator()
	manager := &Manager{
		repo:      mockRepo,
		validator: validator,
	}

	t.Run("successful retrieval", func(t *testing.T) {
		accountID := "test-uuid"
		expectedRow := &AccountRow{
			ID:           accountID,
			Name:         "Test Account",
			ExternalID:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      1,
		}

		mockRepo.On("GetAccountByID", ctx, accountID).
			Return(expectedRow, nil).Once()

		result, err := manager.GetAccount(ctx, accountID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, accountID, result.Id)
		assert.Equal(t, expectedRow.Name, result.Name)
		mockRepo.AssertExpectations(t)
	})

	t.Run("account not found", func(t *testing.T) {
		accountID := "non-existent"

		mockRepo.On("GetAccountByID", ctx, accountID).
			Return(nil, status.Error(codes.NotFound, "account non-existent not found")).Once()

		result, err := manager.GetAccount(ctx, accountID)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty account ID", func(t *testing.T) {
		result, err := manager.GetAccount(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockRepo.AssertNotCalled(t, "GetAccountByID")
	})
}

// TestUpdateAccount tests the UpdateAccount method
// Spec: docs/specs/003-account-management.md#story-3-update-account
func TestUpdateAccount(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	validator := NewValidator()
	manager := &Manager{
		repo:      mockRepo,
		validator: validator,
	}

	t.Run("successful update with field mask", func(t *testing.T) {
		accountID := "test-uuid"
		existingRow := &AccountRow{
			ID:           accountID,
			Name:         "Old Name",
			ExternalID:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      1,
		}

		updatedRow := &AccountRow{
			ID:           accountID,
			Name:         "New Name",
			ExternalID:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      2,
		}

		mockRepo.On("GetAccountByID", ctx, accountID).
			Return(existingRow, nil).Once()

		expectedUpdates := map[string]interface{}{
			"name": "New Name",
		}

		mockRepo.On("UpdateAccount", ctx, accountID, expectedUpdates, int64(1)).
			Return(updatedRow, nil).Once()

		account := &pb.Account{
			Name: "New Name",
		}
		fieldMask := &fieldmaskpb.FieldMask{
			Paths: []string{"name"},
		}

		result, err := manager.UpdateAccount(ctx, accountID, account, fieldMask)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Name", result.Name)
		assert.Equal(t, int64(2), result.Version)
		mockRepo.AssertExpectations(t)
	})

	t.Run("optimistic locking conflict", func(t *testing.T) {
		accountID := "test-uuid"
		existingRow := &AccountRow{
			ID:           accountID,
			Name:         "Old Name",
			ExternalID:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      1,
		}

		mockRepo.On("GetAccountByID", ctx, accountID).
			Return(existingRow, nil).Once()

		mockRepo.On("UpdateAccount", ctx, accountID, mock.Anything, int64(1)).
			Return(nil, status.Error(codes.Aborted, "account was modified, retry update")).Once()

		account := &pb.Account{
			Name: "New Name",
		}

		result, err := manager.UpdateAccount(ctx, accountID, account, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Aborted, st.Code())
		mockRepo.AssertExpectations(t)
	})

	t.Run("attempt to update immutable field", func(t *testing.T) {
		accountID := "test-uuid"
		existingRow := &AccountRow{
			ID:           accountID,
			Name:         "Test Account",
			ExternalID:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      1,
		}

		mockRepo.On("GetAccountByID", ctx, accountID).
			Return(existingRow, nil).Once()

		account := &pb.Account{
			ExternalId: "EXT-002", // Trying to change external ID
		}
		fieldMask := &fieldmaskpb.FieldMask{
			Paths: []string{"external_id"},
		}

		result, err := manager.UpdateAccount(ctx, accountID, account, fieldMask)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "immutable")
		// Should not attempt update if field is immutable
		mockRepo.AssertNotCalled(t, "UpdateAccount")
	})
}

// TestListAccounts tests the ListAccounts method
// Spec: docs/specs/003-account-management.md#story-4-list-accounts
func TestListAccounts(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	validator := NewValidator()
	manager := &Manager{
		repo:      mockRepo,
		validator: validator,
	}

	t.Run("successful list with filters", func(t *testing.T) {
		req := &pb.ListAccountsRequest{
			PageSize:    10,
			AccountType: pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		expectedFilters := ListAccountFilters{
			PageSize:    10,
			AccountType: "ACCOUNT_TYPE_ASSET",
		}

		rows := []*AccountRow{
			{
				ID:           "uuid-1",
				Name:         "Account 1",
				ExternalID:   "EXT-001",
				CurrencyCode: "USD",
				AccountType:  "ASSET",
				Version:      1,
			},
			{
				ID:           "uuid-2",
				Name:         "Account 2",
				ExternalID:   "EXT-002",
				CurrencyCode: "USD",
				AccountType:  "ASSET",
				Version:      1,
			},
		}

		mockRepo.On("ListAccounts", ctx, expectedFilters).
			Return(rows, "next-token", int32(2), nil).Once()

		result, err := manager.ListAccounts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Accounts, 2)
		assert.Equal(t, "next-token", result.NextPageToken)
		assert.Equal(t, int32(2), result.TotalCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		req := &pb.ListAccountsRequest{
			PageSize: 10,
		}

		expectedFilters := ListAccountFilters{
			PageSize:    10,
			AccountType: "ACCOUNT_TYPE_UNSPECIFIED", // Default value
		}

		mockRepo.On("ListAccounts", ctx, expectedFilters).
			Return([]*AccountRow{}, "", int32(0), nil).Once()

		result, err := manager.ListAccounts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Accounts, 0)
		assert.Equal(t, "", result.NextPageToken)
		assert.Equal(t, int32(0), result.TotalCount)
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid page size", func(t *testing.T) {
		req := &pb.ListAccountsRequest{
			PageSize: -1,
		}

		result, err := manager.ListAccounts(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockRepo.AssertNotCalled(t, "ListAccounts")
	})
}

// TestGetAccountByExternalID tests the GetAccountByExternalID method
// Spec: docs/specs/003-account-management.md#story-5-retrieve-account-by-external-id
func TestGetAccountByExternalID(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)
	validator := NewValidator()
	manager := &Manager{
		repo:      mockRepo,
		validator: validator,
	}

	t.Run("successful retrieval", func(t *testing.T) {
		externalID := "EXT-TEST-001"
		expectedRow := &AccountRow{
			ID:           "test-uuid",
			Name:         "Test Account",
			ExternalID:   externalID,
			CurrencyCode: "USD",
			AccountType:  "ASSET",
			Version:      1,
			ExternalGroupID: sql.NullString{
				String: "GROUP-001",
				Valid:  true,
			},
		}

		mockRepo.On("GetAccountByExternalID", ctx, externalID).
			Return(expectedRow, nil).Once()

		result, err := manager.GetAccountByExternalID(ctx, externalID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, externalID, result.ExternalId)
		assert.Equal(t, "GROUP-001", result.ExternalGroupId)
		mockRepo.AssertExpectations(t)
	})

	t.Run("empty external ID", func(t *testing.T) {
		result, err := manager.GetAccountByExternalID(ctx, "")

		assert.Error(t, err)
		assert.Nil(t, result)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockRepo.AssertNotCalled(t, "GetAccountByExternalID")
	})
}