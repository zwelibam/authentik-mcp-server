package authentik

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestGuardWrite(t *testing.T) {
	tests := []struct {
		name        string
		users       []string
		groups      []string
		protectedU  string
		protectedG  string
		allow       string
		wantErr     bool
		wantStrings []string
		notStrings  []string
	}{
		{
			name:       "no hits",
			users:      []string{"random"},
			groups:     []string{"developers"},
			protectedU: "blockeduser",
			protectedG: "admins",
		},
		{
			name:       "protected user only",
			users:      []string{"blockeduser"},
			groups:     []string{"developers"},
			protectedU: "blockeduser",
			protectedG: "admins",
			wantErr:    true,
			wantStrings: []string{
				`user "blockeduser"`,
				"AUTHENTIK_PROTECTED_USERS",
				"AUTHENTIK_PROTECTED_GROUPS",
				"AUTHENTIK_ALLOW_PROTECTED_WRITES",
			},
		},
		{
			name:       "protected group only",
			users:      []string{"random"},
			groups:     []string{"admins"},
			protectedU: "blockeduser",
			protectedG: "admins",
			wantErr:    true,
			wantStrings: []string{
				`group "admins"`,
				"AUTHENTIK_PROTECTED_USERS",
				"AUTHENTIK_PROTECTED_GROUPS",
				"AUTHENTIK_ALLOW_PROTECTED_WRITES",
			},
		},
		{
			name:       "protected user and group",
			users:      []string{"blockeduser"},
			groups:     []string{"admins"},
			protectedU: "blockeduser",
			protectedG: "admins",
			wantErr:    true,
			wantStrings: []string{
				`user "blockeduser"`,
				`group "admins"`,
			},
		},
		{
			name:       "bypass enabled",
			users:      []string{"blockeduser"},
			groups:     []string{"admins"},
			protectedU: "blockeduser",
			protectedG: "admins",
			allow:      "true",
		},
		{
			name:       "mixed lists name only protected hits",
			users:      []string{"blockeduser", "random"},
			groups:     []string{"developers", "admins"},
			protectedU: "blockeduser",
			protectedG: "admins",
			wantErr:    true,
			wantStrings: []string{
				`user "blockeduser"`,
				`group "admins"`,
			},
			notStrings: []string{"random", "developers"},
		},
		{
			name:       "uppercase true does not bypass",
			users:      []string{"blockeduser"},
			protectedU: "blockeduser",
			allow:      "True",
			wantErr:    true,
			wantStrings: []string{
				`user "blockeduser"`,
			},
		},
		{
			name:       "one does not bypass",
			groups:     []string{"admins"},
			protectedG: "admins",
			allow:      "1",
			wantErr:    true,
			wantStrings: []string{
				`group "admins"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTHENTIK_PROTECTED_USERS", tt.protectedU)
			t.Setenv("AUTHENTIK_PROTECTED_GROUPS", tt.protectedG)
			t.Setenv("AUTHENTIK_ALLOW_PROTECTED_WRITES", tt.allow)

			err := guardWrite("test op", tt.users, tt.groups)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("guardWrite() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, ErrProtectedObject) {
				t.Fatalf("guardWrite() error = %v, want ErrProtectedObject", err)
			}
			msg := err.Error()
			for _, want := range tt.wantStrings {
				if !strings.Contains(msg, want) {
					t.Fatalf("guardWrite() error = %q, want to contain %q", msg, want)
				}
			}
			for _, notWant := range tt.notStrings {
				if strings.Contains(msg, notWant) {
					t.Fatalf("guardWrite() error = %q, want not to contain %q", msg, notWant)
				}
			}
		})
	}
}

func TestClientMutationsGuardBeforeHTTP(t *testing.T) {
	tests := []struct {
		name       string
		protectedU string
		protectedG string
		call       func(context.Context, *Client) error
	}{
		{
			name: "set user password protects default akadmin",
			call: func(ctx context.Context, c *Client) error {
				return c.SetUserPassword(ctx, User{Username: "akadmin", PK: 1}, "somepassword123")
			},
		},
		{
			name:       "add user to protected group",
			protectedG: "admins",
			call: func(ctx context.Context, c *Client) error {
				return c.AddUserToGroup(ctx, Group{PK: "g1", Name: "admins"}, User{Username: "someone", PK: 2})
			},
		},
		{
			name:       "create protected user",
			protectedU: "blockeduser",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.CreateUser(ctx, CreateUserRequest{
					Username: "blockeduser",
					Name:     "Blocked User",
					Email:    "blocked@example.com",
					IsActive: true,
				}, nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTHENTIK_PROTECTED_USERS", tt.protectedU)
			t.Setenv("AUTHENTIK_PROTECTED_GROUPS", tt.protectedG)
			t.Setenv("AUTHENTIK_ALLOW_PROTECTED_WRITES", "")

			requests := 0
			defer func() {
				if r := recover(); r != nil {
					if strings.Contains(fmt.Sprint(r), "httptest: failed to listen on a port") {
						t.Skipf("httptest server unavailable in this environment: %v", r)
					}
					panic(r)
				}
			}()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{}`)
			}))
			defer server.Close()

			c := &Client{http: server.Client(), baseURL: server.URL, token: "test"}
			err := tt.call(context.Background(), c)
			if !errors.Is(err, ErrProtectedObject) {
				t.Fatalf("client method error = %v, want ErrProtectedObject", err)
			}
			if requests != 0 {
				t.Fatalf("request count = %d, want 0", requests)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
