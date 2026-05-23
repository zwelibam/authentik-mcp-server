# authentik-mcp-server

Workflow-centric MCP server for the Authentik Identity Provider.

## Quick start

export AUTHENTIK_URL=https://your-authentik-instance:9443
export AUTHENTIK_TOKEN=your-api-token
make generate
make build
make smoke-test

## Tools

- summarize_user_access
- audit_recent_security_events
