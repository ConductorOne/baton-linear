package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const APIEndpoint = "https://api.linear.app/graphql"

type Client struct {
	httpClient *http.Client
	apiKey     string
}

func NewClient(apiKey string, httpClient *http.Client) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
	}
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

type GraphQLViewerResponse struct {
	Data struct {
		Viewer ViewerPermissions `json:"viewer"`
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

// GetUsers returns all users from Linear organization.
func (c *Client) GetUsers(ctx context.Context, getResourceVars GetResourcesVars) ([]User, string, *http.Response, error) {
	vars, err := json.Marshal(getResourceVars)
	if err != nil {
		return nil, "", nil, err
	}
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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(vars),
	})

	var res GraphQLUsersResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", nil, err
	}

	if res.Data.Users.PageInfo.HasNextPage {
		return res.Data.Users.Nodes, res.Data.Users.PageInfo.EndCursor, resp, nil
	}

	return res.Data.Users.Nodes, "", resp, nil
}

// GetTeams returns all teams from Linear organization.
func (c *Client) GetTeams(ctx context.Context, getResourceVars GetResourcesVars) ([]Team, string, *http.Response, error) {
	varsB, err := json.Marshal(getResourceVars)
	if err != nil {
		return nil, "", nil, err
	}
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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(varsB),
	})

	var res GraphQLTeamsResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", nil, err
	}

	if res.Data.Teams.PageInfo.HasNextPage {
		return res.Data.Teams.Nodes, res.Data.Teams.PageInfo.EndCursor, resp, nil
	}

	return res.Data.Teams.Nodes, "", resp, nil
}

// GetProjects returns all projects from Linear organization.
func (c *Client) GetProjects(ctx context.Context, getResourceVars GetResourcesVars) ([]Project, string, *http.Response, error) {
	vars, err := json.Marshal(getResourceVars)
	if err != nil {
		return nil, "", nil, err
	}
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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(vars),
	})

	var res GraphQLProjectsResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", nil, err
	}

	if res.Data.Projects.PageInfo.HasNextPage {
		return res.Data.Projects.Nodes, res.Data.Projects.PageInfo.EndCursor, resp, nil
	}

	return res.Data.Projects.Nodes, "", resp, nil
}

// GetOrganization returns a single Linear organization.
func (c *Client) GetOrganization(ctx context.Context, paginationVars PaginationVars) (Organization, Tokens, *http.Response, error) {
	vars, err := json.Marshal(paginationVars)
	if err != nil {
		return Organization{}, Tokens{}, nil, err
	}

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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(vars),
	})

	var res GraphQLOrganizationResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Organization{}, Tokens{}, nil, err
	}

	var tokens Tokens

	if res.Data.Organization.Users.PageInfo.HasNextPage {
		tokens.UsersToken = res.Data.Organization.Users.PageInfo.EndCursor
	}

	if res.Data.Organization.Teams.PageInfo.HasNextPage {
		tokens.TeamsToken = res.Data.Organization.Teams.PageInfo.EndCursor
	}

	return res.Data.Organization, tokens, resp, nil
}

// GetTeam returns single Team details.
func (c *Client) GetTeam(ctx context.Context, getTeamVars GetTeamVars) (Team, string, *http.Response, error) {
	vars := GetTeamVars{TeamId: getTeamVars.TeamId, First: getTeamVars.First, After: ""}

	if getTeamVars.After != "" {
		vars.After = getTeamVars.After
	}

	jsonString, err := json.Marshal(vars)
	if err != nil {
		return Team{}, "", nil, err
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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(jsonString),
	})

	var res GraphQLTeamResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Team{}, "", nil, err
	}

	if res.Data.Team.Memberships.PageInfo.HasNextPage {
		return res.Data.Team, res.Data.Team.Memberships.PageInfo.EndCursor, resp, nil
	}

	return res.Data.Team, "", resp, nil
}

// GetProject returns single Project details.
func (c *Client) GetProject(ctx context.Context, getProjectVars GetProjectVars) (Project, Tokens, *http.Response, error) {
	vars, err := json.Marshal(getProjectVars)
	if err != nil {
		return Project{}, Tokens{}, nil, err
	}

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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(vars),
	})

	var res GraphQLProjectResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return Project{}, Tokens{}, nil, err
	}

	var tokens Tokens

	if res.Data.Project.Members.PageInfo.HasNextPage {
		tokens.UsersToken = res.Data.Project.Members.PageInfo.EndCursor
	}

	if res.Data.Project.Teams.PageInfo.HasNextPage {
		tokens.TeamsToken = res.Data.Project.Teams.PageInfo.EndCursor
	}

	return res.Data.Project, tokens, resp, nil
}

// Authorize returns permissions of user calling the API.
func (c *Client) Authorize(ctx context.Context) (ViewerPermissions, *http.Response, error) {
	query := `query Viewer{
			viewer {
				guest
				id
				admin
			}
		}`
	b, _ := json.Marshal(map[string]interface{}{
		"query": query,
	})

	var res GraphQLViewerResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return ViewerPermissions{}, nil, err
	}

	return res.Data.Viewer, resp, nil
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

	vars, err := json.Marshal(map[string]interface{}{
		"input": map[string]interface{}{
			"teamId": teamId,
			"userId": userId,
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to marshal vars: %w", err)
	}

	b, _ := json.Marshal(map[string]interface{}{
		"query":     mutation,
		"variables": string(vars),
	},
	)

	var res struct {
		TeamMembership struct {
			ID string `json:"id"`
		} `json:"teamMembership"`
	}
	resp, e := c.doRequest(ctx, b, &res)
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

	vars, err := json.Marshal(map[string]interface{}{
		"teamMembershipDeleteId": teamMembershipId,
	})

	if err != nil {
		return false, err
	}

	b, _ := json.Marshal(map[string]interface{}{
		"query":     mutation,
		"variables": string(vars),
	})

	var res struct {
		Data struct {
			TeamMembershipDelete SuccessResponse `json:"teamMembershipDelete"`
		} `json:"data"`
	}
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	return res.Data.TeamMembershipDelete.Success, nil
}

// GetTeamMemberships returns team memberships from Linear organization.
func (c *Client) GetTeamMemberships(ctx context.Context, getTeamVars GetTeamVars) ([]TeamMembership, string, *http.Response, error) {
	vars := GetTeamVars{TeamId: getTeamVars.TeamId, First: getTeamVars.First, After: ""}

	if getTeamVars.After != "" {
		vars.After = getTeamVars.After
	}

	jsonString, err := json.Marshal(vars)
	if err != nil {
		return nil, "", nil, err
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
	b, _ := json.Marshal(map[string]interface{}{
		"query":     query,
		"variables": string(jsonString),
	})

	var res GraphQLTeamResponse
	resp, err := c.doRequest(ctx, b, &res)
	if err != nil {
		return nil, "", nil, err
	}

	return res.Data.Team.Memberships.Nodes, "", resp, nil
}

func (c *Client) doRequest(ctx context.Context, body []byte, res interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, APIEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", c.apiKey)
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		// failed to parse successful response, try decoding GQL error
		var gqlErr GraphQLError
		if err := json.NewDecoder(resp.Body).Decode(&gqlErr); err == nil {
			return nil, errors.New(gqlErr.Errors[0].Message)
		}
		return nil, err
	}
	return resp, nil
}
