package server

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
	"github.com/zwelibam/authentik-mcp-server/internal/handlers"
)

func Run(ctx context.Context) error {
	c, err := authentik.NewClientWrapper(ctx)
	if err != nil {
		return err
	}
	slog.Info("authentik client ready", "url", c.BaseURL())

	s := server.NewMCPServer("authentik-mcp-server", "0.2.0")

	// Phase 1+2 (read-only)
	handlers.RegisterSummarizeUserAccess(s, c)
	handlers.RegisterAuditRecentSecurityEvents(s, c)

	// Phase 3 (write operations)
	handlers.RegisterListGroups(s, c)
	handlers.RegisterCreateUser(s, c)
	handlers.RegisterSetUserPassword(s, c)
	handlers.RegisterManageUserGroup(s, c)

	slog.Info("starting MCP server", "transport", "stdio")
	return server.ServeStdio(s)
}
