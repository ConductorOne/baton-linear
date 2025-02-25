package connector

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkTicket "github.com/conductorone/baton-sdk/pkg/types/ticket"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (ln *Linear) GetTicket(ctx context.Context, ticketId string) (*v2.Ticket, annotations.Annotations, error) {
	issue, err := ln.client.GetIssue(ctx, ticketId)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to get issue: %w", err)
	}

	return ticketFromIssue(issue), nil, nil
}

func ticketFromIssue(issue *linear.Issue) *v2.Ticket {
	var labels []string
	if issue.Labels.Nodes != nil {
		for _, label := range issue.Labels.Nodes {
			labels = append(labels, label.Name)
		}
	}

	return &v2.Ticket{
		Id:          issue.ID,
		DisplayName: issue.Title,
		Description: issue.Description,
		Status:      &v2.TicketStatus{Id: issue.State.ID, DisplayName: issue.State.Name},
		Labels:      labels,
		Url:         issue.URL,
		CreatedAt:   timestamppb.New(issue.CreatedAt),
		UpdatedAt:   timestamppb.New(issue.UpdatedAt),
	}
}

func (ln *Linear) CreateTicket(ctx context.Context, ticket *v2.Ticket, schema *v2.TicketSchema) (*v2.Ticket, annotations.Annotations, error) {
	payload := linear.CreateIssuePayload{
		TeamId:      schema.Id,
		Title:       ticket.DisplayName,
		Description: ticket.Description,
	}

	// TODO(johnallers) Add Labels
	// TODO(johnallers) Add Custom Fields

	issue, err := ln.client.CreateIssue(ctx, payload)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to create issue: %w", err)
	}

	ticketResp := ticketFromIssue(issue)

	return ticketResp, nil, nil
}

func (ln *Linear) GetTicketSchema(ctx context.Context, schemaID string) (*v2.TicketSchema, annotations.Annotations, error) {
	teams, _, _, _, err := ln.client.ListTeamWorkflowStates(ctx, linear.GetTeamsVars{TeamIds: []string{schemaID}, First: 2})
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to list team workflow states: %w", err)
	}

	if len(teams) != 1 {
		return nil, nil, fmt.Errorf("baton-linear: expected 1 team, got %d", len(teams))
	}

	customFields, err := ln.getCustomFields(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to list custom fields: %w", err)
	}

	return ticketSchemaFromTeam(teams[0], customFields), nil, nil
}

func (ln *Linear) ListTicketSchemas(ctx context.Context, p *pagination.Token) ([]*v2.TicketSchema, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	bag, err := parsePageToken(p.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	// TODO(johnallers): Test with resourcePageSize == 1
	// Linear Issues currently vary only by Workflow State per Team
	teams, nextToken, _, rlData, err := ln.client.ListTeamWorkflowStates(ctx, linear.GetTeamsVars{After: bag.PageToken(), First: resourcePageSize})
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
		ret = append(ret, ticketSchemaFromTeam(team, customFields))
	}

	return ret, pageToken, annotations, nil
}

func ticketSchemaFromTeam(team linear.Team, customFields map[string]*v2.TicketCustomField) *v2.TicketSchema {
	var statuses []*v2.TicketStatus

	sort.Slice(team.States.Nodes, func(i, j int) bool {
		return team.States.Nodes[i].Position < team.States.Nodes[j].Position
	})

	for _, state := range team.States.Nodes {
		statuses = append(statuses, &v2.TicketStatus{
			Id:          state.ID,
			DisplayName: state.Name,
		})
	}

	return &v2.TicketSchema{
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
	}
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
	if strings.HasPrefix(field.Description, "[Internal]") {
		return nil, false
	}
	switch field.Name {
	case "priority":
		objectValues := []*v2.TicketCustomFieldObjectValue{
			{Id: "0", DisplayName: "No priority"},
			{Id: "1", DisplayName: "Urgent"},
			{Id: "2", DisplayName: "High"},
			{Id: "3", DisplayName: "Normal"},
			{Id: "4", DisplayName: "Low"},
		}
		return sdkTicket.PickObjectValueFieldSchema(field.Name, field.Name, true, objectValues), true
	case "assigneeId", "createAsUser", "cycleId", "projectId", "projectMilestoneId", "stateId", "subscriberIds", "templateId":
		switch field.Type.Kind {
		case "SCALAR":
			switch field.Type.Name {
			case "String":
				return sdkTicket.StringFieldSchema(field.Name, field.Name, false), true
			case "Boolean":
				return sdkTicket.BoolFieldSchema(field.Name, field.Name, false), true
			case "Float":
				return sdkTicket.StringFieldSchema(field.Name, field.Name, false), true
			case "Int":
				return sdkTicket.StringFieldSchema(field.Name, field.Name, false), true
			case "JSON", "DateTime", "TimelessDate":
				return nil, false
			}
		case "ENUM":
			enums := make([]string, len(field.Type.EnumValues))
			for i, v := range field.Type.EnumValues {
				enums[i] = v.Name
			}
			return sdkTicket.PickMultipleStringsFieldSchema(field.Name, field.Name, false, enums), true
		case "LIST":
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
	}

	return nil, false
}

func (ln *Linear) BulkCreateTickets(ctx context.Context, request *v2.TicketsServiceBulkCreateTicketsRequest) (*v2.TicketsServiceBulkCreateTicketsResponse, error) {
	return nil, fmt.Errorf("BulkCreateTickets not implemented")
}

func (ln *Linear) BulkGetTickets(ctx context.Context, request *v2.TicketsServiceBulkGetTicketsRequest) (*v2.TicketsServiceBulkGetTicketsResponse, error) {
	return nil, fmt.Errorf("BulkGetTickets not implemented")
}
