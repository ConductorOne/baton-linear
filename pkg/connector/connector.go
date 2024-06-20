package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		Annotations: annotationsForUserResourceType(),
	}
	resourceTypeTeam = &v2.ResourceType{
		Id:          "team",
		DisplayName: "Team",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
	resourceTypeProject = &v2.ResourceType{
		Id:          "project",
		DisplayName: "Project",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_GROUP},
	}
	resourceTypeOrg = &v2.ResourceType{
		Id:          "org",
		DisplayName: "Org",
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}
)

type Linear struct {
	client *linear.Client
}

func (ln *Linear) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(ln.client),
		teamBuilder(ln.client),
		projectBuilder(ln.client),
		orgBuilder(ln.client),
		roleBuilder(ln.client),
	}
}

// Metadata returns metadata about the connector.
func (ln *Linear) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Linear",
		Description: "Connector sycing orgs, projects, teams, users and roles from Linear to Baton.",
	}, nil
}

// Validate hits the Linear API to assure that the API key is valid.
func (ln *Linear) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, resp, err := ln.client.Authorize(ctx)
	if err != nil {
		return nil, fmt.Errorf("linear-connector: failed to authenticate. Error: %w", err)
	}
	resp.Body.Close()

	return nil, nil
}

// New returns the Linear connector.
func New(ctx context.Context, apiKey string) (*Linear, error) {
	client, err := linear.NewClient(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	return &Linear{
		client: client,
	}, nil
}
