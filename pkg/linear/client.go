package linear

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

const APIEndpoint = "https://api.linear.app/graphql"

type Client struct {
	httpClient *uhttp.BaseHttpClient
	apiKey     string
	apiUrl     *url.URL
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	options := []uhttp.Option{uhttp.WithLogger(true, ctxzap.Extract(ctx))}

	httpClient, err := uhttp.NewClient(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP client failed: %w", err)
	}
	wrapper := uhttp.NewBaseHttpClient(httpClient)

	apiUrl, err := url.Parse(APIEndpoint)
	if err != nil {
		return nil, err
	}

	return &Client{
		apiKey:     apiKey,
		apiUrl:     apiUrl,
		httpClient: wrapper,
	}, nil
}

type GraphQLUsersResponse struct {
	Data struct {
		Users Users `json:"users"`
	} `json:"data"`
}

type GraphQLTeamsResponse struct {
	Data struct {
		Teams Teams `json:"teams"`
	} `json:"data"`
}

type GraphQLProjectsResponse struct {
	Data struct {
		Projects Projects `json:"projects"`
	} `json:"data"`
}

type GraphQLOrganizationResponse struct {
	Data struct {
		Organization Organization `json:"organization"`
	} `json:"data"`
}

type GetResourcesVars struct {
	First int    `json:"first,omitempty"`
	After string `json:"after,omitempty"`
}

type GraphQLTeamResponse struct {
	Data struct {
		Team Team `json:"team"`
	} `json:"data"`
}

type GraphQLProjectResponse struct {
	Data struct {
		Project Project `json:"project"`
	} `json:"data"`
}

type GraphQLIssueFieldsResponse struct {
	Data struct {
		Type struct {
			InputFields []IssueField `json:"inputFields"`
		} `json:"__type"`
	} `json:"data"`
}

type GraphQLViewerResponse struct {
	Data struct {
		Viewer ViewerPermissions `json:"viewer"`
	} `json:"data"`
}

type GraphQLIssueLabelsResponse struct {
	Data struct {
		IssueLabels struct {
			Nodes []IssueLabel `json:"nodes"`
		} `json:"issueLabels"`
	} `json:"data"`
}

type GraphQLIssuesResponse struct {
	Data struct {
		Issues struct {
			Nodes []Issue `json:"nodes"`
		} `json:"issues"`
	} `json:"data"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type GetTeamVars struct {
	TeamId string `json:"teamId"`
	After  string `json:"after,omitempty"`
	First  int    `json:"first,omitempty"`
}

type GetTeamsVars struct {
	TeamIDs []string `json:"teamIds,omitempty"`
	After   string   `json:"after,omitempty"`
	First   int      `json:"first,omitempty"`
}

type GetProjectVars struct {
	First      int    `json:"first,omitempty"`
	UsersAfter string `json:"usersAfter,omitempty"`
	TeamsAfter string `json:"teamsAfter,omitempty"`
	ProjectId  string `json:"projectId,omitempty"`
}

type PaginationVars struct {
	First      int    `json:"first,omitempty"`
	UsersAfter string `json:"usersAfter,omitempty"`
	TeamsAfter string `json:"teamsAfter,omitempty"`
}

type FieldOption func(createIssuePayload *CreateIssuePayload)

type CreateIssuePayload struct {
	TeamId       string
	Title        string
	Description  string
	LabelIDs     []string
	FieldOptions map[string]interface{}
}

// GetUsers returns all users from Linear organization.
func (c *Client) GetUsers(ctx context.Context, getResourceVars GetResourcesVars) ([]User, string, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Users($after: String, $first: Int) {
			users(after: $after, first: $first) {
				nodes {
					active
					admin
					displayName
					email
					guest
					id
					isMe
					name
					url
					description
					organization {
						id
					}
				}
				pageInfo {
					endCursor
					hasNextPage
					hasPreviousPage
					startCursor
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": getResourceVars,
	}

	var res GraphQLUsersResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	if res.Data.Users.PageInfo.HasNextPage {
		return res.Data.Users.Nodes, res.Data.Users.PageInfo.EndCursor, resp, rlData, nil
	}

	return res.Data.Users.Nodes, "", resp, rlData, nil
}

// GetTeams returns all teams from Linear organization.
func (c *Client) GetTeams(ctx context.Context, getResourceVars GetResourcesVars) ([]Team, string, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Teams($after: String, $first: Int) {
			teams(after: $after, first: $first) {
				nodes {
					id
					name
					key
					description
				}
				pageInfo {
					hasPreviousPage
					hasNextPage
					startCursor
					endCursor
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": getResourceVars,
	}

	var res GraphQLTeamsResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	if res.Data.Teams.PageInfo.HasNextPage {
		return res.Data.Teams.Nodes, res.Data.Teams.PageInfo.EndCursor, resp, rlData, nil
	}

	return res.Data.Teams.Nodes, "", resp, rlData, nil
}

// GetProjects returns all projects from Linear organization.
func (c *Client) GetProjects(ctx context.Context, getResourceVars GetResourcesVars) ([]Project, string, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Projects($after: String, $first: Int) {
			projects(after: $after, first: $first) {
				nodes {
					description
					id
					name
					slugId
					url
				}
				pageInfo {
					hasPreviousPage
					hasNextPage
					startCursor
					endCursor
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": getResourceVars,
	}

	var res GraphQLProjectsResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	if res.Data.Projects.PageInfo.HasNextPage {
		return res.Data.Projects.Nodes, res.Data.Projects.PageInfo.EndCursor, resp, rlData, nil
	}

	return res.Data.Projects.Nodes, "", resp, rlData, nil
}

// GetOrganization returns a single Linear organization.
func (c *Client) GetOrganization(ctx context.Context, paginationVars PaginationVars) (Organization, Tokens, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Organization($usersAfter: String, $teamsAfter: String, $first: Int) {
			organization {
				id
				name
				samlEnabled
				scimEnabled
				subscription {
					id
				}
				urlKey
				userCount
				teams(after: $teamsAfter, first: $first) {
					nodes {
						id
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
				users(after: $usersAfter, first: $first) {
					nodes {
						id
						admin
						guest
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": paginationVars,
	}

	var res GraphQLOrganizationResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Organization{}, Tokens{}, resp, rlData, err
	}

	var tokens Tokens

	if res.Data.Organization.Users.PageInfo.HasNextPage {
		tokens.UsersToken = res.Data.Organization.Users.PageInfo.EndCursor
	}

	if res.Data.Organization.Teams.PageInfo.HasNextPage {
		tokens.TeamsToken = res.Data.Organization.Teams.PageInfo.EndCursor
	}

	return res.Data.Organization, tokens, resp, rlData, nil
}

// GetTeam returns single Team details.
func (c *Client) GetTeam(ctx context.Context, getTeamVars GetTeamVars) (Team, string, *http.Response, *v2.RateLimitDescription, error) {
	vars := GetTeamVars{TeamId: getTeamVars.TeamId, First: getTeamVars.First, After: ""}

	if getTeamVars.After != "" {
		vars.After = getTeamVars.After
	}

	query := `query Team($teamId: String!, $after: String, $first: Int) {
			team(id: $teamId) {
				id
				name
				key
				description
				memberships(after: $after, first: $first) {
					nodes {
						id
						user {
							id
						}
						team {
							id
						}
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": vars,
	}

	var res GraphQLTeamResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Team{}, "", resp, rlData, err
	}

	if res.Data.Team.Memberships.PageInfo.HasNextPage {
		return res.Data.Team, res.Data.Team.Memberships.PageInfo.EndCursor, resp, rlData, nil
	}

	return res.Data.Team, "", resp, rlData, nil
}

// GetProject returns single Project details.
func (c *Client) GetProject(ctx context.Context, getProjectVars GetProjectVars) (Project, Tokens, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Project($projectId: String!, $usersAfter: String, $teamsAfter: String, $first: Int) {
			project(id: $projectId) {
				description
				id
				name
				slugId
				url
				teams(after: $teamsAfter, first: $first) {
					nodes {
						id
						name
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
				members(after: $usersAfter, first: $first) {
					nodes {
						id
						name
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": getProjectVars,
	}

	var res GraphQLProjectResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Project{}, Tokens{}, resp, rlData, err
	}

	var tokens Tokens

	if res.Data.Project.Members.PageInfo.HasNextPage {
		tokens.UsersToken = res.Data.Project.Members.PageInfo.EndCursor
	}

	if res.Data.Project.Teams.PageInfo.HasNextPage {
		tokens.TeamsToken = res.Data.Project.Teams.PageInfo.EndCursor
	}

	return res.Data.Project, tokens, resp, rlData, nil
}

// Authorize returns permissions of user calling the API.
func (c *Client) Authorize(ctx context.Context) (ViewerPermissions, *http.Response, *v2.RateLimitDescription, error) {
	query := `query Viewer{
			viewer {
				guest
				id
				admin
			}
		}`
	b := map[string]interface{}{
		"query": query,
	}

	var res GraphQLViewerResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return ViewerPermissions{}, resp, rlData, err
	}

	return res.Data.Viewer, resp, rlData, nil
}

func (c *Client) AddMemberToTeam(ctx context.Context, teamId, userId string) (string, error) {
	mutation := `mutation TeamMembershipCreate($input: TeamMembershipCreateInput!){
			teamMembershipCreate(input: $input) {
				success
				teamMembership {
					id
				}
			}
		}`

	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"teamId": teamId,
			"userId": userId,
		},
	}

	b := map[string]interface{}{
		"query":     mutation,
		"variables": vars,
	}

	var res struct {
		TeamMembership struct {
			ID string `json:"id"`
		} `json:"teamMembership"`
	}
	resp, _, e := c.doRequest(ctx, b, &res)
	if e != nil {
		return "", e
	}

	defer resp.Body.Close()

	return res.TeamMembership.ID, nil
}

func (c *Client) RemoveTeamMembership(ctx context.Context, teamMembershipId string) (bool, error) {
	mutation := `mutation TeamMembershipDelete($teamMembershipDeleteId: String!){
			teamMembershipDelete(id: $teamMembershipDeleteId) {
				success
			}
		}`

	b := map[string]interface{}{
		"query": mutation,
		"variables": map[string]interface{}{
			"teamMembershipDeleteId": teamMembershipId,
		},
	}

	var res struct {
		Data struct {
			TeamMembershipDelete SuccessResponse `json:"teamMembershipDelete"`
		} `json:"data"`
	}
	resp, _, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	return res.Data.TeamMembershipDelete.Success, nil
}

// GetTeamMemberships returns team memberships from Linear organization.
func (c *Client) GetTeamMemberships(ctx context.Context, getTeamVars GetTeamVars) ([]TeamMembership, string, *http.Response, *v2.RateLimitDescription, error) {
	vars := GetTeamVars{TeamId: getTeamVars.TeamId, First: getTeamVars.First, After: ""}

	if getTeamVars.After != "" {
		vars.After = getTeamVars.After
	}

	query := `query TeamMemberships($teamId: String!, $after: String, $first: Int) {
			teamMemberships(id: $teamId) {
				id
				memberships(after: $after, first: $first) {
					nodes {
						id
						team {
							id
							name
						}
						user {
							id
							name
						}
					}
					pageInfo {
						hasPreviousPage
						hasNextPage
						startCursor
						endCursor
					}
				}
			}
		}`
	b := map[string]interface{}{
		"query":     query,
		"variables": vars,
	}

	var res GraphQLTeamResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	return res.Data.Team.Memberships.Nodes, "", resp, rlData, nil
}

// ListTeamWorkflowStates returns workflow states for specific teams.
func (c *Client) ListTeamWorkflowStates(ctx context.Context, getTeamsVars GetTeamsVars) ([]Team, string, *http.Response, *v2.RateLimitDescription, error) {
	query := `query TeamWorkflowStates($after: String, $first: Int, $teamIds: [ID!]) {
		teams(after: $after, first: $first, filter: { id: { in: $teamIds } }) {
			nodes {
				id
				name
				states {
					nodes {
						id
						name
						color
						type
						position
					}
				}
			}
			pageInfo {
				hasPreviousPage
				hasNextPage
				startCursor
				endCursor
			}
		}
	}`

	b := map[string]interface{}{
		"query":     query,
		"variables": getTeamsVars,
	}

	var res GraphQLTeamsResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	if res.Data.Teams.PageInfo.HasNextPage {
		return res.Data.Teams.Nodes, res.Data.Teams.PageInfo.EndCursor, resp, rlData, nil
	}

	return res.Data.Teams.Nodes, "", resp, rlData, nil
}

func (c *Client) ListIssueFields(ctx context.Context) ([]IssueField, string, *http.Response, *v2.RateLimitDescription, error) {
	query := `query IssueFields {
				__type(name: "IssueCreateInput") {
					inputFields(includeDeprecated: false) {
						name
						description
						type {
							name
							kind
							ofType {
								name
								kind
								ofType {
									name
									kind
								}
							}
							enumValues(includeDeprecated: false) {
								name
							}
						}
					}
				}
			}`
	b := map[string]interface{}{
		"query": query,
	}

	var res GraphQLIssueFieldsResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", resp, rlData, err
	}

	return res.Data.Type.InputFields, "", resp, rlData, nil
}

func createIssuePayloadToInputMap(payload CreateIssuePayload) *map[string]interface{} {
	input := map[string]interface{}{
		"teamId":      payload.TeamId,
		"title":       payload.Title,
		"description": payload.Description,
	}
	if len(payload.LabelIDs) > 0 {
		input["labelIds"] = payload.LabelIDs
	}
	for key, value := range payload.FieldOptions {
		input[key] = value
	}
	return &input
}

func (c *Client) CreateIssue(ctx context.Context, payload CreateIssuePayload) (*Issue, error) {
	l := ctxzap.Extract(ctx)
	l.Debug("Creating issue", zap.Any("payload", payload))

	mutation := `mutation IssueCreate($input: IssueCreateInput!) {
		issueCreate(input: $input) {
		    issue {
		      id
		      title
		      description
		      state {
		        id
		        name
		      }
		      labels {
		        nodes {
		          id
		          name
		        }
		      }
		      createdAt
		      updatedAt
		      url
		    }
			success
		}
	}`

	input := createIssuePayloadToInputMap(payload)
	vars := map[string]interface{}{
		"input": input,
	}

	b := map[string]interface{}{
		"query":     mutation,
		"variables": vars,
	}

	var res struct {
		Data struct {
			IssueCreate struct {
				Success bool  `json:"success"`
				Issue   Issue `json:"issue"`
			} `json:"issueCreate"`
		} `json:"data"`
	}
	resp, _, e := c.doRequest(ctx, b, &res)
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()

	if !res.Data.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	return &res.Data.IssueCreate.Issue, nil
}

func (c *Client) BulkCreateIssues(ctx context.Context, payloads []CreateIssuePayload) ([]Issue, error) {
	query := `mutation IssueCreateBatch($input: IssueBatchCreateInput!) {
		issueBatchCreate(input: $input) {
			success
			issues {
				id
				title
				description
				state {
					id
					name
				}
				labels {
					nodes {
						id
						name
					}
				}
				createdAt
				updatedAt
				url
			}
		}
	}`

	issues := make([]map[string]interface{}, len(payloads))
	for i, payload := range payloads {
		issues[i] = *createIssuePayloadToInputMap(payload)
	}
	input := map[string]interface{}{"issues": issues}
	b := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"input": input},
	}

	var res struct {
		Data struct {
			IssueBatchCreate struct {
				Success bool    `json:"success"`
				Issues  []Issue `json:"issues"`
			} `json:"issueBatchCreate"`
		} `json:"data"`
	}
	resp, _, e := c.doRequest(ctx, b, &res)
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()

	return res.Data.IssueBatchCreate.Issues, nil
}

func (c *Client) GetIssue(ctx context.Context, issueId string) (*Issue, error) {
	query := `query Issue($issueId: String!) {
				issue(id: $issueId) {
					id
					title
					description
					state {
						id
						name
					}
					labels {
						nodes {
						id
						name
						}
					}
					createdAt
					updatedAt
					url
				}
			}`

	b := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"issueId": issueId},
	}

	var res struct {
		Data struct {
			Issue Issue `json:"issue"`
		} `json:"data"`
	}
	resp, _, e := c.doRequest(ctx, b, &res)
	if e != nil {
		return nil, e
	}

	defer resp.Body.Close()

	return &res.Data.Issue, nil
}

func (c *Client) ListIssuesByIDs(ctx context.Context, issueIDs []string) ([]Issue, *http.Response, *v2.RateLimitDescription, error) {
	if len(issueIDs) == 0 {
		return []Issue{}, nil, nil, nil
	}

	query := `query Issues($issueIds: [ID!]) {
		issues(filter: {
			id: {
				in: $issueIds
			}
		}) {
			nodes {
				id
				title
				description
				state {
					id
					name
				}
				labels {
					nodes {
						id
						name
					}
				}
				createdAt
				updatedAt
				url
			}
		}
	}`

	b := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"issueIds": issueIDs},
	}

	var res GraphQLIssuesResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, resp, rlData, err
	}

	return res.Data.Issues.Nodes, resp, rlData, nil
}

func (c *Client) GetIssueLabel(ctx context.Context, labelName string) (*IssueLabel, *http.Response, *v2.RateLimitDescription, error) {
	query := `query IssueLabel($labelName: String!) {
		issueLabels(filter: {
			name: {
				eq: $labelName
			}
		}) {
			nodes {
				id
				name
			}
		}
	}`

	b := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"labelName": labelName},
	}

	var res GraphQLIssueLabelsResponse
	resp, rlData, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, resp, rlData, err
	}

	if len(res.Data.IssueLabels.Nodes) == 0 {
		return nil, resp, rlData, nil
	}

	return &res.Data.IssueLabels.Nodes[0], resp, rlData, nil
}

func (c *Client) CreateIssueLabel(ctx context.Context, labelName string) (*IssueLabel, *http.Response, *v2.RateLimitDescription, error) {
	mutation := `mutation IssueLabelCreate($input: IssueLabelCreateInput!) {
		issueLabelCreate(input: $input) {
			success
			issueLabel {
				id
				name	
			}
		}
	}`

	vars := map[string]interface{}{
		"input": map[string]interface{}{
			"name": labelName,
		},
	}

	b := map[string]interface{}{
		"query":     mutation,
		"variables": vars,
	}

	var res struct {
		Data struct {
			IssueLabelCreate struct {
				Success    bool       `json:"success"`
				IssueLabel IssueLabel `json:"issueLabel"`
			} `json:"issueLabelCreate"`
		} `json:"data"`
	}
	resp, rlData, e := c.doRequest(ctx, b, &res)
	if e != nil {
		return nil, resp, rlData, e
	}

	defer resp.Body.Close()

	if !res.Data.IssueLabelCreate.Success {
		return nil, resp, rlData, fmt.Errorf("failed to create issue label")
	}

	return &res.Data.IssueLabelCreate.IssueLabel, resp, rlData, nil
}

func (c *Client) doRequest(ctx context.Context, body interface{}, res interface{}) (*http.Response, *v2.RateLimitDescription, error) {
	rlData := &v2.RateLimitDescription{}
	options := []uhttp.RequestOption{
		uhttp.WithHeader("Authorization", c.apiKey),
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithJSONBody(body),
	}

	req, err := c.httpClient.NewRequest(ctx, http.MethodPost, c.apiUrl, options...)
	if err != nil {
		return nil, rlData, err
	}

	var gqlErr GraphQLError
	doOptions := []uhttp.DoOption{
		uhttp.WithRatelimitData(rlData),
		uhttp.WithErrorResponse(&gqlErr),
		uhttp.WithJSONResponse(res),
	}

	resp, err := c.httpClient.Do(req, doOptions...)

	l := ctxzap.Extract(ctx)
	// Linear returns 400 when rate limited, so change it to a retryable error
	if err != nil && resp != nil && (resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusTooManyRequests) {
		l.Debug("rate limiting detected", zap.Int("status_code", resp.StatusCode))

		rlData.Status = v2.RateLimitDescription_STATUS_OVERLIMIT
		return resp, rlData, uhttp.WrapErrorsWithRateLimitInfo(codes.Unavailable, resp, err)
	}

	if resp != nil {
		l.Debug("returning without rate limit wrapping", zap.Int("status_code", resp.StatusCode))
	} else {
		l.Debug("returning without rate limit wrapping", zap.String("status_code", "nil response"))
	}
	return resp, rlData, err
}
