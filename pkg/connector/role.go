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
)

var _ connectorbuilder.ResourceSyncer = (*roleResourceType)(nil)

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

	roleTraitOptions := []resource.RoleTraitOption{
		resource.WithRoleProfile(profile),
	}

	ret, err := resource.NewRoleResource(
		roleDisplayName,
		resourceTypeRole,
		role,
		roleTraitOptions,
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
	}

	assignmentEn := ent.NewAssignmentEntitlement(resource, membership, assignmentOptions...)
	rv = append(rv, assignmentEn)
	return rv, "", nil, nil
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

// Grant assigns a role to a user.
func (o *roleResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	if principal.Id.ResourceType != resourceTypeUser.Id {
		return nil, fmt.Errorf("linear-connector: only users can be granted roles")
	}

	roleID := entitlement.Resource.Id.Resource
	userID := principal.Id.Resource

	switch roleID {
	case roleAdmin:
		// Promote to admin
		admin := true
		err := o.client.UpdateUser(ctx, userID, &admin, nil)
		if err != nil {
			return nil, fmt.Errorf("linear-connector: failed to grant admin role: %w", err)
		}
	case roleUser:
		// Demote from admin to regular user
		admin := false
		err := o.client.UpdateUser(ctx, userID, &admin, nil)
		if err != nil {
			return nil, fmt.Errorf("linear-connector: failed to grant user role: %w", err)
		}
	case roleGuest, roleOwner:
		// Guest is set at invite time, Owner requires enterprise UI
		return nil, fmt.Errorf("linear-connector: %s role cannot be granted via API", roleID)
	default:
		return nil, fmt.Errorf("linear-connector: unknown role: %s", roleID)
	}

	return nil, nil
}

// Revoke removes a role from a user.
func (o *roleResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	principal := grant.Principal
	if principal.Id.ResourceType != resourceTypeUser.Id {
		return nil, fmt.Errorf("linear-connector: only users can have roles revoked")
	}

	roleID := grant.Entitlement.Resource.Id.Resource
	userID := principal.Id.Resource

	switch roleID {
	case roleAdmin:
		// Demote admin to regular user
		admin := false
		err := o.client.UpdateUser(ctx, userID, &admin, nil)
		if err != nil {
			return nil, fmt.Errorf("linear-connector: failed to revoke admin role: %w", err)
		}
	case roleUser:
		// Deactivate user (suspend)
		active := false
		err := o.client.UpdateUser(ctx, userID, nil, &active)
		if err != nil {
			return nil, fmt.Errorf("linear-connector: failed to suspend user: %w", err)
		}
	case roleGuest, roleOwner:
		return nil, fmt.Errorf("linear-connector: %s role cannot be revoked via API", roleID)
	default:
		return nil, fmt.Errorf("linear-connector: unknown role: %s", roleID)
	}

	return nil, nil
}

func roleBuilder(client *linear.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
