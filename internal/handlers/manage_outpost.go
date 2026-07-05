package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterManageOutpost(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("manage_outpost",
		mcp.WithDescription("Lists Authentik outposts or refreshes a named outpost's configuration."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Operation: list or refresh")),
		mcp.WithString("name", mcp.Description("Exact outpost name; required for refresh")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := strings.TrimSpace(req.GetString("action", ""))
		if action != "list" && action != "refresh" {
			return mcp.NewToolResultError("action must be 'list' or 'refresh'"), nil
		}
		name := strings.TrimSpace(req.GetString("name", ""))
		if action == "refresh" && name == "" {
			return mcp.NewToolResultError("name argument is required for refresh"), nil
		}

		outposts, err := c.GetOutposts(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching outposts: %w", err)
		}
		if action == "list" {
			if len(outposts) == 0 {
				return mcp.NewToolResultText("No outposts found."), nil
			}
			var sb strings.Builder
			sb.WriteString("| Name | Type | Providers |\n|------|------|-----------|\n")
			for _, outpost := range outposts {
				fmt.Fprintf(&sb, "| %s | %s | %d |\n",
					sanitizeMD(outpost.Name), sanitizeMD(outpost.Type), len(outpost.Providers))
			}
			return mcp.NewToolResultText(sb.String()), nil
		}

		var outpost *authentik.Outpost
		for i := range outposts {
			if outposts[i].Name == name {
				outpost = &outposts[i]
				break
			}
		}
		if outpost == nil {
			return mcp.NewToolResultError(fmt.Sprintf("outpost not found: %s", name)), nil
		}
		slog.Info("manage_outpost refresh called", "outpost", name)
		if err := c.RefreshOutpost(ctx, *outpost); err != nil {
			return nil, fmt.Errorf("refreshing outpost: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Refreshed outpost %s", name)), nil
	})
}
