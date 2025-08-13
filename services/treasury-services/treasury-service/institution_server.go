package main

import (
	"context"

	pb "example.com/go-mono-repo/proto/treasury"
)

// InstitutionServer implements the FinancialInstitutionService gRPC interface
// Spec: docs/specs/004-financial-institutions.md
type InstitutionServer struct {
	pb.UnimplementedFinancialInstitutionServiceServer
	manager *InstitutionManager
}

// NewInstitutionServer creates a new institution server instance
// Spec: docs/specs/004-financial-institutions.md
func NewInstitutionServer(manager *InstitutionManager) *InstitutionServer {
	return &InstitutionServer{
		manager: manager,
	}
}

// CreateInstitution creates a new financial institution
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func (s *InstitutionServer) CreateInstitution(ctx context.Context, req *pb.CreateInstitutionRequest) (*pb.CreateInstitutionResponse, error) {
	institution, err := s.manager.CreateInstitution(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.CreateInstitutionResponse{Institution: institution}, nil
}

// GetInstitution retrieves institution information
// Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
func (s *InstitutionServer) GetInstitution(ctx context.Context, req *pb.GetInstitutionRequest) (*pb.GetInstitutionResponse, error) {
	institution, err := s.manager.GetInstitution(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.GetInstitutionResponse{Institution: institution}, nil
}

// UpdateInstitution updates institution metadata
// Spec: docs/specs/004-financial-institutions.md#story-3-update-institution-information
func (s *InstitutionServer) UpdateInstitution(ctx context.Context, req *pb.UpdateInstitutionRequest) (*pb.UpdateInstitutionResponse, error) {
	institution, err := s.manager.UpdateInstitution(ctx, req)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateInstitutionResponse{Institution: institution}, nil
}

// DeleteInstitution soft deletes an institution
// Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
func (s *InstitutionServer) DeleteInstitution(ctx context.Context, req *pb.DeleteInstitutionRequest) (*pb.DeleteInstitutionResponse, error) {
	return s.manager.DeleteInstitution(ctx, req)
}

// ListInstitutions lists institutions with filtering
// Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
func (s *InstitutionServer) ListInstitutions(ctx context.Context, req *pb.ListInstitutionsRequest) (*pb.ListInstitutionsResponse, error) {
	return s.manager.ListInstitutions(ctx, req)
}

// CheckInstitutionReferences checks for references before deletion
// Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
func (s *InstitutionServer) CheckInstitutionReferences(ctx context.Context, req *pb.CheckInstitutionReferencesRequest) (*pb.CheckInstitutionReferencesResponse, error) {
	refs, err := s.manager.CheckReferences(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	
	canDelete := len(refs) == 0
	return &pb.CheckInstitutionReferencesResponse{
		References: refs,
		CanDelete:  canDelete,
	}, nil
}

// BulkCreateInstitutions creates multiple institutions
// Spec: docs/specs/004-financial-institutions.md#story-5-bulk-institution-operations
func (s *InstitutionServer) BulkCreateInstitutions(ctx context.Context, req *pb.BulkCreateInstitutionsRequest) (*pb.BulkCreateInstitutionsResponse, error) {
	var createdCount, updatedCount, skippedCount int32
	var errors []string

	for _, instReq := range req.Institutions {
		// Try to create the institution
		_, err := s.manager.CreateInstitution(ctx, instReq)
		if err != nil {
			// Check if it already exists
			if req.SkipDuplicates {
				skippedCount++
				continue
			} else if req.UpdateExisting {
				// Try to update instead
				updateReq := &pb.UpdateInstitutionRequest{
					Code:      instReq.Code,
					Name:      instReq.Name,
					ShortName: instReq.ShortName,
					SwiftCode: instReq.SwiftCode,
					Status:    pb.InstitutionStatus_INSTITUTION_STATUS_ACTIVE,
				}
				_, updateErr := s.manager.UpdateInstitution(ctx, updateReq)
				if updateErr != nil {
					errors = append(errors, "Failed to update "+instReq.Code+": "+updateErr.Error())
				} else {
					updatedCount++
				}
			} else {
				errors = append(errors, "Failed to create "+instReq.Code+": "+err.Error())
			}
		} else {
			createdCount++
		}
	}

	return &pb.BulkCreateInstitutionsResponse{
		CreatedCount: createdCount,
		UpdatedCount: updatedCount,
		SkippedCount: skippedCount,
		Errors:       errors,
	}, nil
}