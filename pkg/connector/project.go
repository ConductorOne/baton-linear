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
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const (
	associated = "associated"
	membership = "member"
)

type projectResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *projectResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a linear project.
func projectResource(project *linear.Project, parentId *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"project_slug": project.SlugID,
		"project_id":   project.ID,
	}
	groupTraitOptions := []rs.GroupTraitOption{rs.WithGroupProfile(profile)}

	ret, err := rs.NewGroupResource(
		project.Name,
		resourceTypeProject,
		project.ID,
		groupTraitOptions,
		rs.WithParentResourceID(parentId),
	)

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *projectResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeProject.Id})
	if err != nil {
		return nil, "", nil, err
	}

	projects, nextToken, resp, err := o.client.GetProjects(ctx, linear.GetResourcesVars{First: resourcePageSize, After: bag.PageToken()})
	if err != nil {
		return nil, "", nil, fmt.Errorf("linear-connector: failed to list projects: %w", err)
	}
	resp.Body.Close()

	restApiRateLimit, err := extractRateLimitData(resp)
	if err != nil {
		return nil, "", nil, err
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, project := range projects {
		projectCopy := project
		ur, err := projectResource(&projectCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ur)
	}
	annotations.WithRateLimiting(restApiRateLimit)

	return rv, pageToken, annotations, nil
}

func (o *projectResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	membershipOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDisplayName(fmt.Sprintf("%s Project %s", resource.DisplayName, membership)),
		ent.WithDescription(fmt.Sprintf("Member of  %s Linear project", resource.DisplayName)),
	}

	membershipEn := ent.NewAssignmentEntitlement(resource, membership, membershipOptions...)
	rv = append(rv, membershipEn)

	associatedOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeTeam),
		ent.WithDescription(fmt.Sprintf("Team associated with %s Linear project", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Project %s", resource.DisplayName, associated)),
	}

	associatedEn := ent.NewAssignmentEntitlement(resource, associated, associatedOptions...)
	rv = append(rv, associatedEn)

	return rv, "", nil, nil
}

func (o *projectResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	bag, err := parsePageToken(token.Token, resource.Id)
	if err != nil {
		return nil, "", nil, err
	}

	paginationOptions, err := parseMultipleTokens(token)
	if err != nil {
		return nil, "", nil, err
	}

	projectTrait, err := rs.GetGroupTrait(resource)
	if err != nil {
		return nil, "", nil, err
	}

	projectId, ok := rs.GetProfileStringValue(projectTrait.Profile, "project_id")
	if !ok {
		return nil, "", nil, fmt.Errorf("error fetching project_id from project profile")
	}

	project, nextTokens, resp, err := o.client.GetProject(
		ctx,
		linear.GetProjectVars{
			ProjectId:  projectId,
			TeamsAfter: paginationOptions.TeamsAfter,
			UsersAfter: paginationOptions.UsersAfter,
			First:      paginationOptions.First,
		},
	)
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

	var rv []*v2.Grant
	for _, member := range project.Members.Nodes {
		memberCopy := member
		ur, err := userResource(ctx, &memberCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		grant := grant.NewGrant(resource, membership, ur.Id)
		rv = append(rv, grant)
	}

	for _, team := range project.Teams.Nodes {
		teamCopy := team
		tr, err := teamResource(&teamCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		grant := grant.NewGrant(resource, associated, tr.Id)
		rv = append(rv, grant)
	}
	return rv, pageToken, nil, nil
}

func projectBuilder(client *linear.Client) *projectResourceType {
	return &projectResourceType{
		resourceType: resourceTypeProject,
		client:       client,
	}
}
