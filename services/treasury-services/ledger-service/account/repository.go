package account

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/codenotary/immudb/pkg/client"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AccountRepository handles database operations for accounts
// Spec: docs/specs/003-account-management.md
type AccountRepository struct {
	db client.ImmuClient
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db client.ImmuClient) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

// AccountRow represents a database row for an account
type AccountRow struct {
	ID              string
	Name            string
	ExternalID      string
	ExternalGroupID sql.NullString
	CurrencyCode    string
	AccountType     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Version         int64
}

// CreateAccount creates a new account in the database
// Spec: docs/specs/003-account-management.md#story-1-create-account
func (r *AccountRepository) CreateAccount(ctx context.Context, account *AccountRow) error {
	// Generate UUID if not provided
	if account.ID == "" {
		account.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now
	account.Version = 1

	// Prepare SQL statement
	query := `
		INSERT INTO accounts (
			id, name, external_id, external_group_id, 
			currency_code, account_type, created_at, updated_at, version
		) VALUES (
			@id, @name, @external_id, @external_group_id,
			@currency_code, @account_type, @created_at, @updated_at, @version
		)`

	params := map[string]interface{}{
		"id":            account.ID,
		"name":          account.Name,
		"external_id":   account.ExternalID,
		"currency_code": account.CurrencyCode,
		"account_type":  account.AccountType,
		"created_at":    account.CreatedAt,
		"updated_at":    account.UpdatedAt,
		"version":       account.Version,
	}
	
	// Handle nullable external_group_id
	if account.ExternalGroupID.Valid {
		params["external_group_id"] = account.ExternalGroupID.String
	} else {
		params["external_group_id"] = nil
	}

	_, err := r.db.SQLExec(ctx, query, params)
	if err != nil {
		// Check for unique constraint violation
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return status.Errorf(codes.AlreadyExists, "account with external_id %s already exists", account.ExternalID)
		}
		return status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return nil
}

// GetAccountByID retrieves an account by its system ID
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func (r *AccountRepository) GetAccountByID(ctx context.Context, accountID string) (*AccountRow, error) {
	query := `
		SELECT 
			id, name, external_id, external_group_id,
			currency_code, account_type, created_at, updated_at, version
		FROM accounts
		WHERE id = @id`

	params := map[string]interface{}{
		"id": accountID,
	}

	result, err := r.db.SQLQuery(ctx, query, params, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query account: %v", err)
	}

	// Check if any rows were returned
	if len(result.Rows) == 0 {
		return nil, status.Errorf(codes.NotFound, "account %s not found", accountID)
	}

	// Parse the first row
	row := result.Rows[0]
	account := &AccountRow{
		ID:           string(row.Values[0].GetS()),
		Name:         string(row.Values[1].GetS()),
		ExternalID:   string(row.Values[2].GetS()),
		CurrencyCode: string(row.Values[4].GetS()),
		AccountType:  string(row.Values[5].GetS()),
		CreatedAt:    time.UnixMicro(row.Values[6].GetTs()),
		UpdatedAt:    time.UnixMicro(row.Values[7].GetTs()),
		Version:      row.Values[8].GetN(),
	}
	
	// Handle optional external_group_id (index 3)
	if row.Values[3] != nil && len(row.Values[3].GetS()) > 0 {
		account.ExternalGroupID = sql.NullString{
			String: string(row.Values[3].GetS()),
			Valid:  true,
		}
	}

	return account, nil
}

// GetAccountByExternalID retrieves an account by its external ID
// Spec: docs/specs/003-account-management.md#story-5-retrieve-account-by-external-id
func (r *AccountRepository) GetAccountByExternalID(ctx context.Context, externalID string) (*AccountRow, error) {
	query := `
		SELECT 
			id, name, external_id, external_group_id,
			currency_code, account_type, created_at, updated_at, version
		FROM accounts
		WHERE external_id = @external_id`

	params := map[string]interface{}{
		"external_id": externalID,
	}

	result, err := r.db.SQLQuery(ctx, query, params, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query account: %v", err)
	}

	// Check if any rows were returned
	if len(result.Rows) == 0 {
		return nil, status.Errorf(codes.NotFound, "account with external_id %s not found", externalID)
	}

	// Parse the first row
	row := result.Rows[0]
	account := &AccountRow{
		ID:           string(row.Values[0].GetS()),
		Name:         string(row.Values[1].GetS()),
		ExternalID:   string(row.Values[2].GetS()),
		CurrencyCode: string(row.Values[4].GetS()),
		AccountType:  string(row.Values[5].GetS()),
		CreatedAt:    time.UnixMicro(row.Values[6].GetTs()),
		UpdatedAt:    time.UnixMicro(row.Values[7].GetTs()),
		Version:      row.Values[8].GetN(),
	}
	
	// Handle optional external_group_id (index 3)
	if row.Values[3] != nil && len(row.Values[3].GetS()) > 0 {
		account.ExternalGroupID = sql.NullString{
			String: string(row.Values[3].GetS()),
			Valid:  true,
		}
	}

	return account, nil
}

// UpdateAccount updates an existing account with optimistic locking
// Spec: docs/specs/003-account-management.md#story-3-update-account
func (r *AccountRepository) UpdateAccount(ctx context.Context, accountID string, updates map[string]interface{}, currentVersion int64) (*AccountRow, error) {
	// Build dynamic UPDATE query based on provided fields
	setClauses := []string{}
	params := map[string]interface{}{
		"id":          accountID,
		"version":     currentVersion,
		"new_version": currentVersion + 1,
		"updated_at":  time.Now(),
	}

	// Always update version and updated_at
	setClauses = append(setClauses, "version = @new_version", "updated_at = @updated_at")

	// Add dynamic fields
	for field, value := range updates {
		switch field {
		case "name":
			setClauses = append(setClauses, "name = @name")
			params["name"] = value
		case "external_group_id":
			setClauses = append(setClauses, "external_group_id = @external_group_id")
			params["external_group_id"] = value
		case "account_type":
			setClauses = append(setClauses, "account_type = @account_type")
			params["account_type"] = value
		}
	}

	query := fmt.Sprintf(`
		UPDATE accounts 
		SET %s
		WHERE id = @id AND version = @version`,
		strings.Join(setClauses, ", "))

	_, err := r.db.SQLExec(ctx, query, params)
	if err != nil {
		// Check if it's a version mismatch
		if strings.Contains(err.Error(), "version") {
			// Check if account exists
			_, err := r.GetAccountByID(ctx, accountID)
			if err != nil {
				return nil, err // Will return NotFound if account doesn't exist
			}
			// Account exists but version mismatch
			return nil, status.Errorf(codes.Aborted, "account was modified, retry update")
		}
		return nil, status.Errorf(codes.Internal, "failed to update account: %v", err)
	}

	// Note: ImmuDB doesn't return affected rows count
	// We assume success if no error, and rely on version check in WHERE clause
	// Check if update succeeded by fetching the account
	updatedAccount, err := r.GetAccountByID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	
	// Verify version was incremented
	if updatedAccount.Version == currentVersion {
		// Version wasn't updated, meaning WHERE clause didn't match
		return nil, status.Errorf(codes.Aborted, "account was modified, retry update")
	}
	
	return updatedAccount, nil
}

// ListAccounts lists accounts with filtering and pagination
// Spec: docs/specs/003-account-management.md#story-4-list-accounts
func (r *AccountRepository) ListAccounts(ctx context.Context, filters ListAccountFilters) ([]*AccountRow, string, int32, error) {
	// Build WHERE clause
	whereClauses := []string{}
	params := map[string]interface{}{}

	if filters.AccountType != "" && filters.AccountType != "ACCOUNT_TYPE_UNSPECIFIED" {
		whereClauses = append(whereClauses, "account_type = @account_type")
		params["account_type"] = strings.TrimPrefix(filters.AccountType, "ACCOUNT_TYPE_")
	}

	if filters.CurrencyCode != "" {
		whereClauses = append(whereClauses, "currency_code = @currency_code")
		params["currency_code"] = filters.CurrencyCode
	}

	if filters.ExternalGroupID != "" {
		whereClauses = append(whereClauses, "external_group_id = @external_group_id")
		params["external_group_id"] = filters.ExternalGroupID
	}

	if filters.NameSearch != "" {
		whereClauses = append(whereClauses, "LOWER(name) LIKE @name_search")
		params["name_search"] = "%" + strings.ToLower(filters.NameSearch) + "%"
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total matching accounts
	countQuery := fmt.Sprintf("SELECT COUNT(*) as total FROM accounts %s", whereClause)
	countResult, err := r.db.SQLQuery(ctx, countQuery, params, false)
	if err != nil {
		return nil, "", 0, status.Errorf(codes.Internal, "failed to count accounts: %v", err)
	}

	totalCount := int32(0)
	if len(countResult.Rows) > 0 {
		totalCount = int32(countResult.Rows[0].Values[0].GetN())
	}

	// Build main query with pagination
	limit := filters.PageSize
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	offset := int32(0)
	if filters.PageToken != "" {
		// Simple offset-based pagination for now
		// In production, use cursor-based pagination
		fmt.Sscanf(filters.PageToken, "%d", &offset)
	}

	query := fmt.Sprintf(`
		SELECT 
			id, name, external_id, external_group_id,
			currency_code, account_type, created_at, updated_at, version
		FROM accounts
		%s
		ORDER BY created_at DESC, id
		LIMIT %d OFFSET %d`,
		whereClause, limit, offset)

	result, err := r.db.SQLQuery(ctx, query, params, false)
	if err != nil {
		return nil, "", 0, status.Errorf(codes.Internal, "failed to list accounts: %v", err)
	}

	// Parse results
	accounts := make([]*AccountRow, 0, len(result.Rows))
	for _, row := range result.Rows {
		account := &AccountRow{
			ID:           string(row.Values[0].GetS()),
			Name:         string(row.Values[1].GetS()),
			ExternalID:   string(row.Values[2].GetS()),
			CurrencyCode: string(row.Values[4].GetS()),
			AccountType:  string(row.Values[5].GetS()),
			CreatedAt:    time.UnixMicro(row.Values[6].GetTs()),
			UpdatedAt:    time.UnixMicro(row.Values[7].GetTs()),
			Version:      row.Values[8].GetN(),
		}
		
		// Handle optional external_group_id (index 3)
		if row.Values[3] != nil && len(row.Values[3].GetS()) > 0 {
			account.ExternalGroupID = sql.NullString{
				String: string(row.Values[3].GetS()),
				Valid:  true,
			}
		}
		
		accounts = append(accounts, account)
	}

	// Calculate next page token
	nextPageToken := ""
	if offset+limit < totalCount {
		nextPageToken = fmt.Sprintf("%d", offset+limit)
	}

	return accounts, nextPageToken, totalCount, nil
}

// ListAccountFilters contains filters for listing accounts
type ListAccountFilters struct {
	PageSize        int32
	PageToken       string
	AccountType     string
	CurrencyCode    string
	ExternalGroupID string
	NameSearch      string
}