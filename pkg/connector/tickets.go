package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkTicket "github.com/conductorone/baton-sdk/pkg/types/ticket"
)

func (ln *Linear) GetTicket(ctx context.Context, ticketId string) (*v2.Ticket, annotations.Annotations, error) {
	return nil, nil, fmt.Errorf("GetTicket not implemented")
}

func (ln *Linear) CreateTicket(ctx context.Context, ticket *v2.Ticket, schema *v2.TicketSchema) (*v2.Ticket, annotations.Annotations, error) {
	return nil, nil, fmt.Errorf("CreateTicket not implemented")
}

func (ln *Linear) GetTicketSchema(ctx context.Context, schemaID string) (*v2.TicketSchema, annotations.Annotations, error) {
	return nil, nil, fmt.Errorf("GetTicketSchema not implemented")
}

// Issue Templates
func (ln *Linear) ListTicketSchemas(ctx context.Context, p *pagination.Token) ([]*v2.TicketSchema, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	bag, err := parsePageToken(p.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	// TODO(johnallers): Test with resourcePageSize == 1
	teams, nextToken, _, rlData, err := ln.client.GetTeamsWorkflowStates(ctx, linear.GetTeamVars{After: bag.PageToken(), First: resourcePageSize})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-linear: failed to list teams: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	customFields, err := ln.getCustomFields(ctx)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-linear: failed to list custom fields: %w", err)
	}

	var ret []*v2.TicketSchema
	for _, team := range teams {
		var statuses []*v2.TicketStatus
		for _, state := range team.States.Nodes {
			statuses = append(statuses, &v2.TicketStatus{
				Id:          state.ID,
				DisplayName: state.Name,
			})
		}
		ret = append(ret, &v2.TicketSchema{
			Id:          team.ID,
			DisplayName: team.Name,
			Statuses:    statuses,
			Types: []*v2.TicketType{
				{
					Id:          "issue",
					DisplayName: "Issue",
				},
			},
			CustomFields: customFields,
		})
	}

	return ret, pageToken, annotations, nil
}

func (ln *Linear) getCustomFields(ctx context.Context) (map[string]*v2.TicketCustomField, error) {
	fields, _, _, _, err := ln.client.GetIssueFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("baton-linear: failed to list issue fields: %w", err)
	}

	fieldMap := make(map[string]*v2.TicketCustomField)
	for _, f := range fields {
		if cfSchema, success := getCustomFieldSchema(f); success {
			fieldMap[f.Name] = cfSchema
		}
	}

	return fieldMap, nil
}

func getCustomFieldSchema(field linear.IssueField) (*v2.TicketCustomField, bool) {
	switch field.Type.Kind {
	case "SCALAR":
		switch field.Type.Name {
		case "String":
			return sdkTicket.StringFieldSchema(field.Name, field.Description, false), true
		case "Boolean":
			return sdkTicket.BoolFieldSchema(field.Name, field.Description, false), true
		case "Float":
			return sdkTicket.NumberFieldSchema(field.Name, field.Description, false), true
		case "Int":
			return sdkTicket.NumberFieldSchema(field.Name, field.Description, false), true
		case "JSON":
			// JSON fields are currently internal to Linear
			return nil, false
		}
	case "ENUM":
		enums := make([]string, len(field.Type.EnumValues))
		for i, v := range field.Type.EnumValues {
			enums[i] = v.Name
		}
		return sdkTicket.PickMultipleStringsFieldSchema(field.Name, field.Description, false, enums), true
	case "LIST":
		// TODO(johnallers): Implement LIST fields
		return nil, false
	case "NON_NULL":
		if field.Type.OfType != nil {
			req, success := getCustomFieldSchema(linear.IssueField{
				Name:        field.Name,
				Description: field.Description,
				Type:        *field.Type.OfType,
			})
			if success {
				req.Required = true
				return req, true
			}
		}
	}
	return nil, false
}

func (ln *Linear) BulkCreateTickets(ctx context.Context, request *v2.TicketsServiceBulkCreateTicketsRequest) (*v2.TicketsServiceBulkCreateTicketsResponse, error) {
	return nil, fmt.Errorf("BulkCreateTickets not implemented")
}

func (ln *Linear) BulkGetTickets(ctx context.Context, request *v2.TicketsServiceBulkGetTicketsRequest) (*v2.TicketsServiceBulkGetTicketsResponse, error) {
	return nil, fmt.Errorf("BulkGetTickets not implemented")
}
