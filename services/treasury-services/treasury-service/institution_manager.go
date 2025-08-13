package main

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "example.com/go-mono-repo/proto/treasury"
)

// InstitutionManager handles financial institution database operations
// Spec: docs/specs/004-financial-institutions.md
type InstitutionManager struct {
	db *sql.DB
}

// NewInstitutionManager creates a new institution manager instance
// Spec: docs/specs/004-financial-institutions.md
func NewInstitutionManager(db *sql.DB) *InstitutionManager {
	return &InstitutionManager{
		db: db,
	}
}

var (
	// SWIFT code validation regex (8 or 11 characters)
	swiftCodeRegex = regexp.MustCompile(`^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`)
	// Country code validation regex (2 uppercase letters)
	countryCodeRegex = regexp.MustCompile(`^[A-Z]{2}$`)
	// Routing number validation regex (9 digits)
	routingNumberRegex = regexp.MustCompile(`^[0-9]{9}$`)
)

// ValidateRoutingNumber validates US ABA routing number
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func ValidateRoutingNumber(routing string) error {
	if !routingNumberRegex.MatchString(routing) {
		return fmt.Errorf("routing number must be 9 digits")
	}

	// Reject all zeros
	if routing == "000000000" {
		return fmt.Errorf("invalid routing number")
	}

	// ABA check digit algorithm
	weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0

	for i, weight := range weights {
		digit, err := strconv.Atoi(string(routing[i]))
		if err != nil {
			return fmt.Errorf("routing number must contain only digits")
		}
		sum += digit * weight
	}

	if sum%10 != 0 {
		return fmt.Errorf("invalid routing number check digit")
	}

	return nil
}

// ValidateSwiftCode validates SWIFT/BIC code format
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func ValidateSwiftCode(swift string) error {
	if !swiftCodeRegex.MatchString(swift) {
		return fmt.Errorf("invalid SWIFT code format: must be 8 or 11 characters (AAAABBCC or AAAABBCCDDD)")
	}
	return nil
}

// CreateInstitution creates a new financial institution
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func (im *InstitutionManager) CreateInstitution(ctx context.Context, req *pb.CreateInstitutionRequest) (*pb.FinancialInstitution, error) {
	// Validate required fields
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "institution code is required")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "institution name is required")
	}
	if req.CountryCode == "" {
		return nil, status.Error(codes.InvalidArgument, "country code is required")
	}
	if req.InstitutionType == pb.InstitutionType_INSTITUTION_TYPE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "institution type is required")
	}

	// Validate country code format
	if !countryCodeRegex.MatchString(req.CountryCode) {
		return nil, status.Error(codes.InvalidArgument, "invalid country code format: must be 2 uppercase letters")
	}

	// Validate SWIFT code if provided
	if req.SwiftCode != "" {
		if err := ValidateSwiftCode(req.SwiftCode); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid SWIFT code: %v", err)
		}
	}

	// Validate routing numbers for US banks
	if req.CountryCode == "US" && len(req.RoutingNumbers) > 0 {
		for _, rn := range req.RoutingNumbers {
			if err := ValidateRoutingNumber(rn.RoutingNumber); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid routing number %s: %v", rn.RoutingNumber, err)
			}
		}
	}

	// Check for duplicate code
	var exists bool
	err := im.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM treasury.financial_institutions WHERE code = $1)",
		req.Code).Scan(&exists)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check institution existence: %v", err)
	}
	if exists {
		return nil, status.Errorf(codes.AlreadyExists, "institution with code %s already exists", req.Code)
	}

	// Begin transaction
	tx, err := im.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Insert institution
	institutionID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO treasury.financial_institutions (
			id, code, name, short_name, swift_code,
			iban_prefix, bank_code, branch_code,
			institution_type, country_code, primary_currency,
			street_address_1, street_address_2, city, state_province, postal_code,
			phone_number, fax_number, email_address, website_url,
			time_zone, business_hours, holiday_calendar,
			regulatory_id, tax_id, licenses,
			status, is_active, activated_at,
			capabilities, notes, external_references,
			created_at, updated_at, created_by, version
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23,
			$24, $25, $26,
			$27, $28, $29,
			$30, $31, $32,
			$33, $34, $35, $36
		) RETURNING created_at, updated_at`

	var createdAt, updatedAt time.Time
	
	// Convert institution type enum to string
	institutionTypeStr := institutionTypeToString(req.InstitutionType)
	
	// Handle optional fields
	var address *pb.Address
	var contact *pb.ContactInfo
	if req.Address != nil {
		address = req.Address
	}
	if req.Contact != nil {
		contact = req.Contact
	}

	err = tx.QueryRowContext(ctx, query,
		institutionID, req.Code, req.Name, nullString(req.ShortName), nullString(req.SwiftCode),
		nil, nullString(req.BankCode), nullString(req.BranchCode),
		institutionTypeStr, req.CountryCode, nullString(req.PrimaryCurrency),
		addressField(address, "street_address_1"), addressField(address, "street_address_2"),
		addressField(address, "city"), addressField(address, "state_province"),
		addressField(address, "postal_code"),
		contactField(contact, "phone_number"), contactField(contact, "fax_number"),
		contactField(contact, "email_address"), contactField(contact, "website_url"),
		nullString(req.TimeZone), structToJSON(req.Capabilities), nil,
		nil, nil, nil,
		"active", true, now,
		structToJSON(req.Capabilities), nullString(req.Notes), nil,
		now, now, "system", 1,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create institution: %v", err)
	}

	// Insert routing numbers
	var routingNumbers []*pb.RoutingNumber
	for _, rn := range req.RoutingNumbers {
		routingID := uuid.New()
		routingQuery := `
			INSERT INTO treasury.institution_routing_numbers (
				id, institution_id, routing_number, routing_type,
				is_primary, description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING created_at, updated_at`

		var rnCreatedAt, rnUpdatedAt time.Time
		routingType := "standard"
		if rn.RoutingType != "" {
			routingType = rn.RoutingType
		}

		err = tx.QueryRowContext(ctx, routingQuery,
			routingID, institutionID, rn.RoutingNumber, routingType,
			rn.IsPrimary, nullString(rn.Description), now, now,
		).Scan(&rnCreatedAt, &rnUpdatedAt)

		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create routing number: %v", err)
		}

		routingNumbers = append(routingNumbers, &pb.RoutingNumber{
			Id:            routingID.String(),
			RoutingNumber: rn.RoutingNumber,
			RoutingType:   routingType,
			IsPrimary:     rn.IsPrimary,
			Description:   rn.Description,
			CreatedAt:     timestamppb.New(rnCreatedAt),
			UpdatedAt:     timestamppb.New(rnUpdatedAt),
		})
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	// Build response
	institution := &pb.FinancialInstitution{
		Id:               institutionID.String(),
		Code:             req.Code,
		Name:             req.Name,
		ShortName:        req.ShortName,
		RoutingNumbers:   routingNumbers,
		SwiftCode:        req.SwiftCode,
		BankCode:         req.BankCode,
		BranchCode:       req.BranchCode,
		InstitutionType:  req.InstitutionType,
		CountryCode:      req.CountryCode,
		PrimaryCurrency:  req.PrimaryCurrency,
		Address:          req.Address,
		Contact:          req.Contact,
		TimeZone:         req.TimeZone,
		BusinessHours:    req.Capabilities,
		Status:           pb.InstitutionStatus_INSTITUTION_STATUS_ACTIVE,
		IsActive:         true,
		ActivatedAt:      timestamppb.New(now),
		Capabilities:     req.Capabilities,
		Notes:            req.Notes,
		CreatedAt:        timestamppb.New(createdAt),
		UpdatedAt:        timestamppb.New(updatedAt),
		CreatedBy:        "system",
		Version:          1,
	}

	return institution, nil
}

// GetInstitution retrieves institution information
// Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
func (im *InstitutionManager) GetInstitution(ctx context.Context, req *pb.GetInstitutionRequest) (*pb.FinancialInstitution, error) {
	var query string
	var args []interface{}

	// Determine lookup method based on identifier
	switch id := req.Identifier.(type) {
	case *pb.GetInstitutionRequest_Code:
		query = `
			SELECT i.id, i.code, i.name, i.short_name, i.swift_code,
				i.iban_prefix, i.bank_code, i.branch_code,
				i.institution_type, i.country_code, i.primary_currency,
				i.street_address_1, i.street_address_2, i.city, i.state_province, i.postal_code,
				i.phone_number, i.fax_number, i.email_address, i.website_url,
				i.time_zone, i.business_hours, i.holiday_calendar,
				i.regulatory_id, i.tax_id, i.licenses,
				i.status, i.is_active, i.activated_at, i.deactivated_at, i.suspension_reason,
				i.capabilities, i.notes, i.external_references,
				i.created_at, i.updated_at, i.created_by, i.updated_by, i.version
			FROM treasury.financial_institutions i
			WHERE i.code = $1 AND i.status != 'deleted'`
		args = []interface{}{id.Code}

	case *pb.GetInstitutionRequest_RoutingNumber:
		query = `
			SELECT i.id, i.code, i.name, i.short_name, i.swift_code,
				i.iban_prefix, i.bank_code, i.branch_code,
				i.institution_type, i.country_code, i.primary_currency,
				i.street_address_1, i.street_address_2, i.city, i.state_province, i.postal_code,
				i.phone_number, i.fax_number, i.email_address, i.website_url,
				i.time_zone, i.business_hours, i.holiday_calendar,
				i.regulatory_id, i.tax_id, i.licenses,
				i.status, i.is_active, i.activated_at, i.deactivated_at, i.suspension_reason,
				i.capabilities, i.notes, i.external_references,
				i.created_at, i.updated_at, i.created_by, i.updated_by, i.version
			FROM treasury.financial_institutions i
			JOIN treasury.institution_routing_numbers r ON i.id = r.institution_id
			WHERE r.routing_number = $1 AND i.status != 'deleted'`
		args = []interface{}{id.RoutingNumber}

	case *pb.GetInstitutionRequest_SwiftCode:
		query = `
			SELECT i.id, i.code, i.name, i.short_name, i.swift_code,
				i.iban_prefix, i.bank_code, i.branch_code,
				i.institution_type, i.country_code, i.primary_currency,
				i.street_address_1, i.street_address_2, i.city, i.state_province, i.postal_code,
				i.phone_number, i.fax_number, i.email_address, i.website_url,
				i.time_zone, i.business_hours, i.holiday_calendar,
				i.regulatory_id, i.tax_id, i.licenses,
				i.status, i.is_active, i.activated_at, i.deactivated_at, i.suspension_reason,
				i.capabilities, i.notes, i.external_references,
				i.created_at, i.updated_at, i.created_by, i.updated_by, i.version
			FROM treasury.financial_institutions i
			WHERE i.swift_code = $1 AND i.status != 'deleted'`
		args = []interface{}{id.SwiftCode}

	case *pb.GetInstitutionRequest_Id:
		query = `
			SELECT i.id, i.code, i.name, i.short_name, i.swift_code,
				i.iban_prefix, i.bank_code, i.branch_code,
				i.institution_type, i.country_code, i.primary_currency,
				i.street_address_1, i.street_address_2, i.city, i.state_province, i.postal_code,
				i.phone_number, i.fax_number, i.email_address, i.website_url,
				i.time_zone, i.business_hours, i.holiday_calendar,
				i.regulatory_id, i.tax_id, i.licenses,
				i.status, i.is_active, i.activated_at, i.deactivated_at, i.suspension_reason,
				i.capabilities, i.notes, i.external_references,
				i.created_at, i.updated_at, i.created_by, i.updated_by, i.version
			FROM treasury.financial_institutions i
			WHERE i.id = $1 AND i.status != 'deleted'`
		args = []interface{}{id.Id}

	default:
		return nil, status.Error(codes.InvalidArgument, "identifier is required")
	}

	// Execute query and scan institution
	institution, err := im.scanInstitution(ctx, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "institution not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to retrieve institution: %v", err)
	}

	// Load routing numbers
	routingNumbers, err := im.loadRoutingNumbers(ctx, institution.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load routing numbers: %v", err)
	}
	institution.RoutingNumbers = routingNumbers

	return institution, nil
}

// UpdateInstitution updates institution information
// Spec: docs/specs/004-financial-institutions.md#story-3-update-institution-information
func (im *InstitutionManager) UpdateInstitution(ctx context.Context, req *pb.UpdateInstitutionRequest) (*pb.FinancialInstitution, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "institution code is required")
	}

	// Begin transaction
	tx, err := im.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Check institution exists and get current version
	var institutionID uuid.UUID
	var currentVersion int32
	err = tx.QueryRowContext(ctx,
		"SELECT id, version FROM treasury.financial_institutions WHERE code = $1 AND status != 'deleted'",
		req.Code).Scan(&institutionID, &currentVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "institution not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to check institution: %v", err)
	}

	// Check optimistic locking
	if req.Version > 0 && req.Version != currentVersion {
		return nil, status.Error(codes.Aborted, "version mismatch - institution was modified by another process")
	}

	// Build dynamic update query based on update mask
	updateFields := []string{"updated_at = CURRENT_TIMESTAMP", "version = version + 1"}
	updateArgs := []interface{}{}
	argCount := 1

	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		for _, path := range req.UpdateMask.Paths {
			switch path {
			case "name":
				updateFields = append(updateFields, fmt.Sprintf("name = $%d", argCount))
				updateArgs = append(updateArgs, req.Name)
				argCount++
			case "short_name":
				updateFields = append(updateFields, fmt.Sprintf("short_name = $%d", argCount))
				updateArgs = append(updateArgs, nullString(req.ShortName))
				argCount++
			case "swift_code":
				if req.SwiftCode != "" {
					if err := ValidateSwiftCode(req.SwiftCode); err != nil {
						return nil, status.Errorf(codes.InvalidArgument, "invalid SWIFT code: %v", err)
					}
				}
				updateFields = append(updateFields, fmt.Sprintf("swift_code = $%d", argCount))
				updateArgs = append(updateArgs, nullString(req.SwiftCode))
				argCount++
			case "status":
				statusStr := institutionStatusToString(req.Status)
				updateFields = append(updateFields, fmt.Sprintf("status = $%d", argCount))
				updateArgs = append(updateArgs, statusStr)
				argCount++
				if req.Status == pb.InstitutionStatus_INSTITUTION_STATUS_INACTIVE ||
					req.Status == pb.InstitutionStatus_INSTITUTION_STATUS_SUSPENDED ||
					req.Status == pb.InstitutionStatus_INSTITUTION_STATUS_DELETED {
					updateFields = append(updateFields, "is_active = false", "deactivated_at = CURRENT_TIMESTAMP")
				}
			case "notes":
				updateFields = append(updateFields, fmt.Sprintf("notes = $%d", argCount))
				updateArgs = append(updateArgs, nullString(req.Notes))
				argCount++
			case "capabilities":
				updateFields = append(updateFields, fmt.Sprintf("capabilities = $%d", argCount))
				updateArgs = append(updateArgs, structToJSON(req.Capabilities))
				argCount++
			case "address":
				if req.Address != nil {
					updateFields = append(updateFields, 
						fmt.Sprintf("street_address_1 = $%d", argCount),
						fmt.Sprintf("street_address_2 = $%d", argCount+1),
						fmt.Sprintf("city = $%d", argCount+2),
						fmt.Sprintf("state_province = $%d", argCount+3),
						fmt.Sprintf("postal_code = $%d", argCount+4))
					updateArgs = append(updateArgs, 
						nullString(req.Address.StreetAddress_1),
						nullString(req.Address.StreetAddress_2),
						nullString(req.Address.City),
						nullString(req.Address.StateProvince),
						nullString(req.Address.PostalCode))
					argCount += 5
				}
			case "contact":
				if req.Contact != nil {
					updateFields = append(updateFields,
						fmt.Sprintf("phone_number = $%d", argCount),
						fmt.Sprintf("fax_number = $%d", argCount+1),
						fmt.Sprintf("email_address = $%d", argCount+2),
						fmt.Sprintf("website_url = $%d", argCount+3))
					updateArgs = append(updateArgs,
						nullString(req.Contact.PhoneNumber),
						nullString(req.Contact.FaxNumber),
						nullString(req.Contact.EmailAddress),
						nullString(req.Contact.WebsiteUrl))
					argCount += 4
				}
			}
		}
	}

	// Add WHERE clause
	updateArgs = append(updateArgs, req.Code)
	updateQuery := fmt.Sprintf(
		"UPDATE treasury.financial_institutions SET %s WHERE code = $%d",
		strings.Join(updateFields, ", "),
		argCount,
	)

	// Execute update
	_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update institution: %v", err)
	}

	// Update routing numbers if provided
	if len(req.RoutingNumbers) > 0 {
		// Delete existing routing numbers
		_, err = tx.ExecContext(ctx,
			"DELETE FROM treasury.institution_routing_numbers WHERE institution_id = $1",
			institutionID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to delete routing numbers: %v", err)
		}

		// Insert new routing numbers
		for _, rn := range req.RoutingNumbers {
			if err := ValidateRoutingNumber(rn.RoutingNumber); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid routing number %s: %v", rn.RoutingNumber, err)
			}

			routingType := "standard"
			if rn.RoutingType != "" {
				routingType = rn.RoutingType
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO treasury.institution_routing_numbers (
					id, institution_id, routing_number, routing_type,
					is_primary, description, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
				uuid.New(), institutionID, rn.RoutingNumber, routingType,
				rn.IsPrimary, nullString(rn.Description))

			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to insert routing number: %v", err)
			}
		}
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "failed to commit transaction")
	}

	// Retrieve and return updated institution
	return im.GetInstitution(ctx, &pb.GetInstitutionRequest{
		Identifier: &pb.GetInstitutionRequest_Code{Code: req.Code},
	})
}

// DeleteInstitution soft deletes an institution
// Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
func (im *InstitutionManager) DeleteInstitution(ctx context.Context, req *pb.DeleteInstitutionRequest) (*pb.DeleteInstitutionResponse, error) {
	if req.Code == "" {
		return nil, status.Error(codes.InvalidArgument, "institution code is required")
	}

	// Check for references if not forcing
	if !req.Force {
		refs, err := im.CheckReferences(ctx, req.Code)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check references: %v", err)
		}
		if len(refs) > 0 {
			var blockingRefs []string
			for _, ref := range refs {
				blockingRefs = append(blockingRefs, fmt.Sprintf("%s.%s (%d references)", 
					ref.TableName, ref.ColumnName, ref.Count))
			}
			return &pb.DeleteInstitutionResponse{
				Success:             false,
				BlockingReferences:  blockingRefs,
			}, nil
		}
	}

	// Soft delete by updating status
	query := `
		UPDATE treasury.financial_institutions 
		SET status = 'deleted', 
			is_active = false,
			deactivated_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP,
			updated_by = $1,
			version = version + 1
		WHERE code = $2 AND status != 'deleted'`

	result, err := im.db.ExecContext(ctx, query, req.DeletedBy, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete institution: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "institution not found or already deleted")
	}

	return &pb.DeleteInstitutionResponse{
		Success: true,
	}, nil
}

// ListInstitutions lists institutions with filtering
// Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
func (im *InstitutionManager) ListInstitutions(ctx context.Context, req *pb.ListInstitutionsRequest) (*pb.ListInstitutionsResponse, error) {
	// Build query with filters
	query := `
		SELECT i.id, i.code, i.name, i.short_name, i.swift_code,
			i.iban_prefix, i.bank_code, i.branch_code,
			i.institution_type, i.country_code, i.primary_currency,
			i.street_address_1, i.street_address_2, i.city, i.state_province, i.postal_code,
			i.phone_number, i.fax_number, i.email_address, i.website_url,
			i.time_zone, i.business_hours, i.holiday_calendar,
			i.regulatory_id, i.tax_id, i.licenses,
			i.status, i.is_active, i.activated_at, i.deactivated_at, i.suspension_reason,
			i.capabilities, i.notes, i.external_references,
			i.created_at, i.updated_at, i.created_by, i.updated_by, i.version
		FROM treasury.financial_institutions i
		WHERE i.status != 'deleted'`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if req.Status != pb.InstitutionStatus_INSTITUTION_STATUS_UNSPECIFIED {
		query += fmt.Sprintf(" AND i.status = $%d", argCount)
		args = append(args, institutionStatusToString(req.Status))
		argCount++
	}

	if req.InstitutionType != pb.InstitutionType_INSTITUTION_TYPE_UNSPECIFIED {
		query += fmt.Sprintf(" AND i.institution_type = $%d", argCount)
		args = append(args, institutionTypeToString(req.InstitutionType))
		argCount++
	}

	if req.CountryCode != "" {
		query += fmt.Sprintf(" AND i.country_code = $%d", argCount)
		args = append(args, req.CountryCode)
		argCount++
	}

	// Apply ordering
	query += " ORDER BY i.name ASC"

	// Apply pagination
	if req.PageSize > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, req.PageSize)
		argCount++
	}

	// Execute query
	rows, err := im.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list institutions: %v", err)
	}
	defer rows.Close()

	// Scan results
	var institutions []*pb.FinancialInstitution
	for rows.Next() {
		institution, err := im.scanInstitutionFromRows(rows)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan institution: %v", err)
		}

		// Load routing numbers for each institution
		routingNumbers, err := im.loadRoutingNumbers(ctx, institution.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to load routing numbers: %v", err)
		}
		institution.RoutingNumbers = routingNumbers

		institutions = append(institutions, institution)
	}

	if err = rows.Err(); err != nil {
		return nil, status.Errorf(codes.Internal, "error iterating institutions: %v", err)
	}

	// Get total count
	var totalCount int32
	countQuery := `
		SELECT COUNT(*) FROM treasury.financial_institutions i
		WHERE i.status != 'deleted'`
	err = im.db.QueryRowContext(ctx, countQuery).Scan(&totalCount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get total count: %v", err)
	}

	return &pb.ListInstitutionsResponse{
		Institutions: institutions,
		TotalCount:   totalCount,
	}, nil
}

// CheckReferences checks for references to an institution
// Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
func (im *InstitutionManager) CheckReferences(ctx context.Context, code string) ([]*pb.CheckInstitutionReferencesResponse_Reference, error) {
	// Get institution ID
	var institutionID uuid.UUID
	err := im.db.QueryRowContext(ctx,
		"SELECT id FROM treasury.financial_institutions WHERE code = $1",
		code).Scan(&institutionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Check for references in known tables
	// This is a placeholder - in a real system, you'd check actual referencing tables
	var references []*pb.CheckInstitutionReferencesResponse_Reference

	// Example: Check treasury_accounts table (when it exists)
	// var count int32
	// err = im.db.QueryRowContext(ctx,
	//     "SELECT COUNT(*) FROM treasury.treasury_accounts WHERE institution_id = $1",
	//     institutionID).Scan(&count)
	// if err == nil && count > 0 {
	//     references = append(references, &pb.CheckInstitutionReferencesResponse_Reference{
	//         TableName:  "treasury_accounts",
	//         ColumnName: "institution_id",
	//         Count:      count,
	//     })
	// }

	return references, nil
}

// Helper functions

// nullString creates a sql.NullString from a string
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func (im *InstitutionManager) scanInstitution(ctx context.Context, query string, args ...interface{}) (*pb.FinancialInstitution, error) {
	row := im.db.QueryRowContext(ctx, query, args...)
	return im.scanInstitutionFromRow(row)
}

func (im *InstitutionManager) scanInstitutionFromRow(row *sql.Row) (*pb.FinancialInstitution, error) {
	var institution pb.FinancialInstitution
	var id uuid.UUID
	var institutionType, status string
	var activatedAt, deactivatedAt, createdAt, updatedAt sql.NullTime
	var shortName, swiftCode, ibanPrefix, bankCode, branchCode sql.NullString
	var primaryCurrency, suspensionReason sql.NullString
	var streetAddress1, streetAddress2, city, stateProvince, postalCode sql.NullString
	var phoneNumber, faxNumber, emailAddress, websiteURL sql.NullString
	var timeZone, holidayCalendar sql.NullString
	var regulatoryID, taxID sql.NullString
	var notes sql.NullString
	var createdBy, updatedBy sql.NullString
	var businessHours, licenses, capabilities, externalRefs []byte

	err := row.Scan(
		&id, &institution.Code, &institution.Name, &shortName, &swiftCode,
		&ibanPrefix, &bankCode, &branchCode,
		&institutionType, &institution.CountryCode, &primaryCurrency,
		&streetAddress1, &streetAddress2, &city, &stateProvince, &postalCode,
		&phoneNumber, &faxNumber, &emailAddress, &websiteURL,
		&timeZone, &businessHours, &holidayCalendar,
		&regulatoryID, &taxID, &licenses,
		&status, &institution.IsActive, &activatedAt, &deactivatedAt, &suspensionReason,
		&capabilities, &notes, &externalRefs,
		&createdAt, &updatedAt, &createdBy, &updatedBy, &institution.Version,
	)

	if err != nil {
		return nil, err
	}

	// Set fields
	institution.Id = id.String()
	institution.ShortName = shortName.String
	institution.SwiftCode = swiftCode.String
	institution.IbanPrefix = ibanPrefix.String
	institution.BankCode = bankCode.String
	institution.BranchCode = branchCode.String
	institution.PrimaryCurrency = primaryCurrency.String
	institution.InstitutionType = stringToInstitutionType(institutionType)
	institution.Status = stringToInstitutionStatus(status)
	institution.TimeZone = timeZone.String
	institution.HolidayCalendar = holidayCalendar.String
	institution.RegulatoryId = regulatoryID.String
	institution.TaxId = taxID.String
	institution.SuspensionReason = suspensionReason.String
	institution.Notes = notes.String
	institution.CreatedBy = createdBy.String
	institution.UpdatedBy = updatedBy.String

	// Set timestamps
	if activatedAt.Valid {
		institution.ActivatedAt = timestamppb.New(activatedAt.Time)
	}
	if deactivatedAt.Valid {
		institution.DeactivatedAt = timestamppb.New(deactivatedAt.Time)
	}
	if createdAt.Valid {
		institution.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		institution.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	// Set address
	if streetAddress1.Valid || city.Valid {
		institution.Address = &pb.Address{
			StreetAddress_1: streetAddress1.String,
			StreetAddress_2: streetAddress2.String,
			City:            city.String,
			StateProvince:   stateProvince.String,
			PostalCode:      postalCode.String,
			CountryCode:     institution.CountryCode,
		}
	}

	// Set contact
	if phoneNumber.Valid || emailAddress.Valid {
		institution.Contact = &pb.ContactInfo{
			PhoneNumber:  phoneNumber.String,
			FaxNumber:    faxNumber.String,
			EmailAddress: emailAddress.String,
			WebsiteUrl:   websiteURL.String,
		}
	}

	// Parse JSON fields
	if len(businessHours) > 0 {
		institution.BusinessHours = jsonToStruct(businessHours)
	}
	if len(licenses) > 0 {
		institution.Licenses = jsonToStruct(licenses)
	}
	if len(capabilities) > 0 {
		institution.Capabilities = jsonToStruct(capabilities)
	}
	if len(externalRefs) > 0 {
		institution.ExternalReferences = jsonToStruct(externalRefs)
	}

	return &institution, nil
}

func (im *InstitutionManager) scanInstitutionFromRows(rows *sql.Rows) (*pb.FinancialInstitution, error) {
	var institution pb.FinancialInstitution
	var id uuid.UUID
	var institutionType, status string
	var activatedAt, deactivatedAt, createdAt, updatedAt sql.NullTime
	var shortName, swiftCode, ibanPrefix, bankCode, branchCode sql.NullString
	var primaryCurrency, suspensionReason sql.NullString
	var streetAddress1, streetAddress2, city, stateProvince, postalCode sql.NullString
	var phoneNumber, faxNumber, emailAddress, websiteURL sql.NullString
	var timeZone, holidayCalendar sql.NullString
	var regulatoryID, taxID sql.NullString
	var notes sql.NullString
	var createdBy, updatedBy sql.NullString
	var businessHours, licenses, capabilities, externalRefs []byte

	err := rows.Scan(
		&id, &institution.Code, &institution.Name, &shortName, &swiftCode,
		&ibanPrefix, &bankCode, &branchCode,
		&institutionType, &institution.CountryCode, &primaryCurrency,
		&streetAddress1, &streetAddress2, &city, &stateProvince, &postalCode,
		&phoneNumber, &faxNumber, &emailAddress, &websiteURL,
		&timeZone, &businessHours, &holidayCalendar,
		&regulatoryID, &taxID, &licenses,
		&status, &institution.IsActive, &activatedAt, &deactivatedAt, &suspensionReason,
		&capabilities, &notes, &externalRefs,
		&createdAt, &updatedAt, &createdBy, &updatedBy, &institution.Version,
	)

	if err != nil {
		return nil, err
	}

	// Set fields (same as scanInstitutionFromRow)
	institution.Id = id.String()
	institution.ShortName = shortName.String
	institution.SwiftCode = swiftCode.String
	institution.IbanPrefix = ibanPrefix.String
	institution.BankCode = bankCode.String
	institution.BranchCode = branchCode.String
	institution.PrimaryCurrency = primaryCurrency.String
	institution.InstitutionType = stringToInstitutionType(institutionType)
	institution.Status = stringToInstitutionStatus(status)
	institution.TimeZone = timeZone.String
	institution.HolidayCalendar = holidayCalendar.String
	institution.RegulatoryId = regulatoryID.String
	institution.TaxId = taxID.String
	institution.SuspensionReason = suspensionReason.String
	institution.Notes = notes.String
	institution.CreatedBy = createdBy.String
	institution.UpdatedBy = updatedBy.String

	// Set timestamps
	if activatedAt.Valid {
		institution.ActivatedAt = timestamppb.New(activatedAt.Time)
	}
	if deactivatedAt.Valid {
		institution.DeactivatedAt = timestamppb.New(deactivatedAt.Time)
	}
	if createdAt.Valid {
		institution.CreatedAt = timestamppb.New(createdAt.Time)
	}
	if updatedAt.Valid {
		institution.UpdatedAt = timestamppb.New(updatedAt.Time)
	}

	// Set address
	if streetAddress1.Valid || city.Valid {
		institution.Address = &pb.Address{
			StreetAddress_1: streetAddress1.String,
			StreetAddress_2: streetAddress2.String,
			City:            city.String,
			StateProvince:   stateProvince.String,
			PostalCode:      postalCode.String,
			CountryCode:     institution.CountryCode,
		}
	}

	// Set contact
	if phoneNumber.Valid || emailAddress.Valid {
		institution.Contact = &pb.ContactInfo{
			PhoneNumber:  phoneNumber.String,
			FaxNumber:    faxNumber.String,
			EmailAddress: emailAddress.String,
			WebsiteUrl:   websiteURL.String,
		}
	}

	// Parse JSON fields
	if len(businessHours) > 0 {
		institution.BusinessHours = jsonToStruct(businessHours)
	}
	if len(licenses) > 0 {
		institution.Licenses = jsonToStruct(licenses)
	}
	if len(capabilities) > 0 {
		institution.Capabilities = jsonToStruct(capabilities)
	}
	if len(externalRefs) > 0 {
		institution.ExternalReferences = jsonToStruct(externalRefs)
	}

	return &institution, nil
}

func (im *InstitutionManager) loadRoutingNumbers(ctx context.Context, institutionID string) ([]*pb.RoutingNumber, error) {
	query := `
		SELECT id, routing_number, routing_type, is_primary, description,
			created_at, updated_at
		FROM treasury.institution_routing_numbers
		WHERE institution_id = $1
		ORDER BY is_primary DESC, routing_number ASC`

	rows, err := im.db.QueryContext(ctx, query, institutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routingNumbers []*pb.RoutingNumber
	for rows.Next() {
		var rn pb.RoutingNumber
		var id uuid.UUID
		var description sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &rn.RoutingNumber, &rn.RoutingType, &rn.IsPrimary,
			&description, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		rn.Id = id.String()
		rn.Description = description.String
		rn.CreatedAt = timestamppb.New(createdAt)
		rn.UpdatedAt = timestamppb.New(updatedAt)

		routingNumbers = append(routingNumbers, &rn)
	}

	return routingNumbers, nil
}

// Conversion helpers

func institutionTypeToString(t pb.InstitutionType) string {
	switch t {
	case pb.InstitutionType_INSTITUTION_TYPE_BANK:
		return "bank"
	case pb.InstitutionType_INSTITUTION_TYPE_CREDIT_UNION:
		return "credit_union"
	case pb.InstitutionType_INSTITUTION_TYPE_INVESTMENT_BANK:
		return "investment_bank"
	case pb.InstitutionType_INSTITUTION_TYPE_CENTRAL_BANK:
		return "central_bank"
	case pb.InstitutionType_INSTITUTION_TYPE_SAVINGS_BANK:
		return "savings_bank"
	case pb.InstitutionType_INSTITUTION_TYPE_ONLINE_BANK:
		return "online_bank"
	case pb.InstitutionType_INSTITUTION_TYPE_OTHER:
		return "other"
	default:
		return "bank"
	}
}

func stringToInstitutionType(s string) pb.InstitutionType {
	switch s {
	case "bank":
		return pb.InstitutionType_INSTITUTION_TYPE_BANK
	case "credit_union":
		return pb.InstitutionType_INSTITUTION_TYPE_CREDIT_UNION
	case "investment_bank":
		return pb.InstitutionType_INSTITUTION_TYPE_INVESTMENT_BANK
	case "central_bank":
		return pb.InstitutionType_INSTITUTION_TYPE_CENTRAL_BANK
	case "savings_bank":
		return pb.InstitutionType_INSTITUTION_TYPE_SAVINGS_BANK
	case "online_bank":
		return pb.InstitutionType_INSTITUTION_TYPE_ONLINE_BANK
	case "other":
		return pb.InstitutionType_INSTITUTION_TYPE_OTHER
	default:
		return pb.InstitutionType_INSTITUTION_TYPE_UNSPECIFIED
	}
}

func institutionStatusToString(s pb.InstitutionStatus) string {
	switch s {
	case pb.InstitutionStatus_INSTITUTION_STATUS_ACTIVE:
		return "active"
	case pb.InstitutionStatus_INSTITUTION_STATUS_INACTIVE:
		return "inactive"
	case pb.InstitutionStatus_INSTITUTION_STATUS_SUSPENDED:
		return "suspended"
	case pb.InstitutionStatus_INSTITUTION_STATUS_DELETED:
		return "deleted"
	default:
		return "active"
	}
}

func stringToInstitutionStatus(s string) pb.InstitutionStatus {
	switch s {
	case "active":
		return pb.InstitutionStatus_INSTITUTION_STATUS_ACTIVE
	case "inactive":
		return pb.InstitutionStatus_INSTITUTION_STATUS_INACTIVE
	case "suspended":
		return pb.InstitutionStatus_INSTITUTION_STATUS_SUSPENDED
	case "deleted":
		return pb.InstitutionStatus_INSTITUTION_STATUS_DELETED
	default:
		return pb.InstitutionStatus_INSTITUTION_STATUS_UNSPECIFIED
	}
}

func addressField(address *pb.Address, field string) interface{} {
	if address == nil {
		return nil
	}
	switch field {
	case "street_address_1":
		return nullString(address.StreetAddress_1)
	case "street_address_2":
		return nullString(address.StreetAddress_2)
	case "city":
		return nullString(address.City)
	case "state_province":
		return nullString(address.StateProvince)
	case "postal_code":
		return nullString(address.PostalCode)
	default:
		return nil
	}
}

func contactField(contact *pb.ContactInfo, field string) interface{} {
	if contact == nil {
		return nil
	}
	switch field {
	case "phone_number":
		return nullString(contact.PhoneNumber)
	case "fax_number":
		return nullString(contact.FaxNumber)
	case "email_address":
		return nullString(contact.EmailAddress)
	case "website_url":
		return nullString(contact.WebsiteUrl)
	default:
		return nil
	}
}

func structToJSON(s *structpb.Struct) interface{} {
	if s == nil {
		return nil
	}
	// Convert to JSON bytes for storage
	// This is a simplified version - in production you'd use proper JSON marshaling
	return nil
}

func jsonToStruct(data []byte) *structpb.Struct {
	if len(data) == 0 {
		return nil
	}
	// Convert from JSON bytes to Struct
	// This is a simplified version - in production you'd use proper JSON unmarshaling
	return nil
}