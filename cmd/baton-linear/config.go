package main

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var apiKey = field.StringField("api-key", field.WithRequired(true), field.WithDescription("The Linear Personal API key used to connect to the Linear API"))

var configuration = field.NewConfiguration([]field.SchemaField{apiKey})
