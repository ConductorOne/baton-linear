package main

import (
	"context"
	"fmt"
	"os"

	cfg "github.com/conductorone/baton-linear/pkg/config"
	"github.com/conductorone/baton-linear/pkg/connector"
	"github.com/conductorone/baton-sdk/pkg/cli"
	configschema "github.com/conductorone/baton-sdk/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/connectorrunner"
	"github.com/conductorone/baton-sdk/pkg/types"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var version = "dev"

func main() {
	ctx := context.Background()

	_, cmd, err := configschema.DefineConfigurationV2(
		ctx,
		"baton-linear",
		getConnector,
		cfg.Config,
		connectorrunner.WithDefaultCapabilitiesConnectorBuilder(&connector.Linear{}),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	cmd.Version = version

	err = cmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func getConnector(ctx context.Context, lc *cfg.Linear, runTimeOpts cli.RunTimeOpts) (types.ConnectorServer, error) {
	l := ctxzap.Extract(ctx)

	// Role grants are emitted from the user syncer, so the user builder needs
	// to know whether the customer's sync filter includes the role resource
	// type. WillSyncResourceType returns true when roles are explicitly
	// selected or when no filter is set at all (e.g. local CLI runs).
	connectorOpts := &cli.ConnectorOpts{SyncResourceTypeIDs: runTimeOpts.SyncResourceTypeIDs}
	syncRoles := connectorOpts.WillSyncResourceType(connector.RoleResourceTypeID)

	cb, err := connector.New(ctx, lc.ApiKey, lc.SkipProjects, lc.TicketSchemaTeamIdsFilter, lc.BaseUrl, syncRoles)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	opts := make([]connectorbuilder.Opt, 0)
	if lc.Ticketing {
		opts = append(opts, connectorbuilder.WithTicketingEnabled())
	}

	c, err := connectorbuilder.NewConnector(ctx, cb, opts...)
	if err != nil {
		l.Error("error creating connector", zap.Error(err))
		return nil, err
	}

	return c, nil
}
