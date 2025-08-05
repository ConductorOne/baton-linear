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
)

var externalTicketField = field.TicketingField.ExportAs(field.ExportTargetGUI)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	[]field.SchemaField{apiKey, externalTicketField, skipProjects},
	field.WithConnectorDisplayName("Linear"),
	field.WithHelpUrl("/docs/baton/linear"),
	field.WithIconUrl("/static/app-icons/linear.svg"),
)
