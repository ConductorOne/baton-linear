package linear

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
