# authentik-mcp-server

Workflow-centric MCP server for the [Authentik](https://goauthentik.io/) Identity Provider, written in Go with stdio transport.

Exposes Authentik identity operations as callable tools in [Claude Code](https://docs.anthropic.com/en/docs/claude-code/overview), enabling you to query user access, audit security events, and inspect your IAM posture directly from the CLI.

## Tools

| Tool | Access | Purpose |
|------|--------|---------|
| `summarize_user_access` | Read | Summarize a user's identity state and access |
| `audit_recent_security_events` | Read | List recent security-relevant events |
| `list_groups` | Read | List and filter groups |
| `list_applications` | Read | List and filter applications |
| `check_policy` | Read | Approximate a user's application access posture from policy bindings |
| `create_user` | Write-gated | Create a user |
| `set_user_password` | Write-gated | Set a user's password |
| `manage_user_group` | Write-gated | Add or remove a user from a group |
| `manage_outpost` | Write-gated | List outposts or refresh an outpost configuration |

Write-gated tools are registered only when `AUTHENTIK_ENABLE_WRITE=true`.

### `summarize_user_access`

Returns a structured JSON summary of a user's identity, group memberships, recent activity, and accessible applications.

**Input:** `username` (string, required)

**Output:**
```json
{
  "username": "<username>",
  "email": "<user@example.com>",
  "is_active": true,
  "last_login": "2026-01-15T09:00:00Z",
  "groups": ["admins", "vpn-users"],
  "recent_events": [
    {"action": "login", "datetime": "2026-01-15T09:00:00Z", "client_ip": "192.0.2.10"}
  ],
  "accessible_apps": ["Grafana", "Vault"]
}
```

### `audit_recent_security_events`

Returns a markdown table of recent security-relevant events: `login_failed`, `policy_denied`, and `secret_view`. Events are fetched in parallel across all three action types, merged, and sorted by timestamp descending.

**Input:** `limit` (int, optional, default 20)

**Output:**
```
| DateTime | Action | Username | ClientIP |
|----------|--------|----------|----------|
| 2026-01-15T09:00:00Z | login_failed | &lt;username&gt; | 192.0.2.10 |
```

## Quick Start

### Prerequisites

- Go 1.24+
- Authentik instance with an API token ([Settings → System → Tokens](https://docs.goauthentik.io/docs/sys-mgmt/tokens))

### Build

```bash
make build
# binary at bin/authentik-mcp
```

### Run

```bash
export AUTHENTIK_URL=https://your-authentik-instance:9443
export AUTHENTIK_TOKEN=your-api-token
./bin/authentik-mcp
```

### Smoke test

```bash
make smoke-test
# OK: connected to Authentik at https://your-authentik-instance:9443
```

### Docker

```bash
make docker-build
docker run -e AUTHENTIK_URL=... -e AUTHENTIK_TOKEN=... authentik-mcp:latest
```

## Claude Code Integration

Add to `~/.claude.json`:

```json
{
  "mcpServers": {
    "authentik": {
      "command": "/path/to/bin/authentik-mcp",
      "env": {
        "AUTHENTIK_URL": "https://your-authentik-instance:9443",
        "AUTHENTIK_TOKEN": "your-api-token"
      }
    }
  }
}
```

Restart Claude Code, then use the tools directly in conversation:

```
> use the authentik mcp to audit recent security events
> summarize access for user <username>
```

## Configuration

| Env var | Required | Default | Description |
|---------|----------|---------|-------------|
| `AUTHENTIK_URL` | ✅ | — | Base URL of your Authentik instance |
| `AUTHENTIK_TOKEN` | ✅ | — | API token (Settings → System → Tokens) |
| `AUTHENTIK_TLS_SKIP_VERIFY` | — | `false` | Set to `true` to disable TLS verification (default: verify — use for self-signed certs) |
| `AUTHENTIK_ENABLE_WRITE` | — | `false` | The server is read-only by default; set to `true` to register and expose write tools |

## Architecture

- **Transport:** stdio (MCP standard, works with Claude Code and any MCP-compatible client)
- **SDK:** [mark3labs/mcp-go v0.32.0](https://github.com/mark3labs/mcp-go)
- **HTTP client:** Hand-crafted over the Authentik REST API (the official oapi-codegen generated client had type compatibility issues with the 3.2MB schema — see `oapi-codegen.yaml` retained for future use)
- **Retry logic:** 3 attempts, 500ms backoff on 5xx or network errors
- **Auth:** `Authorization: Bearer <token>` injected via transport middleware

### API endpoints used

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/v3/core/users/` | User search |
| GET | `/api/v3/core/users/{id}/` | User details |
| GET | `/api/v3/core/groups/?member_by_pk={id}` | Group memberships |
| GET | `/api/v3/events/events/?user_pk={id}` | Per-user events |
| GET | `/api/v3/events/events/?action={type}` | Events by action type |
| GET | `/api/v3/core/applications/` | Application list |
| GET | `/api/v3/policies/bindings/?target={application_uuid}` | Application policy bindings |
| GET | `/api/v3/outposts/instances/` | Outpost list |
| PATCH | `/api/v3/outposts/instances/{outpost_uuid}/` | Trigger outpost configuration re-sync |
| GET | `/api/v3/root/config/` | Instance capabilities (smoke test) |

## Roadmap

- **Phase 3:** `list_groups`, `create_user`, `set_user_password`, `manage_user_stages`
- **Phase 4:** `list_applications`, `manage_outpost`, `check_policy`
- **Phase 5:** Full oapi-codegen client when schema compatibility improves
- **Phase 6:** Docker Compose sidecar deployment, Semaphore-managed CI

## Make targets

```
make build        # compile to bin/authentik-mcp
make run          # build + run
make smoke-test   # build + connectivity check
make docker-build # build Docker image
make lint         # go vet
make generate     # regenerate oapi client (requires schema at /tmp/authentik-schema.json)
```

## License

MIT
