package connector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

func decodeJSON(t *testing.T, r *http.Request, out interface{}) error {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatalf("failed to decode request: %v", err)
	}
	return nil
}

func newTestUserBuilder(t *testing.T, handler http.HandlerFunc) *userResourceType {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := linear.NewClient(context.Background(), "test-api-key", server.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return userBuilder(client, false)
}

func TestUserCreateAccount_MissingEmail(t *testing.T) {
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called when email is missing")
	})
	_, _, _, err := ub.CreateAccount(context.Background(), &v2.AccountInfo{}, nil)
	if err == nil {
		t.Fatal("expected error for missing email")
	}
	if !strings.Contains(err.Error(), "email is required") {
		t.Errorf("expected 'email is required' error, got: %v", err)
	}
}

func TestUserCreateAccount_FromPrimaryEmail(t *testing.T) {
	var seenInput map[string]interface{}
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = decodeJSON(t, r, &req)
		vars := req["variables"].(map[string]interface{})
		seenInput = vars["input"].(map[string]interface{})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-abc"}}}}`))
	})

	info := &v2.AccountInfo{
		Emails: []*v2.AccountInfo_Email{
			{Address: "alt@example.com", IsPrimary: false},
			{Address: "primary@example.com", IsPrimary: true},
		},
	}
	resp, _, _, err := ub.CreateAccount(context.Background(), info, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	action, ok := resp.(*v2.CreateAccountResponse_ActionRequiredResult)
	if !ok {
		t.Fatalf("expected ActionRequiredResult, got %T", resp)
	}
	if !strings.Contains(action.Message, "inv-abc") {
		t.Errorf("message should contain invite ID, got: %s", action.Message)
	}
	if seenInput["email"] != "primary@example.com" {
		t.Errorf("expected primary email, got %v", seenInput["email"])
	}
	if _, has := seenInput["role"]; has {
		t.Errorf("role should not be sent when profile omits user_role, got %v", seenInput["role"])
	}
}

func TestUserCreateAccount_FallsBackToLogin(t *testing.T) {
	var seenEmail interface{}
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = decodeJSON(t, r, &req)
		vars := req["variables"].(map[string]interface{})
		seenEmail = vars["input"].(map[string]interface{})["email"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-1"}}}}`))
	})

	info := &v2.AccountInfo{Login: "login@example.com"}
	if _, _, _, err := ub.CreateAccount(context.Background(), info, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenEmail != "login@example.com" {
		t.Errorf("expected login fallback, got %v", seenEmail)
	}
}

func TestUserCreateAccount_RoleFromProfile(t *testing.T) {
	tests := []struct {
		profileRole string
		wantRole    interface{}
	}{
		{"admin", "admin"},
		{"  Admin  ", "admin"},
		{"guest", "guest"},
		{"user", "user"},
		{"owner", nil}, // owner is not allowed via invite — should be dropped
		{"junk", nil},
	}
	for _, tt := range tests {
		t.Run(tt.profileRole, func(t *testing.T) {
			var seenRole interface{}
			ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
				var req map[string]interface{}
				_ = decodeJSON(t, r, &req)
				vars := req["variables"].(map[string]interface{})
				seenRole = vars["input"].(map[string]interface{})["role"]
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-1"}}}}`))
			})

			profile, err := structpb.NewStruct(map[string]interface{}{
				userRoleProfileKey: tt.profileRole,
			})
			if err != nil {
				t.Fatalf("structpb: %v", err)
			}
			info := &v2.AccountInfo{
				Emails:  []*v2.AccountInfo_Email{{Address: "x@example.com", IsPrimary: true}},
				Profile: profile,
			}
			if _, _, _, err := ub.CreateAccount(context.Background(), info, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if seenRole != tt.wantRole {
				t.Errorf("role: want %v got %v", tt.wantRole, seenRole)
			}
		})
	}
}

func TestUserDelete_Success(t *testing.T) {
	var seenID interface{}
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = decodeJSON(t, r, &req)
		vars := req["variables"].(map[string]interface{})
		seenID = vars["id"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"userSuspend":{"success":true}}}`))
	})

	_, err := ub.Delete(context.Background(), &v2.ResourceId{
		ResourceType: resourceTypeUser.Id,
		Resource:     "user-xyz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenID != "user-xyz" {
		t.Errorf("user id: want user-xyz got %v", seenID)
	}
}

func TestUserDelete_WrongResourceType(t *testing.T) {
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called for wrong resource type")
	})
	_, err := ub.Delete(context.Background(), &v2.ResourceId{
		ResourceType: resourceTypeTeam.Id,
		Resource:     "team-1",
	})
	if err == nil {
		t.Fatal("expected error for non-user resource type")
	}
}

func TestUserDelete_SuccessFalse(t *testing.T) {
	ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"userSuspend":{"success":false}}}`))
	})
	_, err := ub.Delete(context.Background(), &v2.ResourceId{
		ResourceType: resourceTypeUser.Id,
		Resource:     "user-xyz",
	})
	if err == nil {
		t.Fatal("expected error when userSuspend returns success=false")
	}
}
