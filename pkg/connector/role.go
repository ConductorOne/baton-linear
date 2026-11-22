package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	resource "github.com/conductorone/baton-sdk/pkg/types/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ connectorbuilder.ResourceSyncer      = (*roleResourceType)(nil)
	_ connectorbuilder.ResourceProvisioner = (*roleResourceType)(nil)
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

const (
	roleGuest = "guest"
	roleUser  = "user"
	roleAdmin = "admin"
	roleOwner = "owner"
	roleApp   = "app"
)

var roles = []string{
	roleGuest,
	roleUser,
	roleAdmin,
	roleOwner,
}

// Create a new connector resource for a Linear role.
func roleResource(ctx context.Context, role string, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	roleDisplayName := titleCase(role)
	profile := map[string]interface{}{
		"role_name": roleDisplayName,
		"role_id":   role,
	}

	roleTraitOptions := []resource.RoleTraitOption{}

	ret, err := resource.NewRoleResource(
		roleDisplayName,
		resourceTypeRole,
		role,
		roleTraitOptions,
		resource.WithResourceProfile(profile),
		resource.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *roleResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	var rv []*v2.Resource
	for _, role := range roles {
		rr, err := roleResource(ctx, role, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, rr)
	}

	return rv, "", nil, nil
}

func (o *roleResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDescription(fmt.Sprintf("%s Linear role", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, titleCase(membership))),
		ent.WithExclusionGroup("linear-workspace-role:" + resource.GetParentResourceId().GetResource()),
	}

	assignmentEn := ent.NewAssignmentEntitlement(resource, membership, assignmentOptions...)
	rv = append(rv, assignmentEn)
	return rv, "", nil, nil
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grant replaces the user's single workspace role with the requested role.
func (o *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	userID, role, err := roleProvisioningTarget(principal, entitlement)
	if err != nil {
		return nil, err
	}
	currentRole, rlData, err := o.client.GetUserRole(ctx, userID)
	annos := annotations.New()
	if rlData != nil {
		annos.Append(rlData)
	}
	if err != nil {
		return annos, fmt.Errorf("baton-linear: reading user role: %w", err)
	}
	if currentRole == roleApp {
		return annos, status.Error(codes.Unimplemented, "baton-linear: app accounts cannot have workspace roles provisioned")
	}
	if currentRole == role {
		annos.Append(&v2.GrantAlreadyExists{})
		return annos, nil
	}
	rlData, err = o.client.ChangeUserRole(ctx, userID, role)
	if rlData != nil {
		annos.Update(rlData)
	}
	if err != nil {
		return annos, fmt.Errorf("baton-linear: granting workspace role: %w", err)
	}
	return annos, nil
}

// Revoke resets a matching role to user; it never changes a different role.
func (o *roleResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	userID, role, err := roleProvisioningTarget(grant.GetPrincipal(), grant.GetEntitlement())
	if err != nil {
		return nil, err
	}
	currentRole, rlData, err := o.client.GetUserRole(ctx, userID)
	annos := annotations.New()
	if rlData != nil {
		annos.Append(rlData)
	}
	if err != nil {
		return annos, fmt.Errorf("baton-linear: reading user role: %w", err)
	}
	if currentRole == roleApp {
		return annos, status.Error(codes.Unimplemented, "baton-linear: app accounts cannot have workspace roles provisioned")
	}
	if currentRole != role {
		annos.Append(&v2.GrantAlreadyRevoked{})
		return annos, nil
	}
	if role == roleUser {
		return annos, status.Error(codes.Unimplemented, "baton-linear: the baseline user role cannot be revoked; Linear requires a workspace role")
	}
	rlData, err = o.client.ChangeUserRole(ctx, userID, roleUser)
	if rlData != nil {
		annos.Update(rlData)
	}
	if err != nil {
		return annos, fmt.Errorf("baton-linear: revoking workspace role: %w", err)
	}
	return annos, nil
}

func roleProvisioningTarget(principal *v2.Resource, entitlement *v2.Entitlement) (string, string, error) {
	if principal.GetId().GetResourceType() != resourceTypeUser.Id {
		return "", "", status.Error(codes.InvalidArgument, "baton-linear: only users can have workspace roles provisioned")
	}
	userID := principal.GetExternalId().GetId() //nolint:staticcheck // Preserve explicitly supplied native IDs; ResourceId remains the fallback.
	if userID == "" {
		// Older syncs store the native Linear UUID only in ResourceId.
		userID = principal.GetId().GetResource()
	}
	if userID == "" {
		return "", "", status.Error(codes.InvalidArgument, "baton-linear: missing Linear user ID")
	}
	if entitlement.GetResource().GetId().GetResourceType() != resourceTypeRole.Id || entitlement.GetSlug() != membership {
		return "", "", status.Error(codes.InvalidArgument, "baton-linear: expected a workspace role membership entitlement")
	}
	role := entitlement.GetResource().GetId().GetResource()
	switch role {
	case roleGuest, roleUser, roleAdmin, roleOwner:
		return userID, role, nil
	case roleApp:
		return "", "", status.Error(codes.Unimplemented, "baton-linear: the app role cannot be granted or revoked")
	default:
		return "", "", status.Errorf(codes.InvalidArgument, "baton-linear: unsupported workspace role %q", role)
	}
}

func roleBuilder(client *linear.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
