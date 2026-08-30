package authentik

import (
	"log/slog"
	"os"
	"strings"
)

var defaultProtectedUsers = []string{"akadmin"}

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
	slog.Warn("protected-object write allowed via AUTHENTIK_ALLOW_PROTECTED_WRITES", "target", target)
}
