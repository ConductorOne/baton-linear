package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"github.com/conductorone/baton-sdk/pkg/types/grant"
	sdkResource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

var (
	_ connectorbuilder.ResourceSyncer          = (*userResourceType)(nil)
	_ connectorbuilder.AccountManagerLimited   = (*userResourceType)(nil)
	_ connectorbuilder.ResourceDeleterLimited  = (*userResourceType)(nil)
)

const userRoleProfileKey = "user_role"

type userResourceType struct {
	resourceType *v2.ResourceType
	client       *linear.Client
}

func (o *userResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Linear user.
func userResource(ctx context.Context, user *linear.User, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	names := strings.SplitN(user.Name, " ", 2)
	var firstName, lastName string
	switch len(names) {
	case 1:
		firstName = names[0]
	case 2:
		firstName = names[0]
		lastName = names[1]
	}

	var userRole string
	switch {
	case user.Owner:
		userRole = roleOwner
	case user.Admin:
		userRole = roleAdmin
	case user.Guest:
		userRole = roleGuest
	default:
		userRole = roleUser
	}

	profile := map[string]interface{}{
		"first_name":       firstName,
		"last_name":        lastName,
		"login":            user.Email,
		"user_id":          user.ID,
		userRoleProfileKey: userRole,
	}

	userTraitOptions := []sdkResource.UserTraitOption{
		sdkResource.WithUserProfile(profile),
		sdkResource.WithEmail(user.Email, true),
		sdkResource.WithStatus(v2.UserTrait_Status_STATUS_ENABLED),
	}

	ret, err := sdkResource.NewUserResource(
		user.Name,
		resourceTypeUser,
		user.ID,
		userTraitOptions,
		sdkResource.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *userResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var annotations annotations.Annotations
	if parentId == nil {
		return nil, "", nil, nil
	}

	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeUser.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, nextToken, rlData, err := o.client.GetUsers(ctx, linear.GetResourcesVars{First: resourcePageSize, After: bag.PageToken()})
	annotations.WithRateLimiting(rlData)
	if err != nil {
		return nil, "", annotations, fmt.Errorf("linear-connector: failed to list users: %w", err)
	}

	pageToken, err := bag.NextToken(nextToken)
	if err != nil {
		return nil, "", annotations, err
	}

	var rv []*v2.Resource
	for _, user := range users {
		userCopy := user
		ur, err := userResource(ctx, &userCopy, parentId)
		if err != nil {
			return nil, "", annotations, err
		}
		rv = append(rv, ur)
	}

	return rv, pageToken, annotations, nil
}

func (o *userResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *userResourceType) Grants(ctx context.Context, resource *v2.Resource, pt *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var rv []*v2.Grant
	userTrait, err := sdkResource.GetUserTrait(resource)
	if err != nil {
		return nil, "", nil, fmt.Errorf("list-grants: Failed to get user trait from user: %w", err)
	}
	userProfile := userTrait.GetProfile()
	userRole, present := sdkResource.GetProfileStringValue(userProfile, userRoleProfileKey)
	if !present {
		return nil, "", nil, fmt.Errorf("list-grants: user role was not present on profile")
	}
	rr, err := roleResource(ctx, userRole, resource.ParentResourceId)
	if err != nil {
		return nil, "", nil, err
	}
	gr := grant.NewGrant(rr, membership, resource.Id)

	rv = append(rv, gr)
	return rv, "", nil, nil
}

func (o *userResourceType) CreateAccountCapabilityDetails(_ context.Context) (*v2.CredentialDetailsAccountProvisioning, annotations.Annotations, error) {
	return &v2.CredentialDetailsAccountProvisioning{
		SupportedCredentialOptions: []v2.CapabilityDetailCredentialOption{
			v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
		},
		PreferredCredentialOption: v2.CapabilityDetailCredentialOption_CAPABILITY_DETAIL_CREDENTIAL_OPTION_NO_PASSWORD,
	}, nil, nil
}

// CreateAccount provisions a new Linear user by sending a workspace invite. The
// user only becomes a Linear User after they accept the invite, so this returns
// an ActionRequiredResult — there is no resource yet.
func (o *userResourceType) CreateAccount(
	ctx context.Context,
	accountInfo *v2.AccountInfo,
	_ *v2.LocalCredentialOptions,
) (connectorbuilder.CreateAccountResponse, []*v2.PlaintextData, annotations.Annotations, error) {
	email := accountEmail(accountInfo)
	if email == "" {
		return nil, nil, nil, fmt.Errorf("baton-linear: email is required to create an invite")
	}

	role := accountRole(accountInfo)

	inviteID, err := o.client.CreateOrganizationInvite(ctx, email, role, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("baton-linear: failed to create organization invite: %w", err)
	}

	return &v2.CreateAccountResponse_ActionRequiredResult{
		Resource:              nil,
		Message:               fmt.Sprintf("Invitation sent to %s (invite ID: %s). The user must accept the invite to join the workspace.", email, inviteID),
		IsCreateAccountResult: true,
	}, nil, nil, nil
}

// Delete deprovisions a Linear user by suspending them. Linear does not delete
// user records; userSuspend revokes workspace access and invalidates sessions.
func (o *userResourceType) Delete(ctx context.Context, resourceId *v2.ResourceId) (annotations.Annotations, error) {
	if resourceId.GetResourceType() != resourceTypeUser.Id {
		return nil, fmt.Errorf("baton-linear: non-user resource passed to user delete: %s", resourceId.GetResourceType())
	}

	success, err := o.client.SuspendUser(ctx, resourceId.GetResource())
	if err != nil {
		return nil, fmt.Errorf("baton-linear: failed to suspend user: %w", err)
	}
	if !success {
		return nil, fmt.Errorf("baton-linear: userSuspend returned success=false")
	}
	return nil, nil
}

// accountEmail extracts the invitee's email from AccountInfo, preferring the
// primary email, then any email, then Login.
func accountEmail(accountInfo *v2.AccountInfo) string {
	if accountInfo == nil {
		return ""
	}
	for _, e := range accountInfo.GetEmails() {
		if e.GetIsPrimary() && e.GetAddress() != "" {
			return e.GetAddress()
		}
	}
	for _, e := range accountInfo.GetEmails() {
		if e.GetAddress() != "" {
			return e.GetAddress()
		}
	}
	return accountInfo.GetLogin()
}

// accountRole extracts the requested Linear role from the profile. Defaults to
// "user". Valid Linear values are admin, guest, user (owner is granted manually).
func accountRole(accountInfo *v2.AccountInfo) string {
	if accountInfo == nil {
		return ""
	}
	profile := accountInfo.GetProfile()
	if profile == nil {
		return ""
	}
	field, ok := profile.GetFields()[userRoleProfileKey]
	if !ok || field == nil {
		return ""
	}
	v := strings.ToLower(strings.TrimSpace(field.GetStringValue()))
	switch v {
	case roleAdmin, roleGuest, roleUser:
		return v
	default:
		return ""
	}
}

func userBuilder(client *linear.Client) *userResourceType {
	return &userResourceType{
		resourceType: resourceTypeUser,
		client:       client,
	}
}
