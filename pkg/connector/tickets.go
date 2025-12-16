package connector

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	sdkTicket "github.com/conductorone/baton-sdk/pkg/types/ticket"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
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

func (ln *Linear) createIssuePayloadFromTicket(ctx context.Context, ticket *v2.Ticket, schema *v2.TicketSchema) (*linear.CreateIssuePayload, error) {
	payload := linear.CreateIssuePayload{
		TeamId:      schema.Id,
		Title:       ticket.DisplayName,
		Description: ticket.Description,
	}

	ticketFields := ticket.GetCustomFields()
	payload.FieldOptions = make(map[string]interface{})
	for id, cf := range schema.CustomFields {
		val, err := sdkTicket.GetCustomFieldValueOrDefault(ticketFields[id])
		if err != nil {
			return nil, err
		}
		if val == nil {
			continue
		}
		// TODO(johnallers): Need to convert Int/Float fields from String
		if id == "priority" {
			if objVal, ok := val.(*v2.TicketCustomFieldObjectValue); ok {
				if objVal == nil {
					continue
				}
				intVal, err := strconv.Atoi(objVal.Id)
				if err != nil {
					return nil, fmt.Errorf("baton-linear: failed to convert priority to int: %w", err)
				}
				val = intVal
			}
		} else if id == "stateId" {
			if objVal, ok := val.(*v2.TicketCustomFieldObjectValue); ok {
				if objVal == nil {
					continue
				}
				val = objVal.Id
			} else if strVal, ok := val.(string); ok {
				// For backwards compatibility with baton-linear v0.0.11 and earlier
				val = strVal
			}
		}
		payload.FieldOptions[cf.Id] = val
	}

	labelIDs := make([]string, 0, len(ticket.Labels))
	for _, label := range ticket.Labels {
		// Workaround issue where the ticket may have an empty label
		if label == "" {
			continue
		}
		issueLabel, _, _, err := ln.client.GetIssueLabel(ctx, label)
		if err != nil {
			return nil, fmt.Errorf("baton-linear: failed to get issue label: %w", err)
		}
		if issueLabel == nil {
			issueLabel, _, _, err = ln.client.CreateIssueLabel(ctx, label)
			if err != nil {
				return nil, fmt.Errorf("baton-linear: failed to create issue label: %w", err)
			}
		}
		labelIDs = append(labelIDs, issueLabel.ID)
	}

	payload.LabelIDs = labelIDs
	return &payload, nil
}

func (ln *Linear) CreateTicket(ctx context.Context, ticket *v2.Ticket, schema *v2.TicketSchema) (*v2.Ticket, annotations.Annotations, error) {
	l := ctxzap.Extract(ctx)
	l.Info("Creating ticket", zap.Any("ticket", ticket))

	payload, err := ln.createIssuePayloadFromTicket(ctx, ticket, schema)
	if err != nil {
		return nil, nil, err
	}

	issue, err := ln.client.CreateIssue(ctx, *payload)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to create issue: %w", err)
	}

	ticketResp := ticketFromIssue(issue)
	return ticketResp, nil, nil
}

func (ln *Linear) GetTicketSchema(ctx context.Context, schemaID string) (*v2.TicketSchema, annotations.Annotations, error) {
	teams, _, _, _, err := ln.client.ListTeamWorkflowStates(ctx, linear.GetTeamsVars{TeamIDs: []string{schemaID}, First: 2})
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to list team workflow states: %w", err)
	}
	if len(teams) != 1 {
		return nil, nil, fmt.Errorf("baton-linear: expected 1 team, got %d", len(teams))
	}
	fields, _, _, _, err := ln.client.ListIssueFields(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("baton-linear: failed to list issue fields: %w", err)
	}
	return ticketSchemaFromTeam(ctx, teams[0], fields), nil, nil
}

// ListTicketSchemas lists all the ticket schemas for Linear Issues.
//
// Linear Issues currently vary only by Workflow State per Team.
func (ln *Linear) ListTicketSchemas(ctx context.Context, p *pagination.Token) ([]*v2.TicketSchema, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	bag, err := parsePageToken(p.Token, &v2.ResourceId{ResourceType: resourceTypeTeam.Id})
	if err != nil {
		return nil, "", nil, err
	}

	teams, nextToken, _, rlData, err := ln.client.ListTeamWorkflowStates(ctx, linear.GetTeamsVars{TeamIDs: ln.ticketSchemaTeamIDs, After: bag.PageToken(), First: resourcePageSize})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-linear: failed to list teams: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	fields, _, _, rlData, err := ln.client.ListIssueFields(ctx)
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("baton-linear: failed to list issue fields: %w", err)
	}

	var ret []*v2.TicketSchema
	for _, team := range teams {
		ret = append(ret, ticketSchemaFromTeam(ctx, team, fields))
	}

	return ret, pageToken, annotations, nil
}

func ticketSchemaFromTeam(ctx context.Context, team linear.Team, fields []linear.IssueField) *v2.TicketSchema {
	statuses := ticketStatusesFromTeam(team)
	customFields := getCustomFields(ctx, fields, statuses)

	return &v2.TicketSchema{
		Id:           team.ID,
		DisplayName:  team.Name,
		Statuses:     statuses,
		CustomFields: customFields,
	}
}

func ticketStatusesFromTeam(team linear.Team) []*v2.TicketStatus {
	// Sort the statuses by position in Linear
	sort.Slice(team.States.Nodes, func(i, j int) bool {
		return team.States.Nodes[i].Position < team.States.Nodes[j].Position
	})

	var statuses []*v2.TicketStatus
	for _, state := range team.States.Nodes {
		statuses = append(statuses, &v2.TicketStatus{Id: state.ID, DisplayName: state.Name})
	}
	return statuses
}

func getCustomFields(ctx context.Context, fields []linear.IssueField, statuses []*v2.TicketStatus) map[string]*v2.TicketCustomField {
	fieldMap := make(map[string]*v2.TicketCustomField)
	for _, f := range fields {
		if cfSchema, ok := getCustomFieldSchema(f, statuses); ok {
			fieldMap[f.Name] = cfSchema
		}
		// TODO(johnallers): else, Log that the field is not supported
	}

	return fieldMap
}

func getCustomFieldSchema(field linear.IssueField, statuses []*v2.TicketStatus) (*v2.TicketCustomField, bool) {
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
	case "stateId":
		statusOptions := make([]*v2.TicketCustomFieldObjectValue, len(statuses))
		for i, status := range statuses {
			statusOptions[i] = &v2.TicketCustomFieldObjectValue{
				Id:          status.Id,
				DisplayName: "status",
			}
		}
		return sdkTicket.PickObjectValueFieldSchema(field.Name, field.Name, false, statusOptions), true
	case "assigneeId", "cycleId", "projectId", "projectMilestoneId", "subscriberIds", "templateId":
		switch field.Type.Kind {
		case "SCALAR":
			switch field.Type.Name {
			case "String":
				return sdkTicket.StringFieldSchema(field.Name, field.Name, false), true
			case "Boolean":
				return sdkTicket.BoolFieldSchema(field.Name, field.Name, false), true
			case "Float", "Int":
				// At this time, NumberFieldSchema only supports Float and does not render properly in the UI.
				// These need to be converted from String before being sent to Linear.
				return sdkTicket.StringFieldSchema(field.Name, field.Name, false), true
			case "JSON", "DateTime", "TimelessDate":
				return nil, false
			}
		case "ENUM":
			enums := make([]*v2.TicketCustomFieldObjectValue, len(field.Type.EnumValues))
			for i, v := range field.Type.EnumValues {
				enums[i] = &v2.TicketCustomFieldObjectValue{
					Id:          v.Name,
					DisplayName: v.Name,
				}
			}
			return sdkTicket.PickObjectValueFieldSchema(field.Name, field.Name, false, enums), true
		case "LIST":
			var ofType *linear.IssueFieldType
			if field.Type.OfType.Kind == "NON_NULL" {
				ofType = field.Type.OfType.OfType
			} else {
				ofType = field.Type.OfType
			}
			if ofType.Kind == "SCALAR" && ofType.Name == "String" {
				return sdkTicket.StringsFieldSchema(field.Name, field.Name, false), true
			}
		case "NON_NULL":
			if field.Type.OfType != nil {
				req, ok := getCustomFieldSchema(linear.IssueField{
					Name:        field.Name,
					Description: field.Description,
					Type:        *field.Type.OfType,
				}, statuses)
				if ok {
					req.Required = true
					return req, true
				}
			}
		}
	}

	return nil, false
}

func (ln *Linear) BulkCreateTickets(ctx context.Context, request *v2.TicketsServiceBulkCreateTicketsRequest) (*v2.TicketsServiceBulkCreateTicketsResponse, error) {
	tickets := make([]*v2.TicketsServiceCreateTicketResponse, 0)
	for _, tr := range request.GetTicketRequests() {
		reqBody := tr.GetRequest()
		ticketBody := &v2.Ticket{
			DisplayName:  reqBody.GetDisplayName(),
			Description:  reqBody.GetDescription(),
			Status:       reqBody.GetStatus(),
			Labels:       reqBody.GetLabels(),
			CustomFields: reqBody.GetCustomFields(),
			RequestedFor: reqBody.GetRequestedFor(),
		}
		ticket, annos, err := ln.CreateTicket(ctx, ticketBody, tr.GetSchema())
		// So we can track the external ticket ref annotation
		annos.Merge(tr.GetAnnotations()...)
		var ticketResp *v2.TicketsServiceCreateTicketResponse
		if err != nil {
			ticketResp = &v2.TicketsServiceCreateTicketResponse{Ticket: nil, Annotations: annos, Error: err.Error()}
		} else {
			ticketResp = &v2.TicketsServiceCreateTicketResponse{Ticket: ticket, Annotations: annos}
		}
		tickets = append(tickets, ticketResp)
	}
	return &v2.TicketsServiceBulkCreateTicketsResponse{Tickets: tickets}, nil
}

func (ln *Linear) BulkGetTickets(ctx context.Context, request *v2.TicketsServiceBulkGetTicketsRequest) (*v2.TicketsServiceBulkGetTicketsResponse, error) {
	tickets := make([]*v2.TicketsServiceGetTicketResponse, 0)
	for _, ticketReq := range request.GetTicketRequests() {
		ticket, annos, err := ln.GetTicket(ctx, ticketReq.GetId())
		// So we can track the external ticket ref annotation
		annos.Merge(ticketReq.GetAnnotations()...)
		var ticketResp *v2.TicketsServiceGetTicketResponse
		if err != nil {
			ticketResp = &v2.TicketsServiceGetTicketResponse{Ticket: ticket, Annotations: annos, Error: err.Error()}
		} else {
			ticketResp = &v2.TicketsServiceGetTicketResponse{Ticket: ticket, Annotations: annos}
		}
		tickets = append(tickets, ticketResp)
	}
	return &v2.TicketsServiceBulkGetTicketsResponse{Tickets: tickets}, nil
}
