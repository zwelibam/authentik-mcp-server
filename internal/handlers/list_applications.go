package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterListApplications(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("list_applications",
		mcp.WithDescription("Returns Authentik applications as a markdown table."),
		mcp.WithString("name", mcp.Description("Filter applications by name (case-insensitive contains match)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		applications, err := c.GetApplications(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching applications: %w", err)
		}

		name := strings.TrimSpace(req.GetString("name", ""))
		if name != "" {
			needle := strings.ToLower(name)
			filtered := make([]authentik.Application, 0, len(applications))
			for _, application := range applications {
				if strings.Contains(strings.ToLower(application.Name), needle) {
					filtered = append(filtered, application)
				}
			}
			applications = filtered
		}

		if len(applications) == 0 {
			return mcp.NewToolResultText("No applications found."), nil
		}

		var sb strings.Builder
		sb.WriteString("| Name | Slug | Provider |\n|------|------|----------|\n")
		for _, application := range applications {
			provider := ""
			if application.ProviderObj != nil {
				provider = application.ProviderObj.Name
			} else if application.Provider != nil {
				provider = fmt.Sprint(*application.Provider)
			}
			fmt.Fprintf(&sb, "| %s | %s | %s |\n",
				sanitizeMD(application.Name), sanitizeMD(application.Slug), sanitizeMD(provider))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})
}
