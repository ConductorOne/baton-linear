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
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
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

	teams, nextToken, _, rlData, err := o.client.GetTeams(ctx, linear.GetResourcesVars{After: bag.PageToken(), First: resourcePageSize})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("linear-connector: failed to list teams: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	var rv []*v2.Resource
	for _, team := range teams {
		teamCopy := team
		ur, err := teamResource(&teamCopy, parentId)
		if err != nil {
			return nil, "", annotations, err
		}
		rv = append(rv, ur)
	}

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

	team, nextToken, _, rlData, err := o.client.GetTeam(ctx, linear.GetTeamVars{TeamId: resource.Id.Resource, After: bag.PageToken(), First: resourcePageSize})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, err
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	for _, membership := range team.Memberships.Nodes {
		membershipCopy := membership
		ur, err := userResource(ctx, &membershipCopy.User, resource.Id)
		if err != nil {
			return nil, "", annotations, err
		}

		metadata := map[string]interface{}{
			"membership_id": membership.ID,
		}

		grant := grant.NewGrant(resource, memberEntitlement, ur.Id, grant.WithGrantMetadata(metadata))
		rv = append(rv, grant)
	}

	return rv, pageToken, annotations, nil
}

func (o *teamResourceType) Grant(ctx context.Context, principal *v2.Resource, entitlement *v2.Entitlement) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"baton-linear: only users can be granted team membership",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-linear: only users can be granted team membership")
	}

	_, err := o.client.AddMemberToTeam(ctx, entitlement.Resource.Id.Resource, principal.Id.Resource)
	if err != nil {
		return nil, fmt.Errorf("baton-linear: failed adding user to team: %w", err)
	}
	return nil, nil
}

func (o *teamResourceType) Revoke(ctx context.Context, grant *v2.Grant) (annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)

	principal := grant.Principal

	if principal.Id.ResourceType != resourceTypeUser.Id {
		l.Warn(
			"baton-linear: only users can have team membership revoked",
			zap.String("principal_type", principal.Id.ResourceType),
			zap.String("principal_id", principal.Id.Resource),
		)
		return nil, fmt.Errorf("baton-linear: only users can have team membership revoked")
	}

	metadata := &structpb.Struct{}
	annos := annotations.Annotations(grant.Annotations)
	ok, err := annos.Pick(metadata)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("baton-linear: annotation does not exist: %w", err)
	}

	membershipId := metadata.Fields["membership_id"].GetStringValue()

	success, err := o.client.RemoveTeamMembership(ctx, membershipId)
	if err != nil {
		return nil, fmt.Errorf("baton-linear: failed removing user from team: %w", err)
	}

	if !success {
		return nil, fmt.Errorf("baton-linear: failed removing user from team")
	}

	return nil, nil
}

func teamBuilder(client *linear.Client) *teamResourceType {
	return &teamResourceType{
		resourceType: resourceTypeTeam,
		client:       client,
	}
}
