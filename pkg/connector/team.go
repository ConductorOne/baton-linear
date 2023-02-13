package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const memberEntitlement = "member"

type teamResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *teamResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Linear team.
func teamResource(team *linear.Team, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"team_id":   team.ID,
		"team_name": team.Name,
	}

	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}

	ret, err := rs.NewGroupResource(
		team.Name,
		resourceTypeTeam,
		team.ID,
		groupTraitOptions,
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *teamResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	teams, nextToken, resp, err := o.client.GetTeams(ctx, linear.GetResourcesVars{After: bag.PageToken(), First: resourcePageSize})
	if err != nil {
		return nil, "", nil, fmt.Errorf("linear-connector: failed to list teams: %w", err)
	}
	resp.Body.Close()

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	restApiRateLimit, err := extractRateLimitData(resp)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, team := range teams {
		teamCopy := team
		ur, err := teamResource(&teamCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}
	annotations.WithRateLimiting(restApiRateLimit)

	return rv, pageToken, annotations, nil
}

func (o *teamResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assigmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDescription(fmt.Sprintf("Member of %s team in Linear", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Team %s", resource.DisplayName, memberEntitlement)),
	}

	en := ent.NewAssignmentEntitlement(resource, memberEntitlement, assigmentOptions...)
	rv = append(rv, en)

	return rv, "", nil, nil
}

func (o *teamResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	var rv []*v2.Grant

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	teamTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	teamId, ok := rs.GetProfileStringValue(teamTrait.Profile, "team_id")
	if !ok {
		return nil, "", nil, fmt.Errorf("error fetching team_id from team profile")
	}

	team, nextToken, resp, err := o.client.GetTeam(ctx, linear.GetTeamVars{TeamId: teamId, After: bag.PageToken(), First: resourcePageSize})
	if err != nil {
		return nil, "", nil, err
	}
	resp.Body.Close()

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	restApiRateLimit, err := extractRateLimitData(resp)
	if err != nil {
		return nil, "", nil, err
	}

	annotations.WithRateLimiting(restApiRateLimit)

	for _, member := range team.Members.Nodes {
		memberCopy := member
		ur, err := userResource(ctx, &memberCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		grant := grant.NewGrant(resource, memberEntitlement, ur.Id)
		rv = append(rv, grant)
	}

	return rv, pageToken, nil, nil
}

func teamBuilder(client *linear.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
