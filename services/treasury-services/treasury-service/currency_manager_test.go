package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	pb "example.com/go-mono-repo/proto/treasury"
)

// TestNewCurrencyManager tests the creation of a new CurrencyManager
func TestNewCurrencyManager(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	manager := NewCurrencyManager(db)
	assert.NotNil(t, manager)
	assert.Equal(t, db, manager.db)
}

// TestCreateCurrency tests the CreateCurrency method
func TestCreateCurrency(t *testing.T) {
	tests := []struct {
		name      string
		request   *pb.CreateCurrencyRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errCode   codes.Code
	}{
		{
			name: "successful creation",
			request: &pb.CreateCurrencyRequest{
				Code:         "TST",
				NumericCode:  "999",
				Name:         "Test Currency",
				MinorUnits:   2,
				Symbol:       "T",
				CountryCodes: []string{"TS"},
				IsCrypto:     false,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Check existence
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("TST").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

				// Insert currency
				mock.ExpectQuery("INSERT INTO treasury.currencies").
					WithArgs(
						sqlmock.AnyArg(), // UUID
						"TST",
						sqlmock.AnyArg(), // numeric_code
						"Test Currency",
						int32(2),
						sqlmock.AnyArg(), // symbol
						sqlmock.AnyArg(), // country_codes
						false,
						"active",
						true,
						sqlmock.AnyArg(), // created_at
						sqlmock.AnyArg(), // updated_at
						"system",
						1,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(uuid.New(), time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name: "invalid ISO code format",
			request: &pb.CreateCurrencyRequest{
				Code: "TS", // Only 2 characters
				Name: "Test Currency",
			},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   true,
			errCode:   codes.InvalidArgument,
		},
		{
			name: "invalid numeric code format",
			request: &pb.CreateCurrencyRequest{
				Code:        "TST",
				NumericCode: "99", // Only 2 digits
				Name:        "Test Currency",
			},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   true,
			errCode:   codes.InvalidArgument,
		},
		{
			name: "currency already exists",
			request: &pb.CreateCurrencyRequest{
				Code: "TST",
				Name: "Test Currency",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("TST").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			wantErr: true,
			errCode: codes.AlreadyExists,
		},
		{
			name: "default minor units for non-crypto",
			request: &pb.CreateCurrencyRequest{
				Code:     "TST",
				Name:     "Test Currency",
				IsCrypto: false,
				// MinorUnits not set, should default to 2
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("TST").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

				mock.ExpectQuery("INSERT INTO treasury.currencies").
					WithArgs(
						sqlmock.AnyArg(),
						"TST",
						sqlmock.AnyArg(),
						"Test Currency",
						int32(2), // Default minor units
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						false,
						"active",
						true,
						sqlmock.AnyArg(),
						sqlmock.AnyArg(),
						"system",
						1,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(uuid.New(), time.Now(), time.Now()))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.CreateCurrency(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode != codes.OK {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.errCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.request.Code, result.Code)
				assert.Equal(t, tt.request.Name, result.Name)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestGetCurrency tests the GetCurrency method
func TestGetCurrency(t *testing.T) {
	fixedTime := time.Now()
	fixedUUID := uuid.New()

	tests := []struct {
		name      string
		request   *pb.GetCurrencyRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errCode   codes.Code
		validate  func(*testing.T, *pb.Currency)
	}{
		{
			name: "get by code",
			request: &pb.GetCurrencyRequest{
				Identifier: &pb.GetCurrencyRequest_Code{Code: "USD"},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				)
				mock.ExpectQuery("SELECT .* FROM treasury.currencies WHERE code = ").
					WithArgs("USD").
					WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, currency *pb.Currency) {
				assert.Equal(t, "USD", currency.Code)
				assert.Equal(t, "840", currency.NumericCode)
				assert.Equal(t, "United States Dollar", currency.Name)
				assert.Equal(t, int32(2), currency.MinorUnits)
				assert.Equal(t, "$", currency.Symbol)
				assert.True(t, currency.IsActive)
			},
		},
		{
			name: "get by numeric code",
			request: &pb.GetCurrencyRequest{
				Identifier: &pb.GetCurrencyRequest_NumericCode{NumericCode: "840"},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				)
				mock.ExpectQuery("SELECT .* FROM treasury.currencies WHERE numeric_code = ").
					WithArgs("840").
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "get by id",
			request: &pb.GetCurrencyRequest{
				Identifier: &pb.GetCurrencyRequest_Id{Id: fixedUUID.String()},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				)
				mock.ExpectQuery("SELECT .* FROM treasury.currencies WHERE id = ").
					WithArgs(fixedUUID.String()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "currency not found",
			request: &pb.GetCurrencyRequest{
				Identifier: &pb.GetCurrencyRequest_Code{Code: "XXX"},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .* FROM treasury.currencies WHERE code = ").
					WithArgs("XXX").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errCode: codes.NotFound,
		},
		{
			name:    "no identifier provided",
			request: &pb.GetCurrencyRequest{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.GetCurrency(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode != codes.OK {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.errCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestUpdateCurrency tests the UpdateCurrency method
func TestUpdateCurrency(t *testing.T) {
	fixedTime := time.Now()
	fixedUUID := uuid.New()

	tests := []struct {
		name      string
		request   *pb.UpdateCurrencyRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errCode   codes.Code
	}{
		{
			name: "update name and symbol",
			request: &pb.UpdateCurrencyRequest{
				Code: "USD",
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"name", "symbol"},
				},
				Name:    "US Dollar",
				Symbol:  "US$",
				Version: 1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "US Dollar", 2,
					"US$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", "system", 2,
				)
				mock.ExpectQuery("UPDATE treasury.currencies").
					WithArgs(
						"US Dollar",
						sqlmock.AnyArg(), // symbol
						sqlmock.AnyArg(), // updated_at
						"system",
						"USD",
						int32(1),
					).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "no fields to update",
			request: &pb.UpdateCurrencyRequest{
				Code:       "USD",
				UpdateMask: &fieldmaskpb.FieldMask{},
				Version:    1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {},
			wantErr:   true,
			errCode:   codes.InvalidArgument,
		},
		{
			name: "version conflict",
			request: &pb.UpdateCurrencyRequest{
				Code: "USD",
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"name"},
				},
				Name:    "Updated Name",
				Version: 1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("UPDATE treasury.currencies").
					WithArgs(
						"Updated Name",
						sqlmock.AnyArg(),
						"system",
						"USD",
						int32(1),
					).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errCode: codes.Aborted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.UpdateCurrency(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode != codes.OK {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.errCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestDeactivateCurrency tests the DeactivateCurrency method
func TestDeactivateCurrency(t *testing.T) {
	fixedTime := time.Now()
	fixedUUID := uuid.New()

	tests := []struct {
		name      string
		request   *pb.DeactivateCurrencyRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errCode   codes.Code
	}{
		{
			name: "successful deactivation",
			request: &pb.DeactivateCurrencyRequest{
				Code:      "USD",
				Status:    pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE,
				UpdatedBy: "admin",
				Version:   1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), false,
					false, "inactive", fixedTime, fixedTime,
					fixedTime, fixedTime, "system", "admin", 2,
				)
				mock.ExpectQuery("UPDATE treasury.currencies").
					WithArgs("inactive", "admin", "USD", int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "version conflict",
			request: &pb.DeactivateCurrencyRequest{
				Code:      "USD",
				Status:    pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE,
				UpdatedBy: "admin",
				Version:   1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("UPDATE treasury.currencies").
					WithArgs("inactive", "admin", "USD", int32(1)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errCode: codes.Aborted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.DeactivateCurrency(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode != codes.OK {
					st, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.errCode, st.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.False(t, result.IsActive)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestListCurrencies tests the ListCurrencies method
func TestListCurrencies(t *testing.T) {
	fixedTime := time.Now()
	fixedUUID := uuid.New()

	tests := []struct {
		name      string
		request   *pb.ListCurrenciesRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		validate  func(*testing.T, *pb.ListCurrenciesResponse)
	}{
		{
			name: "list all currencies",
			request: &pb.ListCurrenciesRequest{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Query for currencies
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				).AddRow(
					uuid.New().String(), "EUR", "978", "Euro", 2,
					"€", "before", pq.Array([]string{"EU"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				)
				mock.ExpectQuery("SELECT .* FROM treasury.currencies").
					WillReturnRows(rows)

				// Query for count
				countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery("SELECT COUNT").
					WillReturnRows(countRows)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *pb.ListCurrenciesResponse) {
				assert.Len(t, resp.Currencies, 2)
				assert.Equal(t, int32(2), resp.TotalCount)
			},
		},
		{
			name: "list with filters",
			request: &pb.ListCurrenciesRequest{
				Status:   pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE,
				IsActive: true,
				PageSize: 10,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "code", "numeric_code", "name", "minor_units",
					"symbol", "symbol_position", "country_codes", "is_active",
					"is_crypto", "status", "activated_at", "deactivated_at",
					"created_at", "updated_at", "created_by", "updated_by", "version",
				}).AddRow(
					fixedUUID.String(), "USD", "840", "United States Dollar", 2,
					"$", "before", pq.Array([]string{"US"}), true,
					false, "active", fixedTime, nil,
					fixedTime, fixedTime, "system", nil, 1,
				)
				mock.ExpectQuery("SELECT .* FROM treasury.currencies").
					WithArgs("active", true, int32(10)).
					WillReturnRows(rows)

				countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery("SELECT COUNT").
					WithArgs("active").
					WillReturnRows(countRows)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *pb.ListCurrenciesResponse) {
				assert.Len(t, resp.Currencies, 1)
				assert.Equal(t, int32(1), resp.TotalCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.ListCurrencies(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestBulkCreateCurrencies tests the BulkCreateCurrencies method
func TestBulkCreateCurrencies(t *testing.T) {
	tests := []struct {
		name      string
		request   *pb.BulkCreateCurrenciesRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		validate  func(*testing.T, *pb.BulkCreateCurrenciesResponse)
	}{
		{
			name: "create multiple currencies",
			request: &pb.BulkCreateCurrenciesRequest{
				Currencies: []*pb.CreateCurrencyRequest{
					{
						Code: "TST",
						Name: "Test Currency 1",
					},
					{
						Code: "TST2",
						Name: "Test Currency 2",
					},
				},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				
				// First currency - doesn't exist
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("TST").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				mock.ExpectExec("INSERT INTO treasury.currencies").
					WithArgs(
						sqlmock.AnyArg(), // UUID
						"TST",
						sqlmock.AnyArg(), // numeric_code
						"Test Currency 1",
						int32(0),
						sqlmock.AnyArg(), // symbol
						sqlmock.AnyArg(), // country_codes
						false,
						"active",
						true,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				
				// Second currency - doesn't exist
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("TST2").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
				mock.ExpectExec("INSERT INTO treasury.currencies").
					WithArgs(
						sqlmock.AnyArg(), // UUID
						"TST2",
						sqlmock.AnyArg(), // numeric_code
						"Test Currency 2",
						int32(0),
						sqlmock.AnyArg(), // symbol
						sqlmock.AnyArg(), // country_codes
						false,
						"active",
						true,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, resp *pb.BulkCreateCurrenciesResponse) {
				assert.Equal(t, int32(2), resp.CreatedCount)
				assert.Equal(t, int32(0), resp.UpdatedCount)
				assert.Equal(t, int32(0), resp.SkippedCount)
				assert.Empty(t, resp.Errors)
			},
		},
		{
			name: "skip duplicates",
			request: &pb.BulkCreateCurrenciesRequest{
				Currencies: []*pb.CreateCurrencyRequest{
					{
						Code: "USD",
						Name: "US Dollar",
					},
				},
				SkipDuplicates: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				
				// Currency exists
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("USD").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, resp *pb.BulkCreateCurrenciesResponse) {
				assert.Equal(t, int32(0), resp.CreatedCount)
				assert.Equal(t, int32(0), resp.UpdatedCount)
				assert.Equal(t, int32(1), resp.SkippedCount)
				assert.Empty(t, resp.Errors)
			},
		},
		{
			name: "update existing",
			request: &pb.BulkCreateCurrenciesRequest{
				Currencies: []*pb.CreateCurrencyRequest{
					{
						Code:       "USD",
						Name:       "Updated US Dollar",
						MinorUnits: 2,
					},
				},
				UpdateExisting: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				
				// Currency exists
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs("USD").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
				
				// Update existing
				mock.ExpectExec("UPDATE treasury.currencies").
					WithArgs(
						"Updated US Dollar",
						int32(2),
						sqlmock.AnyArg(), // symbol
						sqlmock.AnyArg(), // country_codes
						"USD",
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
				
				mock.ExpectCommit()
			},
			wantErr: false,
			validate: func(t *testing.T, resp *pb.BulkCreateCurrenciesResponse) {
				assert.Equal(t, int32(0), resp.CreatedCount)
				assert.Equal(t, int32(1), resp.UpdatedCount)
				assert.Equal(t, int32(0), resp.SkippedCount)
				assert.Empty(t, resp.Errors)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			manager := NewCurrencyManager(db)
			result, err := manager.BulkCreateCurrencies(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestHelperFunctions tests the helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("nullString", func(t *testing.T) {
		// Test empty string
		ns := nullString("")
		assert.False(t, ns.Valid)

		// Test non-empty string
		ns = nullString("test")
		assert.True(t, ns.Valid)
		assert.Equal(t, "test", ns.String)
	})

	t.Run("mapCurrencyStatus", func(t *testing.T) {
		tests := []struct {
			input    string
			expected pb.CurrencyStatus
		}{
			{"active", pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE},
			{"inactive", pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE},
			{"deprecated", pb.CurrencyStatus_CURRENCY_STATUS_DEPRECATED},
			{"deleted", pb.CurrencyStatus_CURRENCY_STATUS_DELETED},
			{"unknown", pb.CurrencyStatus_CURRENCY_STATUS_UNSPECIFIED},
		}

		for _, tt := range tests {
			result := mapCurrencyStatus(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("mapStatusToString", func(t *testing.T) {
		tests := []struct {
			input    pb.CurrencyStatus
			expected string
		}{
			{pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE, "active"},
			{pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE, "inactive"},
			{pb.CurrencyStatus_CURRENCY_STATUS_DEPRECATED, "deprecated"},
			{pb.CurrencyStatus_CURRENCY_STATUS_DELETED, "deleted"},
			{pb.CurrencyStatus_CURRENCY_STATUS_UNSPECIFIED, "active"},
		}

		for _, tt := range tests {
			result := mapStatusToString(tt.input)
			assert.Equal(t, tt.expected, result)
		}
	})
}

// TestValidationRegexes tests the validation regular expressions
func TestValidationRegexes(t *testing.T) {
	t.Run("ISO code regex", func(t *testing.T) {
		validCodes := []string{"USD", "EUR", "GBP", "JPY"}
		invalidCodes := []string{"US", "USDD", "usd", "123", "US1"}

		for _, code := range validCodes {
			assert.True(t, isoCodeRegex.MatchString(code), "Expected %s to be valid", code)
		}

		for _, code := range invalidCodes {
			assert.False(t, isoCodeRegex.MatchString(code), "Expected %s to be invalid", code)
		}
	})

	t.Run("Numeric code regex", func(t *testing.T) {
		validCodes := []string{"840", "978", "000", "999"}
		invalidCodes := []string{"84", "8400", "ABC", "12A"}

		for _, code := range validCodes {
			assert.True(t, numericCodeRegex.MatchString(code), "Expected %s to be valid", code)
		}

		for _, code := range invalidCodes {
			assert.False(t, numericCodeRegex.MatchString(code), "Expected %s to be invalid", code)
		}
	})
}