package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	resource "github.com/conductorone/baton-sdk/pkg/types/resource"
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
)

var roles = []string{
	roleGuest,
	roleUser,
	roleAdmin,
}

// Create a new connector resource for a Linear role.
func roleResource(ctx context.Context, role string, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	roleDisplayName := titleCaser.String(role)
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
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, titleCaser.String(membership))),
	}

	assignmentEn := ent.NewAssignmentEntitlement(resource, membership, assignmentOptions...)
	rv = append(rv, assignmentEn)
	return rv, "", nil, nil
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	allUsers, nextToken, resp, err := o.client.GetUsers(ctx, linear.GetResourcesVars{First: resourcePageSize, After: bag.PageToken()})
	if err != nil {
		return nil, "", nil, fmt.Errorf("linear-connector: failed to list users: %w", err)
	}
	resp.Body.Close()

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var userRole string
	var rv []*v2.Grant

	for _, user := range allUsers {
		switch {
		case user.Admin:
			userRole = roleAdmin
		case user.Guest:
			userRole = roleGuest
		default:
			userRole = roleUser
		}

		if resource.Id.Resource == userRole {
			userCopy := user
			ur, err := userResource(ctx, &userCopy, resource.Id)
			if err != nil {
				return nil, "", nil, err
			}

			gr := grant.NewGrant(resource, membership, ur.Id)
			rv = append(rv, gr)
		}
	}

	return rv, pageToken, nil, nil
}

func roleBuilder(client *linear.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
