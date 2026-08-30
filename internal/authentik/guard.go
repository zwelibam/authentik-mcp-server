package authentik

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

var defaultProtectedUsers = []string{"akadmin"}

// ErrProtectedObject is returned by mutating Client methods when a target is
// on a protected denylist and AUTHENTIK_ALLOW_PROTECTED_WRITES != "true".
// Handlers map it to a tool error via errors.Is.
var ErrProtectedObject = errors.New("write to protected object refused")

func envSet(name string, defaults []string) map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return stringSet(defaults)
	}

	set := stringSet(strings.Split(raw, ","))
	if len(set) == 0 {
		return stringSet(defaults)
	}
	return set
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func IsProtectedUser(username string) bool {
	protectedUsers := envSet("AUTHENTIK_PROTECTED_USERS", nil)
	for user := range stringSet(defaultProtectedUsers) {
		protectedUsers[user] = struct{}{}
	}
	_, ok := protectedUsers[strings.ToLower(strings.TrimSpace(username))]
	return ok
}

func IsProtectedGroup(group string) bool {
	_, ok := envSet("AUTHENTIK_PROTECTED_GROUPS", nil)[strings.ToLower(strings.TrimSpace(group))]
	return ok
}

func AllowProtectedWrites() bool {
	return os.Getenv("AUTHENTIK_ALLOW_PROTECTED_WRITES") == "true"
}

func WarnProtectedBypass(target string) {
	slog.Error("protected-object write allowed via AUTHENTIK_ALLOW_PROTECTED_WRITES", "target", target)
}

// guardWrite enforces the protected denylists for a mutating operation.
// Behavior parity with the handler-level checks it replaces: refuse lists
// every protected principal and names the env vars; an enabled bypass logs
// each protected principal at error level exactly once and proceeds.
// Unexported: callers reach it through the mutating Client methods.
func guardWrite(op string, users, groups []string) error {
	var hitU, hitG []string
	for _, u := range users {
		if IsProtectedUser(u) {
			hitU = append(hitU, fmt.Sprintf("user %q", u))
		}
	}
	for _, g := range groups {
		if IsProtectedGroup(g) {
			hitG = append(hitG, fmt.Sprintf("group %q", g))
		}
	}
	if len(hitU) == 0 && len(hitG) == 0 {
		return nil
	}
	joined := strings.Join(append(hitU, hitG...), ", ")
	if !AllowProtectedWrites() {
		return fmt.Errorf("%w: %s %s (see AUTHENTIK_PROTECTED_USERS / AUTHENTIK_PROTECTED_GROUPS / AUTHENTIK_ALLOW_PROTECTED_WRITES)",
			ErrProtectedObject, op, joined)
	}
	for _, u := range users {
		if IsProtectedUser(u) {
			WarnProtectedBypass(u)
		}
	}
	for _, g := range groups {
		if IsProtectedGroup(g) {
			WarnProtectedBypass(g)
		}
	}
	return nil
}
