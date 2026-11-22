package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type roleResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message    string `json:"message"`
		Extensions struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"extensions"`
	} `json:"errors"`
}

func (r *roleResponse) Message() string {
	if len(r.Errors) > 0 {
		return r.Errors[0].Message
	}
	return "unknown graphql role error"
}

// GetUserRole reads the current role instead of relying on a potentially stale
// synced profile. Linear exposes workspace roles as flags on User.
func (c *Client) GetUserRole(ctx context.Context, userID string) (string, *v2.RateLimitDescription, error) {
	query := `query UserRole($id: String!) {
		user(id: $id) { id app owner admin guest }
	}`
	var data struct {
		User *User `json:"user"`
	}
	rlData, err := c.doRoleRequest(ctx, query, map[string]interface{}{"id": userID}, &data)
	if err != nil {
		return "", rlData, err
	}
	if data.User == nil || data.User.ID == "" {
		return "", rlData, status.Error(codes.NotFound, "baton-linear: user not found")
	}
	switch {
	case data.User.App:
		return "app", rlData, nil
	case data.User.Owner:
		return "owner", rlData, nil
	case data.User.Admin:
		return "admin", rlData, nil
	case data.User.Guest:
		return "guest", rlData, nil
	default:
		return "user", rlData, nil
	}
}

// ChangeUserRole changes only the workspace role. Linear requires workspace
// admin permissions and a caller role at least as high as the requested role.
func (c *Client) ChangeUserRole(ctx context.Context, userID, role string) (*v2.RateLimitDescription, error) {
	mutation := `mutation UserChangeRole($id: String!, $role: UserRoleType!) {
		userChangeRole(id: $id, role: $role) { success }
	}`
	var data struct {
		UserChangeRole SuccessResponse `json:"userChangeRole"`
	}
	rlData, err := c.doRoleRequest(ctx, mutation, map[string]interface{}{"id": userID, "role": role}, &data)
	if err != nil {
		return rlData, err
	}
	if !data.UserChangeRole.Success {
		return rlData, status.Error(codes.Internal, "baton-linear: userChangeRole returned success=false")
	}
	return rlData, nil
}

// Role requests check GraphQL errors even on HTTP 200. Keep this separate from
// the existing transport so other operations retain their current behavior.
func (c *Client) doRoleRequest(ctx context.Context, query string, variables map[string]interface{}, data interface{}) (*v2.RateLimitDescription, error) {
	rlData := &v2.RateLimitDescription{}
	req, err := c.httpClient.NewRequest(ctx, http.MethodPost, c.apiUrl,
		uhttp.WithHeader("Authorization", c.apiKey),
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithJSONBody(map[string]interface{}{"query": query, "variables": variables}),
	)
	if err != nil {
		return rlData, err
	}
	var response roleResponse
	resp, requestErr := c.httpClient.Do(req,
		uhttp.WithJSONResponse(&response),
		uhttp.WithErrorResponse(&response),
		uhttp.WithRatelimitData(rlData),
	)
	defer closeResponse(resp)

	// Linear has historically returned HTTP 400 for rate limits; only classify
	// it as retryable when the error code confirms this is a rate limit.
	for _, gqlErr := range response.Errors {
		if gqlErr.Extensions.Code == "RATELIMITED" {
			rlData.Status = v2.RateLimitDescription_STATUS_OVERLIMIT
			return rlData, uhttp.WrapErrorsWithRateLimitInfo(codes.Unavailable, resp, fmt.Errorf("baton-linear: %s", gqlErr.Message))
		}
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			rlData.Status = v2.RateLimitDescription_STATUS_OVERLIMIT
			return rlData, uhttp.WrapErrorsWithRateLimitInfo(codes.Unavailable, resp, requestErr)
		case http.StatusForbidden:
			return rlData, status.Error(codes.PermissionDenied, "baton-linear: workspace role changes require admin permissions and a caller role equal to or higher than the target role")
		case http.StatusUnauthorized:
			return rlData, status.Error(codes.Unauthenticated, "baton-linear: invalid API credentials")
		}
	}
	for _, gqlErr := range response.Errors {
		switch strings.ToUpper(gqlErr.Extensions.Code) {
		case "FORBIDDEN", "PERMISSION_DENIED", "INSUFFICIENT_SCOPE":
			return rlData, rolePermissionError(gqlErr.Message)
		}
		if strings.EqualFold(gqlErr.Extensions.Type, "Forbidden") {
			return rlData, rolePermissionError(gqlErr.Message)
		}
	}
	if requestErr != nil {
		return rlData, requestErr
	}
	if len(response.Errors) > 0 {
		return rlData, status.Errorf(codes.Unknown, "baton-linear: role request failed: %s", response.Errors[0].Message)
	}
	if len(response.Data) == 0 || string(response.Data) == "null" {
		return rlData, status.Error(codes.Internal, "baton-linear: role request returned no data")
	}
	if err := json.Unmarshal(response.Data, data); err != nil {
		return rlData, uhttp.WrapErrors(codes.Internal, "baton-linear: decoding role response", err)
	}
	return rlData, nil
}

func rolePermissionError(message string) error {
	return status.Errorf(codes.PermissionDenied,
		"baton-linear: workspace role changes require admin permissions and a caller role equal to or higher than the target role: %s", message)
}
