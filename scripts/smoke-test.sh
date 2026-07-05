#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
server_source="$repo_root/internal/server/server.go"
binary="$repo_root/bin/authentik-mcp"

# Registration counts are derived from server.go so this test remains valid across
# branches. Register calls before the write feature gate are read tools; calls
# between that gate and its else clause are the write-only delta.
read_count="$(awk '
  /if os\.Getenv\("AUTHENTIK_ENABLE_WRITE"\)/ { in_write = 1; next }
  in_write && /^[[:space:]]*} else {/ { in_write = 0; write_done = 1; next }
  !write_done && /handlers\.Register[A-Za-z0-9_]*\(s, c\)/ {
    if (in_write) write_count++; else read_count++
  }
  END { print read_count + 0 }
' "$server_source")"
write_count="$(awk '
  /if os\.Getenv\("AUTHENTIK_ENABLE_WRITE"\)/ { in_write = 1; next }
  in_write && /^[[:space:]]*} else {/ { in_write = 0; next }
  in_write && /handlers\.Register[A-Za-z0-9_]*\(s, c\)/ { write_count++ }
  END { print write_count + 0 }
' "$server_source")"

if (( read_count == 0 )); then
  echo "smoke test failed: found no read-tool registrations in $server_source" >&2
  exit 1
fi

mkdir -p "$repo_root/bin"
(
  cd "$repo_root"
  go build -o bin/authentik-mcp ./cmd/authentik-mcp
)

request_input='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke-test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'

check_tool_count() {
  local expected="$1"
  local write_enabled="$2"
  local output

  output="$(printf '%s\n' "$request_input" | AUTHENTIK_URL=https://dummy.invalid AUTHENTIK_TOKEN=dummy AUTHENTIK_ENABLE_WRITE="$write_enabled" "$binary")"

  if ! printf '%s\n' "$output" | python3 -c '
import json
import sys

expected = int(sys.argv[1])
responses = [json.loads(line) for line in sys.stdin if line.strip()]
response = next((item for item in responses if item.get("id") == 2), None)
if response is None:
    print("smoke test failed: tools/list response was not returned", file=sys.stderr)
    sys.exit(1)
if "error" in response:
    print("smoke test failed: tools/list returned an error: {}".format(response["error"]), file=sys.stderr)
    sys.exit(1)
actual = len(response.get("result", {}).get("tools", []))
if actual != expected:
    print(f"smoke test failed: expected {expected} tools, got {actual}", file=sys.stderr)
    sys.exit(1)
' "$expected"; then
    return 1
  fi
}

check_tool_count "$read_count" false
check_tool_count "$((read_count + write_count))" true

echo "smoke test passed: $read_count read tools, $write_count write tools"
