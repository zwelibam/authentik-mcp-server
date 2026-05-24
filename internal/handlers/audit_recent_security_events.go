package handlers

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func sanitizeMD(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func RegisterAuditRecentSecurityEvents(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("audit_recent_security_events",
		mcp.WithDescription("Returns a markdown table of recent security events (login_failed, policy_denied, secret_view)."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of events to return (default 20)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int(req.GetFloat("limit", 20))
		if limit <= 0 {
			limit = 20
		}

		actions := []string{"login_failed", "policy_denied", "secret_view"}
		var allEvents []authentik.Event
		for _, action := range actions {
			evts, err := c.GetEventsByAction(ctx, action, limit)
			if err != nil {
				return nil, fmt.Errorf("fetching %s events: %w", action, err)
			}
			allEvents = append(allEvents, evts...)
		}

		// Sort by DateTime descending (ISO8601 strings sort lexicographically)
		sort.Slice(allEvents, func(i, j int) bool {
			return allEvents[i].DateTime > allEvents[j].DateTime
		})

		// Take top limit
		if len(allEvents) > limit {
			allEvents = allEvents[:limit]
		}

		if len(allEvents) == 0 {
			return mcp.NewToolResultText("No security events found."), nil
		}

		var sb strings.Builder
		sb.WriteString("| DateTime | Action | Username | ClientIP |\n")
		sb.WriteString("|----------|--------|----------|----------|\n")
		for _, e := range allEvents {
			username := ""
			if e.User != nil {
				username = e.User.Username
			}
			fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n",
				sanitizeMD(e.DateTime), sanitizeMD(e.Action), sanitizeMD(username), sanitizeMD(e.ClientIP))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})
}
