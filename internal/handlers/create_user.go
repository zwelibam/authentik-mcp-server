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

func RegisterCreateUser(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("create_user",
		mcp.WithDescription("Creates a new Authentik user account."),
		mcp.WithString("username", mcp.Required()),
		mcp.WithString("email", mcp.Required()),
		mcp.WithString("name", mcp.Description("Display name, defaults to username if empty")),
		mcp.WithBoolean("is_active", mcp.Description("Whether the account is active, defaults to true")),
		mcp.WithString("groups_csv", mcp.Description("Comma-separated group names to add the user to")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		username := strings.TrimSpace(req.GetString("username", ""))
		if username == "" {
			return mcp.NewToolResultError("username argument is required"), nil
		}

		email := strings.TrimSpace(req.GetString("email", ""))
		if !strings.Contains(email, "@") {
			return mcp.NewToolResultError("invalid email address"), nil
		}

		name := strings.TrimSpace(req.GetString("name", ""))
		if name == "" {
			name = username
		}

		isActive := req.GetBool("is_active", true)

		var groupNames []string
		groupsCSV := strings.TrimSpace(req.GetString("groups_csv", ""))
		if groupsCSV != "" {
			for _, part := range strings.Split(groupsCSV, ",") {
				groupName := strings.TrimSpace(part)
				if groupName != "" {
					groupNames = append(groupNames, groupName)
				}
			}
		}

		resolvedPKs := make([]string, 0, len(groupNames))
		for _, groupName := range groupNames {
			group, err := c.GetGroupByName(ctx, groupName)
			if err != nil {
				return nil, fmt.Errorf("fetching group %q: %w", groupName, err)
			}
			if group == nil {
				return mcp.NewToolResultError(fmt.Sprintf("group not found: %s", groupName)), nil
			}
			resolvedPKs = append(resolvedPKs, group.PK)
		}

		createReq := authentik.CreateUserRequest{
			Username: username,
			Name:     name,
			Email:    email,
			IsActive: isActive,
			Groups:   resolvedPKs,
		}
		slog.Info("create_user called", "username", username, "email", email)
		created, err := c.CreateUser(ctx, createReq)
		if err != nil {
			return nil, fmt.Errorf("creating user: %w", err)
		}

		response := fmt.Sprintf("Created user %s (PK: %d)", created.Username, created.PK)
		if len(groupNames) > 0 {
			response += "\nGroups: " + strings.Join(groupNames, ", ")
		}
		return mcp.NewToolResultText(response), nil
	})
}
