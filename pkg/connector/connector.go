package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
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
	}
}

// Metadata returns metadata about the connector.
func (ln *Linear) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Linear",
	}, nil
}

// Validate hits the Linear API to validate that the API key passed has admin rights.
func (ln *Linear) Validate(ctx context.Context) (annotations.Annotations, error) {
	currentUser, resp, err := ln.client.Authorize(ctx)
	if err != nil {
		return nil, fmt.Errorf("linear-connector: failed to authenticate. Error: %w", err)
	}
	resp.Body.Close()

	if !currentUser.Admin {
		return nil, fmt.Errorf("linear-connector: authenticated user is not an admin")
	}

	return nil, nil
}

// New returns the Linear connector.
func New(ctx context.Context, apiKey string) (*Linear, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &Linear{
		client: linear.NewClient(apiKey, httpClient),
	}, nil
}
