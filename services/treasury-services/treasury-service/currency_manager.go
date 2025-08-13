package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "example.com/go-mono-repo/proto/treasury"
)

// CurrencyManager handles currency database operations
// Spec: docs/specs/003-currency-management.md
type CurrencyManager struct {
	db *sql.DB
}

// NewCurrencyManager creates a new currency manager instance
// Spec: docs/specs/003-currency-management.md
func NewCurrencyManager(db *sql.DB) *CurrencyManager {
	return &CurrencyManager{
		db: db,
	}
}

var (
	// ISO 4217 code validation regex (3 uppercase letters)
	isoCodeRegex = regexp.MustCompile(`^[A-Z]{3}$`)
	// ISO 4217 numeric code validation regex (3 digits)
	numericCodeRegex = regexp.MustCompile(`^[0-9]{3}$`)
)

// CreateCurrency creates a new currency in the database
// Spec: docs/specs/003-currency-management.md#story-1-create-new-currency
func (cm *CurrencyManager) CreateCurrency(ctx context.Context, req *pb.CreateCurrencyRequest) (*pb.Currency, error) {
	// Validate ISO code format
	if !isoCodeRegex.MatchString(req.Code) {
		return nil, status.Error(codes.InvalidArgument, "invalid ISO code format: must be 3 uppercase letters")
	}

	// Validate numeric code if provided
	if req.NumericCode != "" && !numericCodeRegex.MatchString(req.NumericCode) {
		return nil, status.Error(codes.InvalidArgument, "invalid numeric code format: must be 3 digits")
	}

	// Set default minor units if not provided
	if req.MinorUnits == 0 && !req.IsCrypto {
		req.MinorUnits = 2
	}

	// Check for duplicate code
	var exists bool
	err := cm.db.QueryRowContext(ctx, 
		"SELECT EXISTS(SELECT 1 FROM treasury.currencies WHERE code = $1)",
		req.Code).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check currency existence: %v", err)
	}
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "currency with code %s already exists", req.Code)
	}

	// Insert currency
	id := uuid.New()
	now := time.Now()
	
	query := `
		INSERT INTO treasury.currencies (
			id, code, numeric_code, name, minor_units, symbol,
			country_codes, is_crypto, status, is_active,
			created_at, updated_at, created_by, version
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14
		) RETURNING id, created_at, updated_at`

	var createdAt, updatedAt time.Time
	err = cm.db.QueryRowContext(ctx, query,
		id, req.Code, nullString(req.NumericCode), req.Name, req.MinorUnits, nullString(req.Symbol),
		pq.Array(req.CountryCodes), req.IsCrypto, "active", true,
		now, now, "system", 1,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create currency: %v", err)
	}

	// Return created currency
	return &pb.Currency{
		Id:           id.String(),
		Code:         req.Code,
		NumericCode:  req.NumericCode,
		Name:         req.Name,
		MinorUnits:   req.MinorUnits,
		Symbol:       req.Symbol,
		CountryCodes: req.CountryCodes,
		IsActive:     true,
		IsCrypto:     req.IsCrypto,
		Status:       pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE,
		CreatedAt:    timestamppb.New(createdAt),
		UpdatedAt:    timestamppb.New(updatedAt),
		CreatedBy:    "system",
		Version:      1,
	}, nil
}

// GetCurrency retrieves a currency by code, numeric code, or ID
// Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
func (cm *CurrencyManager) GetCurrency(ctx context.Context, req *pb.GetCurrencyRequest) (*pb.Currency, error) {
	var query string
	var arg interface{}

	switch id := req.Identifier.(type) {
	case *pb.GetCurrencyRequest_Code:
		query = `SELECT id, code, numeric_code, name, minor_units, symbol, symbol_position,
				country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
				created_at, updated_at, created_by, updated_by, version
				FROM treasury.currencies WHERE code = $1`
		arg = id.Code
	case *pb.GetCurrencyRequest_NumericCode:
		query = `SELECT id, code, numeric_code, name, minor_units, symbol, symbol_position,
				country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
				created_at, updated_at, created_by, updated_by, version
				FROM treasury.currencies WHERE numeric_code = $1`
		arg = id.NumericCode
	case *pb.GetCurrencyRequest_Id:
		query = `SELECT id, code, numeric_code, name, minor_units, symbol, symbol_position,
				country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
				created_at, updated_at, created_by, updated_by, version
				FROM treasury.currencies WHERE id = $1`
		arg = id.Id
	default:
		return nil, status.Error(codes.InvalidArgument, "identifier required")
	}

	var (
		id             string
		code           string
		numericCode    sql.NullString
		name           string
		minorUnits     int32
		symbol         sql.NullString
		symbolPosition sql.NullString
		countryCodes   pq.StringArray
		isActive       bool
		isCrypto       bool
		statusStr      string
		activatedAt    sql.NullTime
		deactivatedAt  sql.NullTime
		createdAt      time.Time
		updatedAt      time.Time
		createdBy      sql.NullString
		updatedBy      sql.NullString
		version        int32
	)

	err := cm.db.QueryRowContext(ctx, query, arg).Scan(
		&id, &code, &numericCode, &name, &minorUnits, &symbol, &symbolPosition,
		&countryCodes, &isActive, &isCrypto, &statusStr, &activatedAt, &deactivatedAt,
		&createdAt, &updatedAt, &createdBy, &updatedBy, &version,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.NotFound, "currency not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get currency: %v", err)
	}

	currency := &pb.Currency{
		Id:             id,
		Code:           code,
		Name:           name,
		MinorUnits:     minorUnits,
		CountryCodes:   countryCodes,
		IsActive:       isActive,
		IsCrypto:       isCrypto,
		Status:         mapCurrencyStatus(statusStr),
		CreatedAt:      timestamppb.New(createdAt),
		UpdatedAt:      timestamppb.New(updatedAt),
		Version:        version,
	}

	if numericCode.Valid {
		currency.NumericCode = numericCode.String
	}
	if symbol.Valid {
		currency.Symbol = symbol.String
	}
	if symbolPosition.Valid {
		currency.SymbolPosition = symbolPosition.String
	}
	if activatedAt.Valid {
		currency.ActivatedAt = timestamppb.New(activatedAt.Time)
	}
	if deactivatedAt.Valid {
		currency.DeactivatedAt = timestamppb.New(deactivatedAt.Time)
	}
	if createdBy.Valid {
		currency.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		currency.UpdatedBy = updatedBy.String
	}

	return currency, nil
}

// UpdateCurrency updates currency metadata
// Spec: docs/specs/003-currency-management.md#story-3-update-currency-metadata
func (cm *CurrencyManager) UpdateCurrency(ctx context.Context, req *pb.UpdateCurrencyRequest) (*pb.Currency, error) {
	// Build dynamic update query based on update mask
	updates := []string{}
	args := []interface{}{}
	argCount := 1

	if req.UpdateMask != nil {
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "name":
				updates = append(updates, fmt.Sprintf("name = $%d", argCount))
				args = append(args, req.Name)
				argCount++
			case "minor_units":
				updates = append(updates, fmt.Sprintf("minor_units = $%d", argCount))
				args = append(args, req.MinorUnits)
				argCount++
			case "symbol":
				updates = append(updates, fmt.Sprintf("symbol = $%d", argCount))
				args = append(args, nullString(req.Symbol))
				argCount++
			case "country_codes":
				updates = append(updates, fmt.Sprintf("country_codes = $%d", argCount))
				args = append(args, pq.Array(req.CountryCodes))
				argCount++
			case "status":
				updates = append(updates, fmt.Sprintf("status = $%d", argCount))
				args = append(args, mapStatusToString(req.Status))
				argCount++
				if req.Status != pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE {
					updates = append(updates, fmt.Sprintf("is_active = $%d", argCount))
					args = append(args, false)
					argCount++
					updates = append(updates, fmt.Sprintf("deactivated_at = $%d", argCount))
					args = append(args, time.Now())
					argCount++
				}
			}
		}
	}

	if len(updates) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no fields to update")
	}

	// Add standard update fields
	updates = append(updates, 
		fmt.Sprintf("updated_at = $%d", argCount),
		fmt.Sprintf("updated_by = $%d", argCount+1),
		"version = version + 1")
	args = append(args, time.Now(), "system")

	// Add WHERE conditions
	args = append(args, req.Code, req.Version)

	query := fmt.Sprintf(`
		UPDATE treasury.currencies 
		SET %s
		WHERE code = $%d AND version = $%d
		RETURNING id, code, numeric_code, name, minor_units, symbol, symbol_position,
				country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
				created_at, updated_at, created_by, updated_by, version`,
		strings.Join(updates, ", "), argCount+2, argCount+3)

	var (
		id             string
		code           string
		numericCode    sql.NullString
		name           string
		minorUnits     int32
		symbol         sql.NullString
		symbolPosition sql.NullString
		countryCodes   pq.StringArray
		isActive       bool
		isCrypto       bool
		statusStr      string
		activatedAt    sql.NullTime
		deactivatedAt  sql.NullTime
		createdAt      time.Time
		updatedAt      time.Time
		createdBy      sql.NullString
		updatedBy      sql.NullString
		version        int32
	)

	err := cm.db.QueryRowContext(ctx, query, args...).Scan(
		&id, &code, &numericCode, &name, &minorUnits, &symbol, &symbolPosition,
		&countryCodes, &isActive, &isCrypto, &statusStr, &activatedAt, &deactivatedAt,
		&createdAt, &updatedAt, &createdBy, &updatedBy, &version,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Aborted, "version conflict or currency not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update currency: %v", err)
	}

	currency := &pb.Currency{
		Id:             id,
		Code:           code,
		Name:           name,
		MinorUnits:     minorUnits,
		CountryCodes:   countryCodes,
		IsActive:       isActive,
		IsCrypto:       isCrypto,
		Status:         mapCurrencyStatus(statusStr),
		CreatedAt:      timestamppb.New(createdAt),
		UpdatedAt:      timestamppb.New(updatedAt),
		Version:        version,
	}

	if numericCode.Valid {
		currency.NumericCode = numericCode.String
	}
	if symbol.Valid {
		currency.Symbol = symbol.String
	}
	if symbolPosition.Valid {
		currency.SymbolPosition = symbolPosition.String
	}
	if updatedBy.Valid {
		currency.UpdatedBy = updatedBy.String
	}

	return currency, nil
}

// DeactivateCurrency updates currency status to inactive/deprecated/deleted
// Spec: docs/specs/003-currency-management.md#story-4-deactivate-currency
func (cm *CurrencyManager) DeactivateCurrency(ctx context.Context, req *pb.DeactivateCurrencyRequest) (*pb.Currency, error) {
	query := `
		UPDATE treasury.currencies 
		SET status = $1, 
			is_active = false,
			deactivated_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = $2,
			version = version + 1
		WHERE code = $3 AND version = $4
		RETURNING id, code, numeric_code, name, minor_units, symbol, symbol_position,
				country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
				created_at, updated_at, created_by, updated_by, version`

	var (
		id             string
		code           string
		numericCode    sql.NullString
		name           string
		minorUnits     int32
		symbol         sql.NullString
		symbolPosition sql.NullString
		countryCodes   pq.StringArray
		isActive       bool
		isCrypto       bool
		statusStr      string
		activatedAt    sql.NullTime
		deactivatedAt  sql.NullTime
		createdAt      time.Time
		updatedAt      time.Time
		createdBy      sql.NullString
		updatedBy      sql.NullString
		version        int32
	)

	err := cm.db.QueryRowContext(ctx, query, 
		mapStatusToString(req.Status), req.UpdatedBy, req.Code, req.Version).Scan(
		&id, &code, &numericCode, &name, &minorUnits, &symbol, &symbolPosition,
		&countryCodes, &isActive, &isCrypto, &statusStr, &activatedAt, &deactivatedAt,
		&createdAt, &updatedAt, &createdBy, &updatedBy, &version,
	)

	if err == sql.ErrNoRows {
		return nil, status.Error(codes.Aborted, "version conflict or currency not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to deactivate currency: %v", err)
	}

	currency := &pb.Currency{
		Id:             id,
		Code:           code,
		Name:           name,
		MinorUnits:     minorUnits,
		CountryCodes:   countryCodes,
		IsActive:       isActive,
		IsCrypto:       isCrypto,
		Status:         mapCurrencyStatus(statusStr),
		CreatedAt:      timestamppb.New(createdAt),
		UpdatedAt:      timestamppb.New(updatedAt),
		Version:        version,
	}

	if deactivatedAt.Valid {
		currency.DeactivatedAt = timestamppb.New(deactivatedAt.Time)
	}
	if updatedBy.Valid {
		currency.UpdatedBy = updatedBy.String
	}

	return currency, nil
}

// ListCurrencies retrieves currencies with optional filters
// Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
func (cm *CurrencyManager) ListCurrencies(ctx context.Context, req *pb.ListCurrenciesRequest) (*pb.ListCurrenciesResponse, error) {
	query := `
		SELECT id, code, numeric_code, name, minor_units, symbol, symbol_position,
			   country_codes, is_active, is_crypto, status, activated_at, deactivated_at,
			   created_at, updated_at, created_by, updated_by, version
		FROM treasury.currencies
		WHERE 1=1`

	args := []interface{}{}
	argCount := 1

	// Add filters
	if req.Status != pb.CurrencyStatus_CURRENCY_STATUS_UNSPECIFIED {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, mapStatusToString(req.Status))
		argCount++
	}

	if req.IsActive {
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, req.IsActive)
		argCount++
	}

	if req.IsCrypto {
		query += fmt.Sprintf(" AND is_crypto = $%d", argCount)
		args = append(args, req.IsCrypto)
		argCount++
	}

	if req.CountryCode != "" {
		query += fmt.Sprintf(" AND $%d = ANY(country_codes)", argCount)
		args = append(args, req.CountryCode)
		argCount++
	}

	// Add ordering
	query += " ORDER BY code"

	// Add pagination
	if req.PageSize > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, req.PageSize)
		argCount++
	}

	// TODO: Implement page token logic for cursor-based pagination

	rows, err := cm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list currencies: %v", err)
	}
	defer rows.Close()

	currencies := []*pb.Currency{}
	for rows.Next() {
		var (
			id             string
			code           string
			numericCode    sql.NullString
			name           string
			minorUnits     int32
			symbol         sql.NullString
			symbolPosition sql.NullString
			countryCodes   pq.StringArray
			isActive       bool
			isCrypto       bool
			statusStr      string
			activatedAt    sql.NullTime
			deactivatedAt  sql.NullTime
			createdAt      time.Time
			updatedAt      time.Time
			createdBy      sql.NullString
			updatedBy      sql.NullString
			version        int32
		)

		err := rows.Scan(
			&id, &code, &numericCode, &name, &minorUnits, &symbol, &symbolPosition,
			&countryCodes, &isActive, &isCrypto, &statusStr, &activatedAt, &deactivatedAt,
			&createdAt, &updatedAt, &createdBy, &updatedBy, &version,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan currency: %v", err)
		}

		currency := &pb.Currency{
			Id:             id,
			Code:           code,
			Name:           name,
			MinorUnits:     minorUnits,
			CountryCodes:   countryCodes,
			IsActive:       isActive,
			IsCrypto:       isCrypto,
			Status:         mapCurrencyStatus(statusStr),
			CreatedAt:      timestamppb.New(createdAt),
			UpdatedAt:      timestamppb.New(updatedAt),
			Version:        version,
		}

		if numericCode.Valid {
			currency.NumericCode = numericCode.String
		}
		if symbol.Valid {
			currency.Symbol = symbol.String
		}

		currencies = append(currencies, currency)
	}

	// Get total count
	var totalCount int32
	countQuery := "SELECT COUNT(*) FROM treasury.currencies WHERE 1=1"
	if req.Status != pb.CurrencyStatus_CURRENCY_STATUS_UNSPECIFIED {
		countQuery += " AND status = $1"
		cm.db.QueryRowContext(ctx, countQuery, mapStatusToString(req.Status)).Scan(&totalCount)
	} else {
		cm.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	}

	return &pb.ListCurrenciesResponse{
		Currencies: currencies,
		TotalCount: totalCount,
	}, nil
}

// BulkCreateCurrencies creates multiple currencies in a single transaction
// Spec: docs/specs/003-currency-management.md#story-5-bulk-currency-operations
func (cm *CurrencyManager) BulkCreateCurrencies(ctx context.Context, req *pb.BulkCreateCurrenciesRequest) (*pb.BulkCreateCurrenciesResponse, error) {
	tx, err := cm.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var createdCount, updatedCount, skippedCount int32
	var errors []string

	for _, currency := range req.Currencies {
		// Check if currency exists
		var exists bool
		err := tx.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM treasury.currencies WHERE code = $1)",
			currency.Code).Scan(&exists)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: database error", currency.Code))
			continue
		}

		if exists {
			if req.SkipDuplicates {
				skippedCount++
				continue
			} else if req.UpdateExisting {
				// Update existing currency
				_, err = tx.ExecContext(ctx, `
					UPDATE treasury.currencies 
					SET name = $1, minor_units = $2, symbol = $3, 
						country_codes = $4, updated_at = CURRENT_TIMESTAMP
					WHERE code = $5`,
					currency.Name, currency.MinorUnits, nullString(currency.Symbol),
					pq.Array(currency.CountryCodes), currency.Code)
				if err != nil {
					errors = append(errors, fmt.Sprintf("%s: update failed", currency.Code))
				} else {
					updatedCount++
				}
			} else {
				errors = append(errors, fmt.Sprintf("%s: already exists", currency.Code))
			}
		} else {
			// Create new currency
			id := uuid.New()
			_, err = tx.ExecContext(ctx, `
				INSERT INTO treasury.currencies (
					id, code, numeric_code, name, minor_units, symbol,
					country_codes, is_crypto, status, is_active,
					created_at, updated_at, created_by, version
				) VALUES (
					$1, $2, $3, $4, $5, $6,
					$7, $8, $9, $10,
					CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'system', 1
				)`,
				id, currency.Code, nullString(currency.NumericCode), currency.Name,
				currency.MinorUnits, nullString(currency.Symbol),
				pq.Array(currency.CountryCodes), currency.IsCrypto, "active", true)
			
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: create failed: %v", currency.Code, err))
			} else {
				createdCount++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &pb.BulkCreateCurrenciesResponse{
		CreatedCount: createdCount,
		UpdatedCount: updatedCount,
		SkippedCount: skippedCount,
		Errors:       errors,
	}, nil
}

// Helper functions
func mapCurrencyStatus(status string) pb.CurrencyStatus {
	switch status {
	case "active":
		return pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE
	case "inactive":
		return pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE
	case "deprecated":
		return pb.CurrencyStatus_CURRENCY_STATUS_DEPRECATED
	case "deleted":
		return pb.CurrencyStatus_CURRENCY_STATUS_DELETED
	default:
		return pb.CurrencyStatus_CURRENCY_STATUS_UNSPECIFIED
	}
}

func mapStatusToString(status pb.CurrencyStatus) string {
	switch status {
	case pb.CurrencyStatus_CURRENCY_STATUS_ACTIVE:
		return "active"
	case pb.CurrencyStatus_CURRENCY_STATUS_INACTIVE:
		return "inactive"
	case pb.CurrencyStatus_CURRENCY_STATUS_DEPRECATED:
		return "deprecated"
	case pb.CurrencyStatus_CURRENCY_STATUS_DELETED:
		return "deleted"
	default:
		return "active"
	}
}