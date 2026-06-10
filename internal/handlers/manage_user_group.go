package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterManageUserGroup(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("manage_user_group",
		mcp.WithDescription("Adds or removes a user from an Authentik group."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Operation: add or remove")),
		mcp.WithString("username", mcp.Required()),
		mcp.WithString("group", mcp.Required(), mcp.Description("Group name")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action := req.GetString("action", "")
		if action != "add" && action != "remove" {
			return mcp.NewToolResultError("action must be 'add' or 'remove'"), nil
		}

		username := req.GetString("username", "")
		if username == "" {
			return mcp.NewToolResultError("username argument is required"), nil
		}

		groupName := req.GetString("group", "")
		if groupName == "" {
			return mcp.NewToolResultError("group argument is required"), nil
		}

		slog.Info("manage_user_group called", "action", action, "username", username, "group", groupName)

		users, err := c.GetUsers(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("fetching user: %w", err)
		}
		var user *authentik.User
		for i := range users {
			if users[i].Username == username {
				user = &users[i]
				break
			}
		}
		if user == nil {
			return mcp.NewToolResultError(fmt.Sprintf("user not found: %s", username)), nil
		}

		group, err := c.GetGroupByName(ctx, groupName)
		if err != nil {
			return nil, fmt.Errorf("fetching group: %w", err)
		}
		if group == nil {
			return mcp.NewToolResultError(fmt.Sprintf("group not found: %s", groupName)), nil
		}

		preposition := "to"
		if action == "add" {
			err = c.AddUserToGroup(ctx, group.PK, user.PK)
		} else {
			preposition = "from"
			err = c.RemoveUserFromGroup(ctx, group.PK, user.PK)
		}
		if err != nil {
			return nil, fmt.Errorf("%sing user group: %w", action, err)
		}

		return mcp.NewToolResultText(fmt.Sprintf("%sed user %s %s group %s", action, username, preposition, groupName)), nil
	})
}
