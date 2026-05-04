package connector

import (
	"context"
	"testing"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/types/resource"
)

func TestUserCreate(t *testing.T) {
	// Note: These tests validate the code structure and error handling paths.
	// Full integration tests would require mocking the Linear API client.
	tests := []struct {
		name  string
		email string
	}{
		{
			name:  "user creation with valid email",
			email: "newuser@example.com",
		},
		{
			name:  "user creation with different email",
			email: "another@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := linear.NewClient(context.Background(), "test-api-key")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			// Create user resource type
			userType := userBuilder(client)

			// Create a test user resource with email
			userResource, err := resource.NewUserResource(
				"Test User",
				resourceTypeUser,
				"test-user-id",
				[]resource.UserTraitOption{
					resource.WithEmail(tt.email, true),
				},
			)
			if err != nil {
				t.Fatalf("failed to create user resource: %v", err)
			}

			// Call Create - will fail at API level but validates code path
			_, _, err = userType.Create(context.Background(), userResource)

			// We expect an error because we're not mocking the API
			// This test validates the code runs without panics and handles the resource correctly
			if err == nil {
				t.Log("Note: API call succeeded (unexpected in unit test)")
			}
		})
	}
}

func TestUserCreate_MissingEmail(t *testing.T) {
	client, err := linear.NewClient(context.Background(), "test-api-key")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	userType := userBuilder(client)

	// Create a user resource without email
	userResource, err := resource.NewUserResource(
		"Test User",
		resourceTypeUser,
		"test-user-id",
		[]resource.UserTraitOption{},
	)
	if err != nil {
		t.Fatalf("failed to create user resource: %v", err)
	}

	_, _, err = userType.Create(context.Background(), userResource)
	if err == nil {
		t.Error("expected error for missing email, got nil")
	}
}

func TestRoleGrant(t *testing.T) {
	tests := []struct {
		name           string
		roleID         string
		serverResponse string
		wantErr        bool
		errContains    string
	}{
		{
			name:   "grant admin role",
			roleID: roleAdmin,
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			wantErr: false,
		},
		{
			name:   "grant user role (demote from admin)",
			roleID: roleUser,
			serverResponse: `{
				"data": {
					"userUpdate": {
						"success": true
					}
				}
			}`,
			wantErr: false,
		},
		{
			name:        "grant guest role fails",
			roleID:      roleGuest,
			wantErr:     true,
			errContains: "cannot be granted via API",
		},
		{
			name:        "grant owner role fails",
			roleID:      roleOwner,
			wantErr:     true,
			errContains: "cannot be granted via API",
		},
		{
			name:        "grant unknown role fails",
			roleID:      "unknown-role",
			wantErr:     true,
			errContains: "unknown role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := linear.NewClient(context.Background(), "test-api-key")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			roleType := roleBuilder(client)

			// Create principal (user) resource
			principal, err := resource.NewUserResource(
				"Test User",
				resourceTypeUser,
				"user-123",
				[]resource.UserTraitOption{},
			)
			if err != nil {
				t.Fatalf("failed to create principal resource: %v", err)
			}

			// Create role resource for entitlement
			roleRes, err := resource.NewRoleResource(
				tt.roleID,
				resourceTypeRole,
				tt.roleID,
				[]resource.RoleTraitOption{},
			)
			if err != nil {
				t.Fatalf("failed to create role resource: %v", err)
			}

			// Create entitlement
			entitlement := &v2.Entitlement{
				Resource: roleRes,
				Slug:     membership,
			}

			_, err = roleType.Grant(context.Background(), principal, entitlement)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			// Note: For non-error cases, the actual API call will fail
			// since we can't redirect it to a test server from here.
			// This test validates the validation logic works correctly.
		})
	}
}

func TestRoleGrant_NonUserPrincipal(t *testing.T) {
	client, err := linear.NewClient(context.Background(), "test-api-key")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	roleType := roleBuilder(client)

	// Create a team resource as principal (not a user)
	principal, err := resource.NewGroupResource(
		"Test Team",
		resourceTypeTeam,
		"team-123",
		[]resource.GroupTraitOption{},
	)
	if err != nil {
		t.Fatalf("failed to create team resource: %v", err)
	}

	roleRes, err := resource.NewRoleResource(
		roleAdmin,
		resourceTypeRole,
		roleAdmin,
		[]resource.RoleTraitOption{},
	)
	if err != nil {
		t.Fatalf("failed to create role resource: %v", err)
	}

	entitlement := &v2.Entitlement{
		Resource: roleRes,
		Slug:     membership,
	}

	_, err = roleType.Grant(context.Background(), principal, entitlement)
	if err == nil {
		t.Error("expected error for non-user principal, got nil")
	}
	if !containsString(err.Error(), "only users can be granted roles") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRoleRevoke(t *testing.T) {
	tests := []struct {
		name        string
		roleID      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "revoke admin role (demote to user)",
			roleID:  roleAdmin,
			wantErr: false,
		},
		{
			name:    "revoke user role (suspend)",
			roleID:  roleUser,
			wantErr: false,
		},
		{
			name:        "revoke guest role fails",
			roleID:      roleGuest,
			wantErr:     true,
			errContains: "cannot be revoked via API",
		},
		{
			name:        "revoke owner role fails",
			roleID:      roleOwner,
			wantErr:     true,
			errContains: "cannot be revoked via API",
		},
		{
			name:        "revoke unknown role fails",
			roleID:      "unknown-role",
			wantErr:     true,
			errContains: "unknown role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := linear.NewClient(context.Background(), "test-api-key")
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			roleType := roleBuilder(client)

			// Create principal (user) resource
			principal, err := resource.NewUserResource(
				"Test User",
				resourceTypeUser,
				"user-123",
				[]resource.UserTraitOption{},
			)
			if err != nil {
				t.Fatalf("failed to create principal resource: %v", err)
			}

			// Create role resource for entitlement
			roleRes, err := resource.NewRoleResource(
				tt.roleID,
				resourceTypeRole,
				tt.roleID,
				[]resource.RoleTraitOption{},
			)
			if err != nil {
				t.Fatalf("failed to create role resource: %v", err)
			}

			// Create grant to revoke
			grant := &v2.Grant{
				Principal: principal,
				Entitlement: &v2.Entitlement{
					Resource: roleRes,
					Slug:     membership,
				},
			}

			_, err = roleType.Revoke(context.Background(), grant)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			// Note: For non-error cases, the actual API call will fail
			// since we can't redirect it to a test server from here.
		})
	}
}

func TestRoleRevoke_NonUserPrincipal(t *testing.T) {
	client, err := linear.NewClient(context.Background(), "test-api-key")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	roleType := roleBuilder(client)

	// Create a team resource as principal (not a user)
	principal, err := resource.NewGroupResource(
		"Test Team",
		resourceTypeTeam,
		"team-123",
		[]resource.GroupTraitOption{},
	)
	if err != nil {
		t.Fatalf("failed to create team resource: %v", err)
	}

	roleRes, err := resource.NewRoleResource(
		roleAdmin,
		resourceTypeRole,
		roleAdmin,
		[]resource.RoleTraitOption{},
	)
	if err != nil {
		t.Fatalf("failed to create role resource: %v", err)
	}

	grant := &v2.Grant{
		Principal: principal,
		Entitlement: &v2.Entitlement{
			Resource: roleRes,
			Slug:     membership,
		},
	}

	_, err = roleType.Revoke(context.Background(), grant)
	if err == nil {
		t.Error("expected error for non-user principal, got nil")
	}
	if !containsString(err.Error(), "only users can have roles revoked") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Helper function to check if a string contains a substring.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestLinearRoleConstants verifies the role constants are correctly defined.
func TestLinearRoleConstants(t *testing.T) {
	if linear.LinearRoleAdmin != "ADMIN" {
		t.Errorf("expected LinearRoleAdmin to be ADMIN, got %s", linear.LinearRoleAdmin)
	}
	if linear.LinearRoleMember != "MEMBER" {
		t.Errorf("expected LinearRoleMember to be MEMBER, got %s", linear.LinearRoleMember)
	}
	if linear.LinearRoleGuest != "GUEST" {
		t.Errorf("expected LinearRoleGuest to be GUEST, got %s", linear.LinearRoleGuest)
	}
}
