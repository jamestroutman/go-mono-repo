package account

import (
	"context"
	"testing"

	pb "example.com/go-mono-repo/proto/ledger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// MockManager is a mock implementation of Manager
type MockManager struct {
	mock.Mock
}

func (m *MockManager) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.Account, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.Account), args.Error(1)
}

func (m *MockManager) GetAccount(ctx context.Context, accountID string) (*pb.Account, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.Account), args.Error(1)
}

func (m *MockManager) GetAccountByExternalID(ctx context.Context, externalID string) (*pb.Account, error) {
	args := m.Called(ctx, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.Account), args.Error(1)
}

func (m *MockManager) UpdateAccount(ctx context.Context, accountID string, account *pb.Account, updateMask *fieldmaskpb.FieldMask) (*pb.Account, error) {
	args := m.Called(ctx, accountID, account, updateMask)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.Account), args.Error(1)
}

func (m *MockManager) ListAccounts(ctx context.Context, req *pb.ListAccountsRequest) (*pb.ListAccountsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pb.ListAccountsResponse), args.Error(1)
}

// TestServerCreateAccount tests the gRPC CreateAccount endpoint
// Spec: docs/specs/003-account-management.md#story-1-create-account
func TestServerCreateAccount(t *testing.T) {
	ctx := context.Background()
	mockManager := new(MockManager)
	server := &Server{
		manager: mockManager,
	}

	t.Run("successful creation", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "Test Account",
			ExternalId:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		expectedAccount := &pb.Account{
			Id:           "generated-uuid",
			Name:         req.Name,
			ExternalId:   req.ExternalId,
			CurrencyCode: req.CurrencyCode,
			AccountType:  req.AccountType,
			Version:      1,
		}

		mockManager.On("CreateAccount", ctx, req).
			Return(expectedAccount, nil).Once()

		resp, err := server.CreateAccount(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedAccount, resp.Account)
		mockManager.AssertExpectations(t)
	})

	t.Run("validation error", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "", // Invalid
			ExternalId:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		mockManager.On("CreateAccount", ctx, req).
			Return(nil, status.Error(codes.InvalidArgument, "field name is required")).Once()

		resp, err := server.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		mockManager.AssertExpectations(t)
	})

	t.Run("duplicate external ID", func(t *testing.T) {
		req := &pb.CreateAccountRequest{
			Name:         "Test Account",
			ExternalId:   "EXT-DUP",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		mockManager.On("CreateAccount", ctx, req).
			Return(nil, status.Error(codes.AlreadyExists, "account with external_id EXT-DUP already exists")).Once()

		resp, err := server.CreateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, st.Code())
		mockManager.AssertExpectations(t)
	})
}

// TestServerGetAccount tests the gRPC GetAccount endpoint
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func TestServerGetAccount(t *testing.T) {
	ctx := context.Background()
	mockManager := new(MockManager)
	server := &Server{
		manager: mockManager,
	}

	t.Run("successful retrieval", func(t *testing.T) {
		req := &pb.GetAccountRequest{
			AccountId: "test-uuid",
		}

		expectedAccount := &pb.Account{
			Id:           req.AccountId,
			Name:         "Test Account",
			ExternalId:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			Version:      1,
		}

		mockManager.On("GetAccount", ctx, req.AccountId).
			Return(expectedAccount, nil).Once()

		resp, err := server.GetAccount(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedAccount, resp.Account)
		mockManager.AssertExpectations(t)
	})

	t.Run("account not found", func(t *testing.T) {
		req := &pb.GetAccountRequest{
			AccountId: "non-existent",
		}

		mockManager.On("GetAccount", ctx, req.AccountId).
			Return(nil, status.Error(codes.NotFound, "account non-existent not found")).Once()

		resp, err := server.GetAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockManager.AssertExpectations(t)
	})
}

// TestServerUpdateAccount tests the gRPC UpdateAccount endpoint
// Spec: docs/specs/003-account-management.md#story-3-update-account
func TestServerUpdateAccount(t *testing.T) {
	ctx := context.Background()
	mockManager := new(MockManager)
	server := &Server{
		manager: mockManager,
	}

	t.Run("successful update", func(t *testing.T) {
		req := &pb.UpdateAccountRequest{
			AccountId: "test-uuid",
			Account: &pb.Account{
				Name: "Updated Name",
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"name"},
			},
		}

		expectedAccount := &pb.Account{
			Id:           req.AccountId,
			Name:         "Updated Name",
			ExternalId:   "EXT-001",
			CurrencyCode: "USD",
			AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			Version:      2,
		}

		mockManager.On("UpdateAccount", ctx, req.AccountId, req.Account, req.UpdateMask).
			Return(expectedAccount, nil).Once()

		resp, err := server.UpdateAccount(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedAccount, resp.Account)
		assert.Equal(t, int64(2), resp.Account.Version)
		mockManager.AssertExpectations(t)
	})

	t.Run("optimistic locking conflict", func(t *testing.T) {
		req := &pb.UpdateAccountRequest{
			AccountId: "test-uuid",
			Account: &pb.Account{
				Name: "Updated Name",
			},
		}

		mockManager.On("UpdateAccount", ctx, req.AccountId, req.Account, (*fieldmaskpb.FieldMask)(nil)).
			Return(nil, status.Error(codes.Aborted, "account was modified, retry update")).Once()

		resp, err := server.UpdateAccount(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Aborted, st.Code())
		mockManager.AssertExpectations(t)
	})
}

// TestServerListAccounts tests the gRPC ListAccounts endpoint
// Spec: docs/specs/003-account-management.md#story-4-list-accounts
func TestServerListAccounts(t *testing.T) {
	ctx := context.Background()
	mockManager := new(MockManager)
	server := &Server{
		manager: mockManager,
	}

	t.Run("successful list", func(t *testing.T) {
		req := &pb.ListAccountsRequest{
			PageSize:    10,
			AccountType: pb.AccountType_ACCOUNT_TYPE_ASSET,
		}

		expectedResp := &pb.ListAccountsResponse{
			Accounts: []*pb.Account{
				{
					Id:           "uuid-1",
					Name:         "Account 1",
					ExternalId:   "EXT-001",
					CurrencyCode: "USD",
					AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
				},
				{
					Id:           "uuid-2",
					Name:         "Account 2",
					ExternalId:   "EXT-002",
					CurrencyCode: "USD",
					AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
				},
			},
			NextPageToken: "next-token",
			TotalCount:    2,
		}

		mockManager.On("ListAccounts", ctx, req).
			Return(expectedResp, nil).Once()

		resp, err := server.ListAccounts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedResp, resp)
		assert.Len(t, resp.Accounts, 2)
		mockManager.AssertExpectations(t)
	})

	t.Run("empty results", func(t *testing.T) {
		req := &pb.ListAccountsRequest{
			PageSize: 10,
		}

		expectedResp := &pb.ListAccountsResponse{
			Accounts:      []*pb.Account{},
			NextPageToken: "",
			TotalCount:    0,
		}

		mockManager.On("ListAccounts", ctx, req).
			Return(expectedResp, nil).Once()

		resp, err := server.ListAccounts(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Accounts, 0)
		mockManager.AssertExpectations(t)
	})
}

// TestServerGetAccountByExternalId tests the gRPC GetAccountByExternalId endpoint
// Spec: docs/specs/003-account-management.md#story-5-retrieve-account-by-external-id
func TestServerGetAccountByExternalId(t *testing.T) {
	ctx := context.Background()
	mockManager := new(MockManager)
	server := &Server{
		manager: mockManager,
	}

	t.Run("successful retrieval", func(t *testing.T) {
		req := &pb.GetAccountByExternalIdRequest{
			ExternalId: "EXT-001",
		}

		expectedAccount := &pb.Account{
			Id:              "test-uuid",
			Name:            "Test Account",
			ExternalId:      req.ExternalId,
			ExternalGroupId: "GROUP-001",
			CurrencyCode:    "USD",
			AccountType:     pb.AccountType_ACCOUNT_TYPE_ASSET,
			Version:         1,
		}

		mockManager.On("GetAccountByExternalID", ctx, req.ExternalId).
			Return(expectedAccount, nil).Once()

		resp, err := server.GetAccountByExternalId(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedAccount, resp.Account)
		assert.Equal(t, "GROUP-001", resp.Account.ExternalGroupId)
		mockManager.AssertExpectations(t)
	})

	t.Run("external ID not found", func(t *testing.T) {
		req := &pb.GetAccountByExternalIdRequest{
			ExternalId: "EXT-NOTFOUND",
		}

		mockManager.On("GetAccountByExternalID", ctx, req.ExternalId).
			Return(nil, status.Error(codes.NotFound, "account with external_id EXT-NOTFOUND not found")).Once()

		resp, err := server.GetAccountByExternalId(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		mockManager.AssertExpectations(t)
	})
}