package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/zwelibam/authentik-mcp-server/internal/authentik"
)

func RegisterSetUserPassword(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("set_user_password",
		mcp.WithDescription("Sets the password for an Authentik user. Use for initial setup or password resets."),
		mcp.WithString("username", mcp.Required()),
		mcp.WithString("password", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		username := req.GetString("username", "")
		if username == "" {
			return mcp.NewToolResultError("username argument is required"), nil
		}

		password := req.GetString("password", "")
		slog.Info("set_user_password called", "username", username)
		if len(password) < 12 {
			return mcp.NewToolResultError("password must be at least 12 characters"), nil
		}

		users, err := c.GetUsers(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("fetching user: %w", err)
		}
		var foundUser *authentik.User
		for i := range users {
			if users[i].Username == username {
				foundUser = &users[i]
				break
			}
		}
		if foundUser == nil {
			return mcp.NewToolResultError(fmt.Sprintf("user not found: %s", username)), nil
		}

		if err := c.SetUserPassword(ctx, foundUser.PK, password); err != nil {
			return nil, fmt.Errorf("setting password: %w", err)
		}
		return mcp.NewToolResultText(fmt.Sprintf("Password updated for user %s", username)), nil
	})
}
