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
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestUserResource_Status(t *testing.T) {
	tests := []struct {
		name       string
		active     bool
		wantStatus v2.Status_ResourceStatus
	}{
		{
			name:       "active user is enabled",
			active:     true,
			wantStatus: v2.Status_RESOURCE_STATUS_ENABLED,
		},
		{
			name:       "suspended user is disabled",
			active:     false,
			wantStatus: v2.Status_RESOURCE_STATUS_DISABLED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := userResource(context.Background(), &linear.User{
				ID:     "user-1",
				Name:   "Test User",
				Email:  "test@example.com",
				Active: tt.active,
			}, &v2.ResourceId{ResourceType: resourceTypeOrg.Id, Resource: "org-1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := resource.GetStatus().GetStatus(); got != tt.wantStatus {
				t.Errorf("status: want %v got %v", tt.wantStatus, got)
			}
		})
	}
}

func TestUserResource_AccountType(t *testing.T) {
	tests := []struct {
		name        string
		app         bool
		wantService bool
	}{
		{
			name:        "human user is not a service account",
			app:         false,
			wantService: false,
		},
		{
			name:        "app user is a service account",
			app:         true,
			wantService: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource, err := userResource(context.Background(), &linear.User{
				ID:     "user-1",
				Name:   "Test User",
				Email:  "test@example.com",
				Active: true,
				App:    tt.app,
			}, &v2.ResourceId{ResourceType: resourceTypeOrg.Id, Resource: "org-1"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			userTrait, err := sdkResource.GetUserTrait(resource)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := userTrait.GetAccountType() == v2.UserTrait_ACCOUNT_TYPE_SERVICE
			if got != tt.wantService {
				t.Errorf("service account: want %v got %v (account type %v)", tt.wantService, got, userTrait.GetAccountType())
			}
		})
	}
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

// createAccountHandler serves the user lookup, pending-invite lookup and invite
// mutation that CreateAccount issues. onInvite is nil when no invite is expected.
func createAccountHandler(t *testing.T, users, invites, inviteResponse string, onInvite func(input map[string]interface{})) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = decodeJSON(t, r, &req)
		query, _ := req["query"].(string)
		vars, _ := req["variables"].(map[string]interface{})
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(query, "UserByEmail"):
			_, _ = w.Write([]byte(`{"data":{"users":{"nodes":` + users + `}}}`))
		case strings.Contains(query, "organizationInvites("):
			_, _ = w.Write([]byte(`{"data":{"organizationInvites":{"nodes":` + invites + `,"pageInfo":{"hasNextPage":false}}}}`))
		case strings.Contains(query, "organizationInviteCreate"):
			if onInvite == nil {
				t.Error("invite should not be sent")
			} else {
				onInvite(vars["input"].(map[string]interface{}))
			}
			_, _ = w.Write([]byte(inviteResponse))
		default:
			t.Errorf("unexpected query: %s", query)
		}
	}
}

const inviteSuccess = `{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-1"}}}}`

func TestUserCreateAccount_FromPrimaryEmail(t *testing.T) {
	var seenInput map[string]interface{}
	ub := newTestUserBuilder(t, createAccountHandler(t, `[]`, `[]`,
		`{"data":{"organizationInviteCreate":{"success":true,"organizationInvite":{"id":"inv-abc"}}}}`,
		func(input map[string]interface{}) { seenInput = input }))

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
	ub := newTestUserBuilder(t, createAccountHandler(t, `[]`, `[]`, inviteSuccess,
		func(input map[string]interface{}) { seenEmail = input["email"] }))

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
			ub := newTestUserBuilder(t, createAccountHandler(t, `[]`, `[]`, inviteSuccess,
				func(input map[string]interface{}) { seenRole = input["role"] }))

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

func TestUserCreateAccount_ExistingUser(t *testing.T) {
	ub := newTestUserBuilder(t, createAccountHandler(t,
		`[{"id":"user-1","name":"Existing User","email":"existing@example.com","active":true,"organization":{"id":"org-1"}}]`,
		`[]`, "", nil))

	info := &v2.AccountInfo{Emails: []*v2.AccountInfo_Email{{Address: "Existing@example.com", IsPrimary: true}}}
	resp, _, _, err := ub.CreateAccount(context.Background(), info, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	exists, ok := resp.(*v2.CreateAccountResponse_AlreadyExistsResult)
	if !ok {
		t.Fatalf("expected AlreadyExistsResult, got %T", resp)
	}
	if got := exists.Resource.GetId().GetResource(); got != "user-1" {
		t.Errorf("resource ID: want user-1 got %s", got)
	}
	if got := exists.Resource.GetParentResourceId().GetResource(); got != "org-1" {
		t.Errorf("parent resource ID: want org-1 got %s", got)
	}
}

func TestUserCreateAccount_SuspendedUser(t *testing.T) {
	ub := newTestUserBuilder(t, createAccountHandler(t,
		`[{"id":"user-1","name":"Suspended User","email":"suspended@example.com","active":false,"organization":{"id":"org-1"}}]`,
		`[]`, "", nil))

	info := &v2.AccountInfo{Login: "suspended@example.com"}
	_, _, _, err := ub.CreateAccount(context.Background(), info, nil)
	if err == nil || !strings.Contains(err.Error(), "suspended") {
		t.Fatalf("expected suspended-account error, got: %v", err)
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestUserCreateAccount_PendingInvite(t *testing.T) {
	ub := newTestUserBuilder(t, createAccountHandler(t, `[]`,
		`[{"id":"inv-old","email":"pending@example.com","acceptedAt":null,"expiresAt":"2000-01-01T00:00:00.000Z"},`+
			`{"id":"inv-pending","email":"Pending@example.com","acceptedAt":null,"expiresAt":"2999-01-01T00:00:00.000Z"}]`,
		"", nil))

	info := &v2.AccountInfo{Login: "pending@example.com"}
	resp, _, _, err := ub.CreateAccount(context.Background(), info, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	action, ok := resp.(*v2.CreateAccountResponse_ActionRequiredResult)
	if !ok {
		t.Fatalf("expected ActionRequiredResult, got %T", resp)
	}
	if !strings.Contains(action.Message, "inv-pending") {
		t.Errorf("message should contain pending invite ID, got: %s", action.Message)
	}
}

func TestUserCreateAccount_InviteErrorDetails(t *testing.T) {
	ub := newTestUserBuilder(t, createAccountHandler(t, `[]`, `[]`,
		`{"data":{"organizationInviteCreate":null},"errors":[{"message":"Invite limit reached"}]}`,
		func(map[string]interface{}) {}))

	info := &v2.AccountInfo{Login: "new@example.com"}
	_, _, _, err := ub.CreateAccount(context.Background(), info, nil)
	if err == nil || !strings.Contains(err.Error(), "Invite limit reached") {
		t.Fatalf("expected error to include Linear's message, got: %v", err)
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
