package server

import (
	"context"
	"log/slog"
	"os"

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
	handlers.RegisterListGroups(s, c)

	// Phase 3 (write operations)
	if os.Getenv("AUTHENTIK_ENABLE_WRITE") == "true" {
		slog.Warn("write operations are active")
		handlers.RegisterCreateUser(s, c)
		handlers.RegisterSetUserPassword(s, c)
		handlers.RegisterManageUserGroup(s, c)
	} else {
		slog.Info("write tools are disabled; set AUTHENTIK_ENABLE_WRITE=true to enable them")
	}

	slog.Info("starting MCP server", "transport", "stdio")
	return server.ServeStdio(s)
}
