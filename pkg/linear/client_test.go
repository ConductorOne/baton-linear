package linear

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// decodeGraphQLRequest parses the GraphQL request body sent by the client and
// returns the input variable map (the "input" key under "variables").
func decodeGraphQLRequest(t *testing.T, body io.Reader) map[string]interface{} {
	t.Helper()
	var req map[string]interface{}
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	vars, ok := req["variables"].(map[string]interface{})
	if !ok {
		t.Fatalf("request body missing variables map: %v", req)
	}
	input, ok := vars["input"].(map[string]interface{})
	if !ok {
		// SuspendUser passes id at the top level, not in input.
		return vars
	}
	return input
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient(context.Background(), "test-api-key", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client, server
}

func TestGetUsersIncludesDisabled(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if !strings.Contains(req.Query, "includeDisabled: true") {
			t.Errorf("users query must include disabled users: %s", req.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"users":{"nodes":[{"id":"user-1","name":"Suspended User","active":false}],"pageInfo":{"hasNextPage":false}}}}`))
	})

	users, _, _, err := client.GetUsers(context.Background(), GetResourcesVars{First: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users: want 1 got %d", len(users))
	}
	if users[0].Active {
		t.Error("expected suspended user to be inactive")
	}
}

func TestGetOrganizationIncludesDisabledUsers(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if !strings.Contains(req.Query, "users(after: $usersAfter, first: $first, includeDisabled: true)") {
			t.Errorf("organization users query must include disabled users: %s", req.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"organization": {
					"id": "org-1",
					"users": {
						"nodes": [{"id": "user-1", "active": false}],
						"pageInfo": {"hasNextPage": false}
					},
					"teams": {"nodes": [], "pageInfo": {"hasNextPage": false}}
				}
			}
		}`))
	})

	organization, _, _, err := client.GetOrganization(context.Background(), PaginationVars{First: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(organization.Users.Nodes) != 1 {
		t.Fatalf("users: want 1 got %d", len(organization.Users.Nodes))
	}
}

func TestGetProjectIncludesDisabledMembers(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if !strings.Contains(req.Query, "members(after: $usersAfter, first: $first, includeDisabled: true)") {
			t.Errorf("project members query must include disabled users: %s", req.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"project": {
					"id": "project-1",
					"members": {
						"nodes": [{"id": "user-1", "active": false}],
						"pageInfo": {"hasNextPage": false}
					},
					"teams": {"nodes": [], "pageInfo": {"hasNextPage": false}}
				}
			}
		}`))
	})

	project, _, _, err := client.GetProject(context.Background(), GetProjectVars{ProjectId: "project-1", First: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(project.Members.Nodes) != 1 {
		t.Fatalf("members: want 1 got %d", len(project.Members.Nodes))
	}
}

func TestCreateOrganizationInvite(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		role         string
		teamIDs      []string
		response     string
		wantErr      bool
		wantInviteID string
		assertInput  func(t *testing.T, input map[string]interface{})
	}{
		{
			name:  "success with default role",
			email: "newuser@example.com",
			response: `{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-1"}}}}`,
			wantInviteID: "inv-1",
			assertInput: func(t *testing.T, input map[string]interface{}) {
				if input["email"] != "newuser@example.com" {
					t.Errorf("email: got %v", input["email"])
				}
				if _, has := input["role"]; has {
					t.Errorf("role should be omitted when caller passes empty string, got %v", input["role"])
				}
				if _, has := input["teamIds"]; has {
					t.Errorf("teamIds should be omitted when empty, got %v", input["teamIds"])
				}
			},
		},
		{
			name:  "success with admin role and teams",
			email: "admin@example.com",
			role:  "admin",
			teamIDs: []string{"team-a", "team-b"},
			response: `{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-2"}}}}`,
			wantInviteID: "inv-2",
			assertInput: func(t *testing.T, input map[string]interface{}) {
				if input["role"] != "admin" {
					t.Errorf("role: got %v", input["role"])
				}
				teams, ok := input["teamIds"].([]interface{})
				if !ok || len(teams) != 2 || teams[0] != "team-a" || teams[1] != "team-b" {
					t.Errorf("teamIds: got %v", input["teamIds"])
				}
			},
		},
		{
			name:     "API returns success=false",
			email:    "fail@example.com",
			response: `{"data":{"organizationInviteCreate":{"success":false,"organizationInvite":{"id":""}}}}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST, got %s", r.Method)
				}
				input := decodeGraphQLRequest(t, r.Body)
				if tt.assertInput != nil {
					tt.assertInput(t, input)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			})

			id, err := client.CreateOrganizationInvite(context.Background(), tt.email, tt.role, tt.teamIDs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantInviteID {
				t.Errorf("invite ID: want %q got %q", tt.wantInviteID, id)
			}
		})
	}
}

func TestSuspendUser(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		response    string
		wantSuccess bool
		wantErr     bool
	}{
		{
			name:        "success",
			userID:      "user-1",
			response:    `{"data":{"userSuspend":{"success":true}}}`,
			wantSuccess: true,
		},
		{
			name:        "API returns success=false",
			userID:      "user-2",
			response:    `{"data":{"userSuspend":{"success":false}}}`,
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				vars := decodeGraphQLRequest(t, r.Body)
				if vars["id"] != tt.userID {
					t.Errorf("id: want %q got %v", tt.userID, vars["id"])
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			})

			ok, err := client.SuspendUser(context.Background(), tt.userID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.wantSuccess {
				t.Errorf("success: want %v got %v", tt.wantSuccess, ok)
			}
		})
	}
}

func TestDoRequestBadRequest(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		header        string
		wantRateLimit bool
		wantMessage   string
	}{
		{
			name:          "rate limit error code is retryable",
			response:      `{"errors":[{"message":"Rate limit exceeded","extensions":{"code":"RATELIMITED"}}]}`,
			wantRateLimit: true,
		},
		{
			name:          "exhausted rate limit header is retryable",
			response:      `{"errors":[{"message":"Too many requests"}]}`,
			header:        "X-RateLimit-Requests-Remaining",
			wantRateLimit: true,
		},
		{
			name:        "invalid request is not retryable and keeps the message",
			response:    `{"errors":[{"message":"Argument Validation Error","extensions":{"code":"INVALID_INPUT"}}]}`,
			wantMessage: "Argument Validation Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tt.header != "" {
					w.Header().Set(tt.header, "0")
				}
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(tt.response))
			})

			_, err := client.CreateOrganizationInvite(context.Background(), "user@example.com", "", nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			isRateLimit := status.Code(err) == codes.Unavailable
			if isRateLimit != tt.wantRateLimit {
				t.Errorf("rate limited: want %v got %v (err: %v)", tt.wantRateLimit, isRateLimit, err)
			}
			if tt.wantMessage != "" && !strings.Contains(err.Error(), tt.wantMessage) {
				t.Errorf("error should contain %q, got: %v", tt.wantMessage, err)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantID   string
		wantErr  bool
	}{
		{
			name:     "found",
			response: `{"data":{"users":{"nodes":[{"id":"user-1","email":"user@example.com","active":true,"organization":{"id":"org-1"}}]}}}`,
			wantID:   "user-1",
		},
		{
			name:     "not found",
			response: `{"data":{"users":{"nodes":[]}}}`,
		},
		{
			name:     "graphql error",
			response: `{"data":null,"errors":[{"message":"Forbidden","extensions":{"code":"FORBIDDEN"}}]}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				vars := decodeGraphQLRequest(t, r.Body)
				if vars["email"] != "user@example.com" {
					t.Errorf("email: got %v", vars["email"])
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.response))
			})

			user, err := client.GetUserByEmail(context.Background(), "user@example.com")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got user %+v", user)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantID == "" {
				if user != nil {
					t.Errorf("expected nil user, got %+v", user)
				}
				return
			}
			if user == nil || user.ID != tt.wantID {
				t.Errorf("user: want ID %q got %+v", tt.wantID, user)
			}
		})
	}
}

func TestGetPendingOrganizationInvite(t *testing.T) {
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	past := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name    string
		pages   []string
		wantID  string
		wantErr bool
	}{
		{
			name: "pending invite on second page",
			pages: []string{
				`{"data":{"organizationInvites":{"nodes":[{"id":"inv-1","email":"other@example.com","expiresAt":"` + future + `"}],"pageInfo":{"hasNextPage":true,"endCursor":"c1"}}}}`,
				`{"data":{"organizationInvites":{"nodes":[{"id":"inv-2","email":"User@Example.com","expiresAt":"` + future + `"}],"pageInfo":{"hasNextPage":false}}}}`,
			},
			wantID: "inv-2",
		},
		{
			name: "accepted and expired invites are skipped",
			pages: []string{
				`{"data":{"organizationInvites":{"nodes":[{"id":"inv-1","email":"user@example.com","acceptedAt":"` + past + `"},{"id":"inv-2","email":"user@example.com","expiresAt":"` + past + `"}],"pageInfo":{"hasNextPage":false}}}}`,
			},
		},
		{
			name: "graphql error",
			pages: []string{
				`{"data":null,"errors":[{"message":"Forbidden","extensions":{"code":"FORBIDDEN"}}]}`,
			},
			wantErr: true,
		},
		{
			name: "cursor does not advance",
			pages: []string{
				`{"data":{"organizationInvites":{"nodes":[],"pageInfo":{"hasNextPage":true,"endCursor":""}}}}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				if calls >= len(tt.pages) {
					t.Errorf("unexpected request %d", calls+1)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.pages[calls]))
				calls++
			})

			invite, err := client.GetPendingOrganizationInvite(context.Background(), "user@example.com")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got invite %+v", invite)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantID == "" {
				if invite != nil {
					t.Errorf("expected nil invite, got %+v", invite)
				}
				return
			}
			if invite == nil || invite.ID != tt.wantID {
				t.Errorf("invite: want ID %q got %+v", tt.wantID, invite)
			}
		})
	}
}
