package connector

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ConductorOne/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	resource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type orgResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *orgResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Linear organization.
func orgResource(org *linear.Organization, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	orgOptions := []resource.ResourceOption{
		resource.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: resourceTypeUser.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeTeam.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeRole.Id}),
		resource.WithParentResourceID(parentResourceID)}

	orgResource, err := resource.NewResource(
		org.Name,
		resourceTypeOrg,
		org.ID,
		orgOptions...,
	)
	if err != nil {
		return nil, err
	}
	return orgResource, nil
}

func (o *orgResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeOrg.Id})
	if err != nil {
		return nil, "", nil, err
	}

	paginationOptions, err := parseMultipleTokens(token)
	if err != nil {
		return nil, "", nil, err
	}

	org, nextTokens, resp, err := o.client.GetOrganization(ctx, paginationOptions)
	if err != nil {
		return nil, "", nil, fmt.Errorf("linear-connector: failed to list an organization: %w", err)
	}
	resp.Body.Close()

	var pageToken string
	if nextTokens.TeamsToken != "" || nextTokens.UsersToken != "" {
		stringTokens, err := json.Marshal(nextTokens)
		if err != nil {
			return nil, "", nil, err
		}
		pageToken, err = bag.NextToken(string(stringTokens))
		if err != nil {
			return nil, "", nil, err
		}
	}

	restApiRateLimit, err := extractRateLimitData(resp)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	ur, err := orgResource(&org, parentId)
	if err != nil {
		return nil, "", nil, err
	}

	rv = append(rv, ur)
	annotations.WithRateLimiting(restApiRateLimit)

	return rv, pageToken, annotations, nil
}

func (o *orgResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeTeam, resourceTypeUser),
		ent.WithDescription(fmt.Sprintf("Member of %s Linear org", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Org %s", resource.DisplayName, titleCaser.String(membership))),
	}

	assignmentEn := ent.NewAssignmentEntitlement(resource, membership, assignmentOptions...)
	rv = append(rv, assignmentEn)

	return rv, "", nil, nil
}

func (o *orgResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant

	bag, err := parsePageToken(token.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	paginationOptions, err := parseMultipleTokens(token)
	if err != nil {
		return nil, "", nil, err
	}

	org, nextTokens, resp, err := o.client.GetOrganization(ctx, paginationOptions)
	if err != nil {
		return nil, "", nil, err
	}
	resp.Body.Close()

	var pageToken string
	if nextTokens.TeamsToken != "" || nextTokens.UsersToken != "" {
		stringTokens, err := json.Marshal(nextTokens)
		if err != nil {
			return nil, "", nil, err
		}
		pageToken, err = bag.NextToken(string(stringTokens))
		if err != nil {
			return nil, "", nil, err
		}
	}

	for _, user := range org.Users.Nodes {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		membershipGrant := grant.NewGrant(resource, membership, ur.Id)
		rv = append(rv, membershipGrant)
	}

	for _, team := range org.Teams.Nodes {
		teamCopy := team
		tr, err := teamResource(&teamCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		grant := grant.NewGrant(resource, membership, tr.Id)
		rv = append(rv, grant)
	}

	return rv, pageToken, nil, nil
}

func orgBuilder(client *linear.Client) *orgResourceType {
	return &orgResourceType{
		resourceType: resourceTypeOrg,
		client:       client,
	}
}
