package account

import (
	"context"
	"testing"

	pb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestValidateCreateAccount tests account creation validation
// Spec: docs/specs/003-account-management.md#story-1-create-account
func TestValidateCreateAccount(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		req     *pb.CreateAccountRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "valid account",
			req: &pb.CreateAccountRequest{
				Name:         "Cash on Hand",
				ExternalId:   "EXT-CASH-001",
				CurrencyCode: "USD",
				AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &pb.CreateAccountRequest{
				ExternalId:   "EXT-001",
				CurrencyCode: "USD",
				AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "missing external ID",
			req: &pb.CreateAccountRequest{
				Name:         "Test Account",
				CurrencyCode: "USD",
				AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "invalid account type",
			req: &pb.CreateAccountRequest{
				Name:         "Test Account",
				ExternalId:   "EXT-001",
				CurrencyCode: "USD",
				AccountType:  pb.AccountType_ACCOUNT_TYPE_UNSPECIFIED,
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "name too long",
			req: &pb.CreateAccountRequest{
				Name:         string(make([]byte, 256)),
				ExternalId:   "EXT-001",
				CurrencyCode: "USD",
				AccountType:  pb.AccountType_ACCOUNT_TYPE_ASSET,
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "with external group ID",
			req: &pb.CreateAccountRequest{
				Name:            "Test Account",
				ExternalId:      "EXT-001",
				ExternalGroupId: "GROUP-001",
				CurrencyCode:    "USD",
				AccountType:     pb.AccountType_ACCOUNT_TYPE_LIABILITY,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCreateAccount(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateAccount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errCode != codes.OK {
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("expected gRPC status error, got %v", err)
					return
				}
				if st.Code() != tt.errCode {
					t.Errorf("expected error code %v, got %v", tt.errCode, st.Code())
				}
			}
		})
	}
}

// TestValidateCurrencyCode tests currency code validation
// Spec: docs/specs/003-account-management.md
func TestValidateCurrencyCode(t *testing.T) {
	v := NewValidator()
	ctx := context.Background()

	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"valid USD", "USD", false},
		{"valid EUR", "EUR", false},
		{"valid GBP", "GBP", false},
		{"valid JPY", "JPY", false},
		{"empty code", "", true},
		{"lowercase", "usd", true},
		{"too short", "US", true},
		{"too long", "USDD", true},
		{"invalid code", "XXX", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCurrencyCode(ctx, tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCurrencyCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateAccountType tests account type validation
// Spec: docs/specs/003-account-management.md#data-models
func TestValidateAccountType(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name        string
		accountType string
		wantErr     bool
	}{
		{"valid ASSET", "ASSET", false},
		{"valid LIABILITY", "LIABILITY", false},
		{"valid REVENUE", "REVENUE", false},
		{"valid EXPENSE", "EXPENSE", false},
		{"valid EQUITY", "EQUITY", false},
		{"lowercase asset", "asset", false}, // Should be converted to uppercase
		{"invalid type", "INVALID", true},
		{"empty type", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAccountType(tt.accountType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccountType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateName tests name validation
func TestValidateName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Cash on Hand", false},
		{"with numbers", "Account 123", false},
		{"with punctuation", "Smith & Co.", false},
		{"with parentheses", "Cash (USD)", false},
		{"empty name", "", true},
		{"too long", string(make([]byte, 256)), true},
		{"special chars", "Account@#$", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateExternalID tests external ID validation
func TestValidateExternalID(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid ID", "EXT-001", false},
		{"with underscore", "EXT_001", false},
		{"alphanumeric", "ABC123", false},
		{"empty ID", "", true},
		{"with spaces", "EXT 001", true},
		{"special chars", "EXT@001", true},
		{"too long", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateExternalID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExternalID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}