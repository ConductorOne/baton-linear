package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRoleProvisioning(t *testing.T) {
	tests := []struct {
		name, role, current, response string
		revoke, mutationError, legacy bool
		httpStatus                    int
		wantCode                      codes.Code
		wantMutation, wantAlready     bool
	}{
		{name: "grant guest", role: roleGuest, current: roleUser, wantMutation: true},
		{name: "grant user", role: roleUser, current: roleGuest, wantMutation: true},
		{name: "grant admin", role: roleAdmin, current: roleUser, wantMutation: true},
		{name: "grant owner", role: roleOwner, current: roleAdmin, wantMutation: true},
		{name: "legacy native ID", role: roleAdmin, current: roleUser, legacy: true, wantMutation: true},
		{name: "grant exists", role: roleAdmin, current: roleAdmin, wantAlready: true},
		{name: "revoke guest", role: roleGuest, current: roleGuest, revoke: true, wantMutation: true},
		{name: "revoke admin", role: roleAdmin, current: roleAdmin, revoke: true, wantMutation: true},
		{name: "revoke owner", role: roleOwner, current: roleOwner, revoke: true, wantMutation: true},
		{name: "already revoked", role: roleAdmin, current: roleUser, revoke: true, wantAlready: true},
		{name: "different role untouched", role: roleAdmin, current: roleOwner, revoke: true, wantAlready: true},
		{name: "baseline cannot be revoked", role: roleUser, current: roleUser, revoke: true, wantCode: codes.Unimplemented},
		{name: "displaced baseline already revoked", role: roleUser, current: roleAdmin, revoke: true, wantAlready: true},
		{name: "grant app rejected", role: "app", wantCode: codes.Unimplemented},
		{name: "revoke app rejected", role: "app", revoke: true, wantCode: codes.Unimplemented},
		{name: "unknown role", role: "invalid", wantCode: codes.InvalidArgument},
		{name: "app account grant rejected", role: roleUser, current: "app", wantCode: codes.Unimplemented},
		{name: "app account revoke rejected", role: roleAdmin, current: "app", revoke: true, wantCode: codes.Unimplemented},
		{name: "missing user", role: roleAdmin, response: `{"data":{"user":null}}`, wantCode: codes.NotFound},
		{name: "lookup failure", role: roleAdmin, response: `{"errors":[{"message":"query failed"}]}`, wantCode: codes.Unknown},
		{name: "owner permission denied", role: roleOwner, current: roleAdmin, mutationError: true, wantMutation: true,
			response: `{"errors":[{"message":"Cannot promote to owner","extensions":{"code":"FORBIDDEN"}}]}`, wantCode: codes.PermissionDenied},
		{name: "revoke permission denied", role: roleOwner, current: roleOwner, revoke: true, mutationError: true, wantMutation: true,
			response: `{"errors":[{"message":"Requires admin","extensions":{"type":"Forbidden"}}]}`, wantCode: codes.PermissionDenied},
		{name: "HTTP forbidden", role: roleAdmin, httpStatus: http.StatusForbidden, wantCode: codes.PermissionDenied},
		{name: "HTTP unauthorized", role: roleAdmin, httpStatus: http.StatusUnauthorized, wantCode: codes.Unauthenticated},
		{name: "false success", role: roleAdmin, current: roleUser, mutationError: true, wantMutation: true,
			response: `{"data":{"userChangeRole":{"success":false}}}`, wantCode: codes.Internal},
		{name: "partial mutation error", role: roleAdmin, current: roleUser, mutationError: true, wantMutation: true,
			response: `{"data":{"userChangeRole":{"success":true}},"errors":[{"message":"failed"}]}`, wantCode: codes.Unknown},
		{name: "no data", role: roleAdmin, response: `{}`, wantCode: codes.Internal},
		{name: "malformed JSON", role: roleAdmin, response: `{`, wantCode: codes.Unknown},
		{name: "bad request is not rate limit", role: roleAdmin, httpStatus: http.StatusBadRequest,
			response: `{"errors":[{"message":"invalid query"}]}`, wantCode: codes.InvalidArgument},
		{name: "rate limit lookup", role: roleAdmin, httpStatus: http.StatusBadRequest,
			response: `{"errors":[{"message":"slow down","extensions":{"code":"RATELIMITED"}}]}`, wantCode: codes.Unavailable},
		{name: "rate limit mutation", role: roleAdmin, current: roleUser, mutationError: true, wantMutation: true, httpStatus: http.StatusTooManyRequests,
			response: `{"errors":[{"message":"slow down","extensions":{"code":"RATELIMITED"}}]}`, wantCode: codes.Unavailable},
		{name: "server error remains retryable", role: roleAdmin, httpStatus: http.StatusServiceUnavailable,
			response: `{"errors":[{"message":"temporarily unavailable"}]}`, wantCode: codes.Unavailable},
		{name: "empty server error remains retryable", role: roleAdmin, httpStatus: http.StatusInternalServerError,
			response: `{}`, wantCode: codes.Unavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests, mutations := 0, 0
			expectedID := "native-user"
			if tt.legacy {
				expectedID = "legacy-user"
			}
			ub := newTestUserBuilder(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				var req struct {
					Query     string            `json:"query"`
					Variables map[string]string `json:"variables"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				if req.Variables["id"] != expectedID || r.Header.Get("Authorization") != "test-api-key" {
					t.Errorf("unexpected target or authentication: %+v", req.Variables)
				}
				mutation := strings.Contains(req.Query, "mutation UserChangeRole")
				if mutation {
					mutations++
					wantRole := tt.role
					if tt.revoke {
						wantRole = roleUser
					}
					if req.Variables["role"] != wantRole || !strings.Contains(req.Query, "userChangeRole(id: $id, role: $role)") {
						t.Errorf("unexpected mutation: %+v", req)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				if tt.response != "" && mutation == tt.mutationError || tt.httpStatus != 0 && !tt.mutationError {
					if tt.httpStatus != 0 {
						w.WriteHeader(tt.httpStatus)
					}
					_, _ = w.Write([]byte(tt.response))
					return
				}
				if mutation {
					_, _ = w.Write([]byte(`{"data":{"userChangeRole":{"success":true}}}`))
					return
				}
				flags := map[string]interface{}{"id": expectedID, "admin": false, "owner": false, "guest": false, "app": false}
				if tt.current != roleUser {
					flags[tt.current] = true
				}
				if err := json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"user": flags}}); err != nil {
					t.Error(err)
				}
			})
			principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: "user", Resource: "legacy-user"}}
			if !tt.legacy {
				principal.ExternalId = &v2.ExternalId{Id: "native-user"} //nolint:staticcheck // Exercise native ID compatibility.
			}
			entitlement := roleTestEntitlement(t, tt.role)
			rb := roleBuilder(ub.client)
			var annos annotations.Annotations
			var err error
			if tt.revoke {
				annos, err = rb.Revoke(context.Background(), &v2.Grant{Principal: principal, Entitlement: entitlement})
			} else {
				annos, err = rb.Grant(context.Background(), principal, entitlement)
			}
			if status.Code(err) != tt.wantCode {
				t.Fatalf("code: got %v, want %v: %v", status.Code(err), tt.wantCode, err)
			}
			switch tt.name {
			case "lookup failure", "false success", "partial mutation error", "no data":
				if _, ok := status.FromError(err); !ok {
					t.Errorf("expected an explicit gRPC status: %v", err)
				}
			}
			if tt.wantCode == codes.PermissionDenied && !strings.Contains(err.Error(), "admin permissions") {
				t.Errorf("permission error lacks actionable context: %v", err)
			}
			if (mutations == 1) != tt.wantMutation || mutations > 1 {
				t.Errorf("mutations: got %d, expected mutation %v", mutations, tt.wantMutation)
			}
			if tt.role == "invalid" || tt.role == "app" {
				if requests != 0 {
					t.Errorf("invalid operation made %d requests", requests)
				}
			}
			if tt.wantAlready {
				if tt.revoke && !annos.Contains(&v2.GrantAlreadyRevoked{}) || !tt.revoke && !annos.Contains(&v2.GrantAlreadyExists{}) {
					t.Error("missing idempotency annotation")
				}
			}
			if tt.wantCode == codes.Unavailable && tt.httpStatus < http.StatusInternalServerError {
				var rl v2.RateLimitDescription
				ok, pickErr := annos.Pick(&rl)
				if pickErr != nil || !ok || rl.Status != v2.RateLimitDescription_STATUS_OVERLIMIT {
					t.Errorf("missing rate limit annotation: %v, %v", &rl, pickErr)
				}
			}
		})
	}
}

func roleTestEntitlement(t *testing.T, role string) *v2.Entitlement {
	t.Helper()
	r, err := roleResource(context.Background(), role, &v2.ResourceId{ResourceType: "org", Resource: "workspace"})
	if err != nil {
		t.Fatal(err)
	}
	entitlements, _, _, err := roleBuilder(nil).Entitlements(context.Background(), r, nil)
	if err != nil {
		t.Fatal(err)
	}
	return entitlements[0]
}

func TestRoleProvisioningInvalidTargets(t *testing.T) {
	principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: "user", Resource: "user-1"}}
	for _, target := range []struct {
		principal   *v2.Resource
		entitlement *v2.Entitlement
	}{
		{nil, roleTestEntitlement(t, roleAdmin)},
		{&v2.Resource{}, roleTestEntitlement(t, roleAdmin)},
		{&v2.Resource{Id: &v2.ResourceId{ResourceType: "team", Resource: "team-1"}}, roleTestEntitlement(t, roleAdmin)},
		{&v2.Resource{Id: &v2.ResourceId{ResourceType: "user"}}, roleTestEntitlement(t, roleAdmin)},
		{principal, nil},
		{principal, &v2.Entitlement{Resource: &v2.Resource{Id: &v2.ResourceId{ResourceType: "team", Resource: "team-1"}}, Slug: membership}},
		{principal, &v2.Entitlement{Resource: roleTestEntitlement(t, roleAdmin).Resource, Slug: "invalid"}},
	} {
		rb := roleBuilder(nil) // Invalid inputs must fail before any client access.
		if _, err := rb.Grant(context.Background(), target.principal, target.entitlement); status.Code(err) != codes.InvalidArgument {
			t.Errorf("Grant: got %v", err)
		}
		if _, err := rb.Revoke(context.Background(), &v2.Grant{Principal: target.principal, Entitlement: target.entitlement}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("Revoke: got %v", err)
		}
	}
	if _, err := roleBuilder(nil).Revoke(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Errorf("nil Revoke: got %v", err)
	}
}

func TestRoleExclusionGroupAndNativeID(t *testing.T) {
	for _, role := range roles {
		entitlement := roleTestEntitlement(t, role)
		var group v2.EntitlementExclusionGroup
		annos := annotations.Annotations(entitlement.Annotations)
		ok, err := annos.Pick(&group)
		if err != nil || !ok || group.ExclusionGroupId != "linear-workspace-role:workspace" {
			t.Errorf("%s exclusion group: %v, %v", role, &group, err)
		}
	}
	u, err := userResource(context.Background(), &linear.User{ID: "native-user"}, nil)
	if err != nil || u.GetExternalId().GetId() != "native-user" { //nolint:staticcheck // Verify compatibility metadata required by provisioning guidance.
		t.Errorf("user native ID not preserved: %v, %v", u, err)
	}
}

func TestTeamProvisioningUnchanged(t *testing.T) {
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Query string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		queries = append(queries, req.Query)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":{"teamMembershipDelete":{"success":true},"teamMembershipCreate":{"success":true}}}`)
	}))
	t.Cleanup(server.Close)
	client, err := linear.NewClient(context.Background(), "test-key", server.URL)
	if err != nil {
		t.Fatal(err)
	}
	team := teamBuilder(client)
	principal := &v2.Resource{Id: &v2.ResourceId{ResourceType: "user", Resource: "user-1"}}
	entitlement := &v2.Entitlement{Resource: &v2.Resource{Id: &v2.ResourceId{ResourceType: "team", Resource: "team-1"}}}
	if _, err := team.Grant(context.Background(), principal, entitlement); err != nil {
		t.Fatal(err)
	}
	metadata, err := structpb.NewStruct(map[string]interface{}{"membership_id": "membership-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := team.Revoke(context.Background(), &v2.Grant{Principal: principal, Entitlement: entitlement, Annotations: annotations.New(metadata)}); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || !strings.Contains(queries[0], "teamMembershipCreate") || !strings.Contains(queries[1], "teamMembershipDelete") {
		t.Errorf("unexpected team operations: %v", queries)
	}
}
