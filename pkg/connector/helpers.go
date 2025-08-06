package connector

import (
	"encoding/json"

	"github.com/conductorone/baton-linear/pkg/linear"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// resourcePageSize defines a default page size for pagination.
const resourcePageSize = 50

func titleCase(s string) string {
	titleCaser := cases.Title(language.English)

	return titleCaser.String(s)
}

func parsePageToken(i string, resourceID *v2.ResourceId) (*pagination.Bag, error) {
	b := &pagination.Bag{}
	err := b.Unmarshal(i)
	if err != nil {
		return nil, err
	}

	if b.Current() == nil {
		b.Push(pagination.PageState{
			ResourceTypeID: resourceID.ResourceType,
			ResourceID:     resourceID.Resource,
		})
	}

	return b, nil
}

// parseMultipleTokens returns pagination options for GraphQL query.
func parseMultipleTokens(token *pagination.Token) (linear.PaginationVars, error) {
	if token == nil {
		return linear.PaginationVars{}, nil
	}

	state := linear.ProjectTokensState{}
	if token.Token != "" {
		err := json.Unmarshal([]byte(token.Token), &state)
		if err != nil {
			return linear.PaginationVars{}, err
		}
	}

	tokens := linear.Tokens{}
	if state.CurrentState.Token != "" {
		err := json.Unmarshal([]byte(state.CurrentState.Token), &tokens)
		if err != nil {
			return linear.PaginationVars{}, err
		}
	}

	paginationOptions := linear.PaginationVars{UsersAfter: "", TeamsAfter: "", First: resourcePageSize}

	if tokens.UsersToken != "" {
		paginationOptions.UsersAfter = tokens.UsersToken
	}

	if tokens.TeamsToken != "" {
		paginationOptions.TeamsAfter = tokens.TeamsToken
	}

	return paginationOptions, nil
}
