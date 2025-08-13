package account

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	pb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator handles input validation for account operations
// Spec: docs/specs/003-account-management.md
type Validator struct {
	// Currency cache for validation
	// In production, this would connect to Treasury Service
	currencyCache map[string]bool
	cacheMutex    sync.RWMutex
	cacheExpiry   time.Time
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{
		currencyCache: initDefaultCurrencies(),
		cacheExpiry:   time.Now().Add(5 * time.Minute),
	}
}

// ValidateCreateAccount validates account creation request
// Spec: docs/specs/003-account-management.md#story-1-create-account
func (v *Validator) ValidateCreateAccount(req *pb.CreateAccountRequest) error {
	// Validate name
	if err := v.ValidateName(req.Name); err != nil {
		return err
	}

	// Validate external ID
	if err := v.ValidateExternalID(req.ExternalId); err != nil {
		return err
	}

	// Validate external group ID (optional)
	if req.ExternalGroupId != "" {
		if err := v.ValidateExternalGroupID(req.ExternalGroupId); err != nil {
			return err
		}
	}

	// Validate account type
	if err := v.ValidateAccountTypeProto(req.AccountType); err != nil {
		return err
	}

	// Currency validation is done separately to allow async Treasury Service call

	return nil
}

// ValidateName validates account name
func (v *Validator) ValidateName(name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "field name is required")
	}
	
	if len(name) > 255 {
		return status.Error(codes.InvalidArgument, "name must be 255 characters or less")
	}

	// Check for valid characters (alphanumeric, spaces, common punctuation)
	validName := regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,&()]+$`)
	if !validName.MatchString(name) {
		return status.Error(codes.InvalidArgument, "name contains invalid characters")
	}

	return nil
}

// ValidateExternalID validates external ID
func (v *Validator) ValidateExternalID(externalID string) error {
	if externalID == "" {
		return status.Error(codes.InvalidArgument, "field external_id is required")
	}

	if len(externalID) > 255 {
		return status.Error(codes.InvalidArgument, "external_id must be 255 characters or less")
	}

	// External IDs should be alphanumeric with hyphens and underscores
	validExtID := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validExtID.MatchString(externalID) {
		return status.Error(codes.InvalidArgument, "external_id contains invalid characters")
	}

	return nil
}

// ValidateExternalGroupID validates external group ID
func (v *Validator) ValidateExternalGroupID(groupID string) error {
	if len(groupID) > 255 {
		return status.Error(codes.InvalidArgument, "external_group_id must be 255 characters or less")
	}

	// External group IDs should be alphanumeric with hyphens and underscores
	validGroupID := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validGroupID.MatchString(groupID) {
		return status.Error(codes.InvalidArgument, "external_group_id contains invalid characters")
	}

	return nil
}

// ValidateCurrencyCode validates ISO 4217 currency code
// Spec: docs/specs/003-account-management.md - Currency validation
func (v *Validator) ValidateCurrencyCode(ctx context.Context, code string) error {
	if code == "" {
		return status.Error(codes.InvalidArgument, "field currency_code is required")
	}

	if len(code) != 3 {
		return status.Error(codes.InvalidArgument, "currency_code must be exactly 3 characters")
	}

	// Check if code is uppercase
	if code != strings.ToUpper(code) {
		return status.Error(codes.InvalidArgument, "currency_code must be uppercase")
	}

	// Check cache
	v.cacheMutex.RLock()
	isValid, found := v.currencyCache[code]
	needsRefresh := time.Now().After(v.cacheExpiry)
	v.cacheMutex.RUnlock()

	if found && !needsRefresh {
		if !isValid {
			return status.Errorf(codes.InvalidArgument, "invalid currency code: %s", code)
		}
		return nil
	}

	// In production, this would call Treasury Service
	// For now, check against common currencies
	if !v.isCommonCurrency(code) {
		return status.Errorf(codes.InvalidArgument, "invalid currency code: %s", code)
	}

	// Update cache
	v.cacheMutex.Lock()
	v.currencyCache[code] = true
	if needsRefresh {
		v.cacheExpiry = time.Now().Add(5 * time.Minute)
	}
	v.cacheMutex.Unlock()

	return nil
}

// ValidateAccountTypeProto validates account type proto enum
func (v *Validator) ValidateAccountTypeProto(accountType pb.AccountType) error {
	switch accountType {
	case pb.AccountType_ACCOUNT_TYPE_ASSET,
		pb.AccountType_ACCOUNT_TYPE_LIABILITY,
		pb.AccountType_ACCOUNT_TYPE_REVENUE,
		pb.AccountType_ACCOUNT_TYPE_EXPENSE,
		pb.AccountType_ACCOUNT_TYPE_EQUITY:
		return nil
	case pb.AccountType_ACCOUNT_TYPE_UNSPECIFIED:
		return status.Error(codes.InvalidArgument, "field account_type is required")
	default:
		return status.Errorf(codes.InvalidArgument, "invalid account type: %v", accountType)
	}
}

// ValidateAccountType validates account type string
func (v *Validator) ValidateAccountType(accountType string) error {
	validTypes := map[string]bool{
		"ASSET":     true,
		"LIABILITY": true,
		"REVENUE":   true,
		"EXPENSE":   true,
		"EQUITY":    true,
	}

	upperType := strings.ToUpper(accountType)
	if !validTypes[upperType] {
		return status.Errorf(codes.InvalidArgument, "invalid account type: %s", accountType)
	}

	return nil
}

// ValidateAccountID validates account ID format
func (v *Validator) ValidateAccountID(accountID string) error {
	if accountID == "" {
		return status.Error(codes.InvalidArgument, "account_id is required")
	}

	// Validate UUID format (basic check)
	uuidRegex := regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	if !uuidRegex.MatchString(strings.ToLower(accountID)) {
		return status.Error(codes.InvalidArgument, "invalid account_id format")
	}

	return nil
}

// isCommonCurrency checks if currency is a common ISO 4217 code
func (v *Validator) isCommonCurrency(code string) bool {
	// Common currencies for MVP
	commonCurrencies := map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"CHF": true, "CAD": true, "AUD": true, "NZD": true,
		"CNY": true, "INR": true, "KRW": true, "SGD": true,
		"HKD": true, "NOK": true, "SEK": true, "DKK": true,
		"PLN": true, "THB": true, "IDR": true, "HUF": true,
		"CZK": true, "ILS": true, "CLP": true, "PHP": true,
		"AED": true, "COP": true, "SAR": true, "MYR": true,
		"RON": true, "BRL": true, "MXN": true, "ZAR": true,
	}

	return commonCurrencies[code]
}

// initDefaultCurrencies initializes the currency cache with common currencies
func initDefaultCurrencies() map[string]bool {
	return map[string]bool{
		"USD": true, "EUR": true, "GBP": true, "JPY": true,
		"CHF": true, "CAD": true, "AUD": true, "NZD": true,
		"CNY": true, "INR": true, "KRW": true, "SGD": true,
		"HKD": true, "NOK": true, "SEK": true, "DKK": true,
	}
}