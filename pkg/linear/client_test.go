package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCreateOrganizationInvite(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		role           string
		serverResponse string
		statusCode     int
		wantErr        bool
		wantInviteID   string
	}{
		{
			name:  "successful invite creation",
			email: "test@example.com",
			role:  LinearRoleMember,
			serverResponse: `{
				"data": {
					"organizationInviteCreate": {
						"success": true,
						"organizationInvite": {
							"id": "invite-123",
							"email": "test@example.com"
						}
					}
				}
			}`,
			statusCode:   http.StatusOK,
			wantErr:      false,
			wantInviteID: "invite-123",
		},
		{
			name:  "invite creation with admin role",
			email: "admin@example.com",
			role:  LinearRoleAdmin,
			serverResponse: `{
				"data": {
					"organizationInviteCreate": {
						"success": true,
						"organizationInvite": {
							"id": "invite-456",
							"email": "admin@example.com"
						}
					}
				}
			}`,
			statusCode:   http.StatusOK,
			wantErr:      false,
			wantInviteID: "invite-456",
		},
		{
			name:  "invite creation with guest role",
			email: "guest@example.com",
			role:  LinearRoleGuest,
			serverResponse: `{
				"data": {
					"organizationInviteCreate": {
						"success": true,
						"organizationInvite": {
							"id": "invite-789",
							"email": "guest@example.com"
						}
					}
				}
			}`,
			statusCode:   http.StatusOK,
			wantErr:      false,
			wantInviteID: "invite-789",
		},
		{
			name:  "invite creation fails - success false",
			email: "test@example.com",
			role:  LinearRoleMember,
			serverResponse: `{
				"data": {
					"organizationInviteCreate": {
						"success": false,
						"organizationInvite": null
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:           "server error",
			email:          "test@example.com",
			role:           LinearRoleMember,
			serverResponse: `{"errors": [{"message": "Internal server error"}]}`,
			statusCode:     http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify request body contains expected data
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				variables, ok := reqBody["variables"].(map[string]interface{})
				if !ok {
					t.Error("request body missing variables")
				}

				input, ok := variables["input"].(map[string]interface{})
				if !ok {
					t.Error("variables missing input")
				}

				if input["email"] != tt.email {
					t.Errorf("expected email %s, got %s", tt.email, input["email"])
				}

				if input["role"] != tt.role {
					t.Errorf("expected role %s, got %s", tt.role, input["role"])
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client, err := NewClient(context.Background(), "test-api-key")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			// Override the API URL to use the test server
			client.apiUrl = mustParseURL(server.URL)

			invite, err := client.CreateOrganizationInvite(context.Background(), tt.email, tt.role)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if invite.ID != tt.wantInviteID {
				t.Errorf("expected invite ID %s, got %s", tt.wantInviteID, invite.ID)
			}

			if invite.Email != tt.email {
				t.Errorf("expected email %s, got %s", tt.email, invite.Email)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		admin          *bool
		active         *bool
		serverResponse string
		statusCode     int
		wantErr        bool
	}{
		{
			name:   "promote to admin",
			userID: "user-123",
			admin:  boolPtr(true),
			active: nil,
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "demote from admin",
			userID: "user-123",
			admin:  boolPtr(false),
			active: nil,
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "suspend user",
			userID: "user-456",
			admin:  nil,
			active: boolPtr(false),
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "reactivate user",
			userID: "user-456",
			admin:  nil,
			active: boolPtr(true),
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update both admin and active",
			userID: "user-789",
			admin:  boolPtr(true),
			active: boolPtr(true),
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "update fails - success false",
			userID: "user-123",
			admin:  boolPtr(true),
			active: nil,
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": false
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
		{
			name:           "server error",
			userID:         "user-123",
			admin:          boolPtr(true),
			active:         nil,
			serverResponse: `{"errors": [{"message": "Internal server error"}]}`,
			statusCode:     http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify request body contains expected data
				var reqBody map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}

				variables, ok := reqBody["variables"].(map[string]interface{})
				if !ok {
					t.Error("request body missing variables")
				}

				if variables["id"] != tt.userID {
					t.Errorf("expected user ID %s, got %s", tt.userID, variables["id"])
				}

				input, ok := variables["input"].(map[string]interface{})
				if !ok {
					t.Error("variables missing input")
				}

				// Verify admin field if set
				if tt.admin != nil {
					adminVal, exists := input["admin"]
					if !exists {
						t.Error("expected admin field in input")
					} else if adminVal != *tt.admin {
						t.Errorf("expected admin %v, got %v", *tt.admin, adminVal)
					}
				}

				// Verify active field if set
				if tt.active != nil {
					activeVal, exists := input["active"]
					if !exists {
						t.Error("expected active field in input")
					} else if activeVal != *tt.active {
						t.Errorf("expected active %v, got %v", *tt.active, activeVal)
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client, err := NewClient(context.Background(), "test-api-key")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			// Override the API URL to use the test server
			client.apiUrl = mustParseURL(server.URL)

			err = client.UpdateUser(context.Background(), tt.userID, tt.admin, tt.active)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
