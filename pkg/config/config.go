package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	apiKey = field.StringField(
		"api-key",
		field.WithDisplayName("API key"),
		field.WithRequired(true),
		field.WithDescription("The Linear Personal API key used to connect to the Linear API"),
		field.WithIsSecret(true),
	)
	skipProjects = field.BoolField(
		"skip-projects",
		field.WithDisplayName("Skip projects"),
		field.WithDescription("Skip syncing projects."),
	)
	teamIDsTicketSchemaFilterField = field.StringSliceField(
		"ticket-schema-team-ids-filter",
		field.WithDisplayName("Teams"),
		field.WithDescription("Comma-separated list of team IDs to use for tickets schemas."),
	)
	baseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Linear API URL (for testing)"),
	)
)

var externalTicketField = field.TicketingField.ExportAs(field.ExportTargetGUI)
var configRelations = []field.SchemaFieldRelationship{
	field.FieldsDependentOn([]field.SchemaField{teamIDsTicketSchemaFilterField}, []field.SchemaField{field.TicketingField}),
}

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{apiKey, externalTicketField, skipProjects, teamIDsTicketSchemaFilterField, baseURLField},
	field.WithConstraints(configRelations...),
	field.WithConnectorDisplayName("Linear"),
	field.WithHelpUrl("/docs/baton/linear"),
	field.WithIconUrl("/static/app-icons/linear.svg"),
)
