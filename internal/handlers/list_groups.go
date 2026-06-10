package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterListGroups(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("list_groups",
		mcp.WithDescription("Returns all Authentik groups as a markdown table."),
		mcp.WithString("search", mcp.Description("Filter groups by name (case-insensitive contains match)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		groups, err := c.GetAllGroups(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching groups: %w", err)
		}

		search := strings.TrimSpace(req.GetString("search", ""))
		if search != "" {
			needle := strings.ToLower(search)
			filtered := make([]authentik.Group, 0, len(groups))
			for _, g := range groups {
				if strings.Contains(strings.ToLower(g.Name), needle) {
					filtered = append(filtered, g)
				}
			}
			groups = filtered
		}

		if len(groups) == 0 {
			return mcp.NewToolResultText("No groups found."), nil
		}

		var sb strings.Builder
		sb.WriteString("| Name | PK |\n|------|----|\n")
		for _, g := range groups {
			fmt.Fprintf(&sb, "| %s | %s |\n", sanitizeMD(g.Name), sanitizeMD(g.PK))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})
}
