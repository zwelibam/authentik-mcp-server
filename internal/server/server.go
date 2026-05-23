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

	s := server.NewMCPServer("authentik-mcp-server", "0.1.0")
	handlers.RegisterSummarizeUserAccess(s, c)
	handlers.RegisterAuditRecentSecurityEvents(s, c)

	slog.Info("starting MCP server", "transport", "stdio")
	return server.ServeStdio(s)
}
