package connector

import (
	"context"
	"fmt"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
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

func (ln *Linear) ListTicketSchemas(ctx context.Context, pToken *pagination.Token) ([]*v2.TicketSchema, string, annotations.Annotations, error) {
	return nil, "", nil, fmt.Errorf("ListTicketSchemas not implemented")
}

func (ln *Linear) BulkCreateTickets(ctx context.Context, request *v2.TicketsServiceBulkCreateTicketsRequest) (*v2.TicketsServiceBulkCreateTicketsResponse, error) {
	return nil, fmt.Errorf("BulkCreateTickets not implemented")
}

func (ln *Linear) BulkGetTickets(ctx context.Context, request *v2.TicketsServiceBulkGetTicketsRequest) (*v2.TicketsServiceBulkGetTicketsResponse, error) {
	return nil, fmt.Errorf("BulkGetTickets not implemented")
}
