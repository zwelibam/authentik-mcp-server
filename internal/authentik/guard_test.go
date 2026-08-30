package authentik

import "testing"

func TestIsProtectedUser(t *testing.T) {
	tests := []struct {
		name     string
		envValue *string
		username string
		want     bool
	}{
		{name: "unset protects akadmin", username: "akadmin", want: true},
		{name: "unset does not protect random", username: "random", want: false},
		{name: "custom merges with akadmin", envValue: strPtr("bob"), username: "akadmin", want: true},
		{name: "custom protects custom user", envValue: strPtr("bob"), username: "bob", want: true},
		{name: "custom mixed case and whitespace matches akadmin", envValue: strPtr("  AkAdmin , Bob "), username: "akadmin", want: true},
		{name: "custom mixed case and whitespace matches bob case-insensitively", envValue: strPtr("  AkAdmin , Bob "), username: "BOB", want: true},
		{name: "empty behaves like unset for akadmin", envValue: strPtr(""), username: "akadmin", want: true},
		{name: "empty behaves like unset for random", envValue: strPtr(""), username: "random", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != nil {
				t.Setenv("AUTHENTIK_PROTECTED_USERS", *tt.envValue)
			} else {
				t.Setenv("AUTHENTIK_PROTECTED_USERS", "")
			}
			if got := IsProtectedUser(tt.username); got != tt.want {
				t.Fatalf("IsProtectedUser(%q) = %v, want %v", tt.username, got, tt.want)
			}
		})
	}
}

func TestIsProtectedGroup(t *testing.T) {
	tests := []struct {
		name     string
		envValue *string
		group    string
		want     bool
	}{
		{name: "unset protects nothing", group: "admins", want: false},
		{name: "configured protects admins", envValue: strPtr("admins,superusers"), group: "admins", want: true},
		{name: "configured protects superusers case-insensitively", envValue: strPtr("admins,superusers"), group: "SUPERUSERS", want: true},
		{name: "configured does not protect random", envValue: strPtr("admins,superusers"), group: "random", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != nil {
				t.Setenv("AUTHENTIK_PROTECTED_GROUPS", *tt.envValue)
			} else {
				t.Setenv("AUTHENTIK_PROTECTED_GROUPS", "")
			}
			if got := IsProtectedGroup(tt.group); got != tt.want {
				t.Fatalf("IsProtectedGroup(%q) = %v, want %v", tt.group, got, tt.want)
			}
		})
	}
}

func TestAllowProtectedWrites(t *testing.T) {
	tests := []struct {
		name     string
		envValue *string
		want     bool
	}{
		{name: "unset false", want: false},
		{name: "true true", envValue: strPtr("true"), want: true},
		{name: "True false", envValue: strPtr("True"), want: false},
		{name: "1 false", envValue: strPtr("1"), want: false},
		{name: "yes false", envValue: strPtr("yes"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != nil {
				t.Setenv("AUTHENTIK_ALLOW_PROTECTED_WRITES", *tt.envValue)
			} else {
				t.Setenv("AUTHENTIK_ALLOW_PROTECTED_WRITES", "")
			}
			if got := AllowProtectedWrites(); got != tt.want {
				t.Fatalf("AllowProtectedWrites() = %v, want %v", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
