package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
)

// RoleResourceTypeID is exported because main.go uses it with
// cli.ConnectorOpts.WillSyncResourceType to decide whether role grants
// should be emitted from the user syncer.
const RoleResourceTypeID = "role"

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
		// userBuilder clones this and adds SkipEntitlements by default, or
		// SkipEntitlementsAndGrants when roles aren't being synced. Any
		// annotations declared here are preserved on the clone.
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
		Id:          RoleResourceTypeID,
		DisplayName: "Role",
		Traits:      []v2.ResourceType_Trait{v2.ResourceType_TRAIT_ROLE},
	}
)

type Linear struct {
	client              *linear.Client
	skipProjects        bool
	ticketSchemaTeamIDs []string
	// skipRoleGrants is true when the customer's sync filter excludes the
	// role resource type. The zero value (false) preserves the default
	// behavior, so the zero-value Linear used as the capabilities stub in
	// main.go advertises the standard user resource type.
	skipRoleGrants bool
}

func (ln *Linear) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	resourceSyncers := []connectorbuilder.ResourceSyncer{
		userBuilder(ln.client, ln.skipRoleGrants),
		teamBuilder(ln.client),
		orgBuilder(ln.client),
		roleBuilder(ln.client),
	}

	if !ln.skipProjects {
		resourceSyncers = append(resourceSyncers, projectBuilder(ln.client))
	}

	return resourceSyncers
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
	_, _, err := ln.client.Authorize(ctx)
	if err != nil {
		return nil, fmt.Errorf("linear-connector: failed to authenticate. Error: %w", err)
	}

	return nil, nil
}

// New returns the Linear connector. syncRoles reports whether the role
// resource type will be synced under the current configuration (derived from
// cli.ConnectorOpts.WillSyncResourceType in main.go); when false, the user
// syncer skips emitting role grants.
func New(ctx context.Context, apiKey string, skipProjects bool, ticketSchemaTeamIDs []string, baseURL string, syncRoles bool) (*Linear, error) {
	client, err := linear.NewClient(ctx, apiKey, baseURL)
	if err != nil {
		return nil, err
	}

	return &Linear{
		client:              client,
		skipProjects:        skipProjects,
		ticketSchemaTeamIDs: ticketSchemaTeamIDs,
		skipRoleGrants:      !syncRoles,
	}, nil
}
