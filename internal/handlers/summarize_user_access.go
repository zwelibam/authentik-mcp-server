package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterSummarizeUserAccess(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("summarize_user_access",
		mcp.WithDescription("Returns a comprehensive summary of a users identity state: groups, authorized applications, and recent login events."),
		mcp.WithString("username", mcp.Required(), mcp.Description("The Authentik username to summarize")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		username := req.GetString("username", "")
		if username == "" {
			return mcp.NewToolResultError("username argument is required"), nil
		}
		slog.Info("summarize_user_access called", "username", username)

		users, err := c.GetUsers(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("fetching user: %w", err)
		}
		var matched *authentik.User
		for i := range users {
			if users[i].Username == username {
				matched = &users[i]
				break
			}
		}
		if matched == nil {
			return mcp.NewToolResultError(fmt.Sprintf("user %q not found", username)), nil
		}
		user := *matched

		groups, err := c.GetGroupsForUser(ctx, user.PK)
		if err != nil {
			return nil, fmt.Errorf("fetching groups: %w", err)
		}
		groupNames := make([]string, len(groups))
		for i, g := range groups {
			groupNames[i] = g.Name
		}

		events, err := c.GetUserEvents(ctx, user.PK, 5)
		if err != nil {
			return nil, fmt.Errorf("fetching events: %w", err)
		}
		type eventSummary struct {
			Action   string `json:"action"`
			DateTime string `json:"datetime"`
			ClientIP string `json:"client_ip"`
		}
		recentEvents := make([]eventSummary, len(events))
		for i, e := range events {
			recentEvents[i] = eventSummary{Action: e.Action, DateTime: e.DateTime, ClientIP: e.ClientIP}
		}

		apps, err := c.GetApplications(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching applications: %w", err)
		}
		groupSet := make(map[string]bool)
		for _, g := range groupNames {
			groupSet[strings.ToLower(g)] = true
		}
		var accessibleApps []string
		for _, app := range apps {
			if groupSet[strings.ToLower(app.Name)] || groupSet[strings.ToLower(app.Slug)] {
				accessibleApps = append(accessibleApps, app.Name)
			}
		}
		sort.Strings(accessibleApps)

		result := map[string]any{
			"username":        user.Username,
			"email":           user.Email,
			"is_active":       user.IsActive,
			"last_login":      user.LastLogin,
			"groups":          groupNames,
			"recent_events":   recentEvents,
			"accessible_apps": accessibleApps,
		}
		b, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshaling result: %w", err)
		}
		return mcp.NewToolResultText(string(b)), nil
	})
}
