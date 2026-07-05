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

func RegisterCheckPolicy(s *server.MCPServer, c *authentik.Client) {
	tool := mcp.NewTool("check_policy",
		mcp.WithDescription("Reports a read-only approximation of a user's access posture for an Authentik application."),
		mcp.WithString("username", mcp.Required()),
		mcp.WithString("application_slug", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		username := strings.TrimSpace(req.GetString("username", ""))
		if username == "" {
			return mcp.NewToolResultError("username argument is required"), nil
		}
		slug := strings.TrimSpace(req.GetString("application_slug", ""))
		if slug == "" {
			return mcp.NewToolResultError("application_slug argument is required"), nil
		}

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

		applications, err := c.GetApplications(ctx)
		if err != nil {
			return nil, fmt.Errorf("fetching applications: %w", err)
		}
		var application *authentik.Application
		for i := range applications {
			if applications[i].Slug == slug {
				application = &applications[i]
				break
			}
		}
		if application == nil {
			return mcp.NewToolResultError(fmt.Sprintf("application not found: %s", slug)), nil
		}

		groups, err := c.GetGroupsForUser(ctx, user.PK)
		if err != nil {
			return nil, fmt.Errorf("fetching user groups: %w", err)
		}
		bindings, err := c.GetPolicyBindings(ctx, application.PK)
		if err != nil {
			return nil, fmt.Errorf("fetching application policy bindings: %w", err)
		}
		sort.Slice(bindings, func(i, j int) bool { return bindings[i].Order < bindings[j].Order })

		groupNames := make(map[string]string, len(groups))
		for _, group := range groups {
			groupNames[group.PK] = group.Name
		}

		mode := application.PolicyEngineMode
		if mode == "" {
			mode = "unspecified"
		}
		var sb strings.Builder
		fmt.Fprintf(&sb, "User **%s** (active: %t) → application **%s** (engine mode: %s)\n\n",
			sanitizeMD(user.Username), user.IsActive, sanitizeMD(application.Name), sanitizeMD(mode))
		if len(bindings) == 0 {
			sb.WriteString("No policy bindings found. Authentik treats a target with no bindings as passing.\n")
		} else {
			sb.WriteString("| Order | Binding | Subject | Enabled | Negate | User/group match |\n")
			sb.WriteString("|------:|---------|---------|---------|--------|------------------|\n")
			for _, binding := range bindings {
				kind, subject, match := bindingDetails(binding, user.PK, groupNames)
				fmt.Fprintf(&sb, "| %d | %s | %s | %t | %t | %s |\n", binding.Order,
					sanitizeMD(kind), sanitizeMD(subject), binding.Enabled, binding.Negate, sanitizeMD(match))
			}
		}
		sb.WriteString("\nThis is a read-only approximation. Direct user and group bindings are matched locally; policy bindings require Authentik's policy engine and are not evaluated here.")
		return mcp.NewToolResultText(sb.String()), nil
	})
}

func bindingDetails(binding authentik.PolicyBinding, userPK int, groupNames map[string]string) (string, string, string) {
	if binding.Group != nil {
		name := *binding.Group
		groupName, matched := groupNames[*binding.Group]
		if matched {
			name = groupName
		}
		if binding.GroupObj != nil && binding.GroupObj.Name != "" {
			name = binding.GroupObj.Name
		}
		if matched {
			return "group", name, "yes"
		}
		return "group", name, "no"
	}
	if binding.User != nil {
		name := fmt.Sprint(*binding.User)
		if binding.UserObj != nil && binding.UserObj.Username != "" {
			name = binding.UserObj.Username
		}
		if *binding.User == userPK {
			return "user", name, "yes"
		}
		return "user", name, "no"
	}
	if binding.Policy != nil {
		name := *binding.Policy
		if binding.PolicyObj != nil && binding.PolicyObj.Name != "" {
			name = binding.PolicyObj.Name
		}
		return "policy", name, "not evaluated"
	}
	return "unknown", "", "not evaluated"
}
