package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const userRoleProfileKey = "user_role"

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Linear user.
func userResource(ctx context.Context, user *linear.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	names := strings.SplitN(user.Name, " ", 2)
	var firstName, lastName string
	switch len(names) {
	case 1:
		firstName = names[0]
	case 2:
		firstName = names[0]
		lastName = names[1]
	}

	var userRole string
	switch {
	case user.Owner:
		userRole = roleOwner
	case user.Admin:
		userRole = roleAdmin
	case user.Guest:
		userRole = roleGuest
	default:
		userRole = roleUser
	}

	profile := map[string]interface{}{
		"first_name":       firstName,
		"last_name":        lastName,
		"login":            user.Email,
		"user_id":          user.ID,
		userRoleProfileKey: userRole,
	}

	userTraitOptions := []sdkResource.UserTraitOption{
		sdkResource.WithUserProfile(profile),
		sdkResource.WithEmail(user.Email, true),
		sdkResource.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
	}

	ret, err := sdkResource.NewUserResource(
		user.Name,
		resourceTypeUser,
		user.ID,
		userTraitOptions,
		sdkResource.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, _, rlData, err := o.client.GetUsers(ctx, linear.GetResourcesVars{First: resourcePageSize, After: bag.PageToken()})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("linear-connector: failed to list users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	var rv []*v2.Resource
	for _, user := range users {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, parentId)
		if err != nil {
			return nil, "", annotations, err
		}
		rv = append(rv, ur)
	}

	return rv, pageToken, annotations, nil
}

func (o *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *userResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	userTrait, err := sdkResource.GetUserTrait(resource)
	if err != nil {
		return nil, "", nil, fmt.Errorf("list-grants: Failed to get user trait from user: %w", err)
	}
	userProfile := userTrait.GetProfile()
	userRole, present := sdkResource.GetProfileStringValue(userProfile, userRoleProfileKey)
	if !present {
		return nil, "", nil, fmt.Errorf("list-grants: user role was not present on profile")
	}
	rr, err := roleResource(ctx, userRole, resource.ParentResourceId)
	if err != nil {
		return nil, "", nil, err
	}
	gr := grant.NewGrant(rr, membership, resource.Id)

	rv = append(rv, gr)
	return rv, "", nil, nil
}

func userBuilder(client *linear.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
