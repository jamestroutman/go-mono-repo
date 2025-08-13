package main

import (
	"database/sql"
	"testing"

	pb "example.com/go-mono-repo/proto/treasury"
)

// TestValidateRoutingNumber tests the routing number validation
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func TestValidateRoutingNumber(t *testing.T) {
	tests := []struct {
		name    string
		routing string
		wantErr bool
	}{
		{
			name:    "valid JPMorgan routing number",
			routing: "021000021",
			wantErr: false,
		},
		{
			name:    "valid Bank of America routing number",
			routing: "026009593",
			wantErr: false,
		},
		{
			name:    "valid Wells Fargo routing number",
			routing: "121000248",
			wantErr: false,
		},
		{
			name:    "invalid - too short",
			routing: "12345678",
			wantErr: true,
		},
		{
			name:    "invalid - too long",
			routing: "1234567890",
			wantErr: true,
		},
		{
			name:    "invalid - contains letters",
			routing: "12345678A",
			wantErr: true,
		},
		{
			name:    "invalid - bad check digit",
			routing: "123456789",
			wantErr: true,
		},
		{
			name:    "invalid - all zeros",
			routing: "000000000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoutingNumber(tt.routing)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoutingNumber(%s) error = %v, wantErr %v", tt.routing, err, tt.wantErr)
			}
		})
	}
}

// TestValidateSwiftCode tests the SWIFT code validation
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func TestValidateSwiftCode(t *testing.T) {
	tests := []struct {
		name    string
		swift   string
		wantErr bool
	}{
		{
			name:    "valid 8-character SWIFT code",
			swift:   "CHASUS33",
			wantErr: false,
		},
		{
			name:    "valid 11-character SWIFT code",
			swift:   "CHASUS33XXX",
			wantErr: false,
		},
		{
			name:    "valid SWIFT with numbers",
			swift:   "BOFAUS3N",
			wantErr: false,
		},
		{
			name:    "valid German bank SWIFT",
			swift:   "DEUTDEFF",
			wantErr: false,
		},
		{
			name:    "invalid - too short",
			swift:   "CHASE",
			wantErr: true,
		},
		{
			name:    "invalid - too long",
			swift:   "CHASUS33XXXX",
			wantErr: true,
		},
		{
			name:    "invalid - lowercase letters",
			swift:   "chasus33",
			wantErr: true,
		},
		{
			name:    "invalid - special characters",
			swift:   "CHAS-US33",
			wantErr: true,
		},
		{
			name:    "invalid - wrong format",
			swift:   "12345678",
			wantErr: true,
		},
		{
			name:    "invalid - 9 characters",
			swift:   "CHASUS33X",
			wantErr: true,
		},
		{
			name:    "invalid - 10 characters",
			swift:   "CHASUS33XX",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSwiftCode(tt.swift)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSwiftCode(%s) error = %v, wantErr %v", tt.swift, err, tt.wantErr)
			}
		})
	}
}

// TestInstitutionTypeConversion tests the enum conversion functions
// Spec: docs/specs/004-financial-institutions.md
func TestInstitutionTypeConversion(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		wantEnum pb.InstitutionType
	}{
		{
			name:     "bank",
			typeStr:  "bank",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_BANK,
		},
		{
			name:     "credit union",
			typeStr:  "credit_union",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_CREDIT_UNION,
		},
		{
			name:     "investment bank",
			typeStr:  "investment_bank",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_INVESTMENT_BANK,
		},
		{
			name:     "central bank",
			typeStr:  "central_bank",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_CENTRAL_BANK,
		},
		{
			name:     "savings bank",
			typeStr:  "savings_bank",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_SAVINGS_BANK,
		},
		{
			name:     "online bank",
			typeStr:  "online_bank",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_ONLINE_BANK,
		},
		{
			name:     "other",
			typeStr:  "other",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_OTHER,
		},
		{
			name:     "unknown",
			typeStr:  "unknown",
			wantEnum: pb.InstitutionType_INSTITUTION_TYPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string to enum
			gotEnum := stringToInstitutionType(tt.typeStr)
			if gotEnum != tt.wantEnum {
				t.Errorf("stringToInstitutionType(%s) = %v, want %v", tt.typeStr, gotEnum, tt.wantEnum)
			}

			// Test enum to string (skip unspecified)
			if tt.wantEnum != pb.InstitutionType_INSTITUTION_TYPE_UNSPECIFIED {
				gotStr := institutionTypeToString(tt.wantEnum)
				if gotStr != tt.typeStr {
					t.Errorf("institutionTypeToString(%v) = %v, want %v", tt.wantEnum, gotStr, tt.typeStr)
				}
			}
		})
	}
}

// TestInstitutionStatusConversion tests the status enum conversion functions
// Spec: docs/specs/004-financial-institutions.md
func TestInstitutionStatusConversion(t *testing.T) {
	tests := []struct {
		name       string
		statusStr  string
		wantEnum   pb.InstitutionStatus
	}{
		{
			name:      "active",
			statusStr: "active",
			wantEnum:  pb.InstitutionStatus_INSTITUTION_STATUS_ACTIVE,
		},
		{
			name:      "inactive",
			statusStr: "inactive",
			wantEnum:  pb.InstitutionStatus_INSTITUTION_STATUS_INACTIVE,
		},
		{
			name:      "suspended",
			statusStr: "suspended",
			wantEnum:  pb.InstitutionStatus_INSTITUTION_STATUS_SUSPENDED,
		},
		{
			name:      "deleted",
			statusStr: "deleted",
			wantEnum:  pb.InstitutionStatus_INSTITUTION_STATUS_DELETED,
		},
		{
			name:      "unknown",
			statusStr: "unknown",
			wantEnum:  pb.InstitutionStatus_INSTITUTION_STATUS_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string to enum
			gotEnum := stringToInstitutionStatus(tt.statusStr)
			if gotEnum != tt.wantEnum {
				t.Errorf("stringToInstitutionStatus(%s) = %v, want %v", tt.statusStr, gotEnum, tt.wantEnum)
			}

			// Test enum to string (skip unspecified)
			if tt.wantEnum != pb.InstitutionStatus_INSTITUTION_STATUS_UNSPECIFIED {
				gotStr := institutionStatusToString(tt.wantEnum)
				if gotStr != tt.statusStr {
					t.Errorf("institutionStatusToString(%v) = %v, want %v", tt.wantEnum, gotStr, tt.statusStr)
				}
			}
		})
	}
}

// TestAddressFieldFunction tests the address field helper function
// Spec: docs/specs/004-financial-institutions.md#story-3-update-institution-information
func TestAddressFieldFunction(t *testing.T) {
	tests := []struct {
		name    string
		address *pb.Address
		field   string
		want    interface{}
	}{
		{
			name: "street_address_1 with value",
			address: &pb.Address{
				StreetAddress_1: "270 Park Avenue",
			},
			field: "street_address_1",
			want:  nullString("270 Park Avenue"),
		},
		{
			name: "city with value",
			address: &pb.Address{
				City: "New York",
			},
			field: "city",
			want:  nullString("New York"),
		},
		{
			name: "state_province with value",
			address: &pb.Address{
				StateProvince: "NY",
			},
			field: "state_province",
			want:  nullString("NY"),
		},
		{
			name: "postal_code with value",
			address: &pb.Address{
				PostalCode: "10017",
			},
			field: "postal_code",
			want:  nullString("10017"),
		},
		{
			name:    "nil address",
			address: nil,
			field:   "street_address_1",
			want:    nil,
		},
		{
			name:    "empty address",
			address: &pb.Address{},
			field:   "street_address_1",
			want:    nullString(""),
		},
		{
			name: "unknown field",
			address: &pb.Address{
				StreetAddress_1: "test",
			},
			field: "unknown_field",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addressField(tt.address, tt.field)
			
			// Compare nullString values properly
			if tt.want != nil {
				wantNullStr, ok := tt.want.(sql.NullString)
				if ok {
					gotNullStr, ok := got.(sql.NullString)
					if !ok {
						t.Errorf("addressField() returned type %T, want sql.NullString", got)
						return
					}
					if gotNullStr.Valid != wantNullStr.Valid || gotNullStr.String != wantNullStr.String {
						t.Errorf("addressField() = %+v, want %+v", gotNullStr, wantNullStr)
					}
					return
				}
			}
			
			if got != tt.want {
				t.Errorf("addressField() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestContactFieldFunction tests the contact field helper function
// Spec: docs/specs/004-financial-institutions.md#story-3-update-institution-information
func TestContactFieldFunction(t *testing.T) {
	tests := []struct {
		name    string
		contact *pb.ContactInfo
		field   string
		want    interface{}
	}{
		{
			name: "phone_number with value",
			contact: &pb.ContactInfo{
				PhoneNumber: "+1-212-270-6000",
			},
			field: "phone_number",
			want:  nullString("+1-212-270-6000"),
		},
		{
			name: "email_address with value",
			contact: &pb.ContactInfo{
				EmailAddress: "contact@jpmorgan.com",
			},
			field: "email_address",
			want:  nullString("contact@jpmorgan.com"),
		},
		{
			name: "website_url with value",
			contact: &pb.ContactInfo{
				WebsiteUrl: "https://www.jpmorganchase.com",
			},
			field: "website_url",
			want:  nullString("https://www.jpmorganchase.com"),
		},
		{
			name: "fax_number with value",
			contact: &pb.ContactInfo{
				FaxNumber: "+1-212-270-7000",
			},
			field: "fax_number",
			want:  nullString("+1-212-270-7000"),
		},
		{
			name:    "nil contact",
			contact: nil,
			field:   "phone_number",
			want:    nil,
		},
		{
			name:    "empty contact",
			contact: &pb.ContactInfo{},
			field:   "phone_number",
			want:    nullString(""),
		},
		{
			name: "unknown field",
			contact: &pb.ContactInfo{
				PhoneNumber: "test",
			},
			field: "unknown_field",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contactField(tt.contact, tt.field)
			
			// Compare nullString values properly
			if tt.want != nil {
				wantNullStr, ok := tt.want.(sql.NullString)
				if ok {
					gotNullStr, ok := got.(sql.NullString)
					if !ok {
						t.Errorf("contactField() returned type %T, want sql.NullString", got)
						return
					}
					if gotNullStr.Valid != wantNullStr.Valid || gotNullStr.String != wantNullStr.String {
						t.Errorf("contactField() = %+v, want %+v", gotNullStr, wantNullStr)
					}
					return
				}
			}
			
			if got != tt.want {
				t.Errorf("contactField() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNullStringHelper tests the nullString helper function
func TestNullStringHelper(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  sql.NullString
	}{
		{
			name:  "empty string",
			input: "",
			want:  sql.NullString{Valid: false},
		},
		{
			name:  "non-empty string",
			input: "test value",
			want:  sql.NullString{String: "test value", Valid: true},
		},
		{
			name:  "space string",
			input: " ",
			want:  sql.NullString{String: " ", Valid: true},
		},
		{
			name:  "address example",
			input: "270 Park Avenue",
			want:  sql.NullString{String: "270 Park Avenue", Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullString(tt.input)
			if got.Valid != tt.want.Valid || got.String != tt.want.String {
				t.Errorf("nullString(%s) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}