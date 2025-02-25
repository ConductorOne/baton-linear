package linear

type PageInfo struct {
	EndCursor       string `json:"endCursor"`
	HasNextPage     bool   `json:"hasNextPage"`
	HasPreviousPage bool   `json:"hasPreviousPage"`
	StartCursor     string `json:"startCursor"`
}

type Users struct {
	Nodes    []User   `json:"nodes"`
	PageInfo PageInfo `json:"pageInfo"`
}

type Teams struct {
	Nodes    []Team   `json:"nodes"`
	PageInfo PageInfo `json:"pageInfo"`
}

type Projects struct {
	Nodes    []Project `json:"nodes"`
	PageInfo PageInfo  `json:"pageInfo"`
}

type Organization struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	SamlEnabled  bool        `json:"samlEnabled"`
	ScimEnabled  bool        `json:"scimEnabled"`
	Subscription interface{} `json:"subscription"`
	URLKey       string      `json:"urlKey"`
	UserCount    int         `json:"userCount"`
	Users        struct {
		Nodes    []User   `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"users"`
	Teams struct {
		Nodes    []Team   `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"teams"`
}

type User struct {
	Active       bool         `json:"active"`
	Admin        bool         `json:"admin"`
	DisplayName  string       `json:"displayName"`
	Email        string       `json:"email"`
	Guest        bool         `json:"guest"`
	ID           string       `json:"id"`
	IsMe         bool         `json:"isMe"`
	Name         string       `json:"name"`
	URL          string       `json:"url"`
	Description  interface{}  `json:"description"`
	Organization Organization `json:"organization"`
	Teams        struct {
		Nodes []Team `json:"nodes"`
	} `json:"teams"`
}

type Team struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Key         string      `json:"key"`
	Description interface{} `json:"description"`
	Memberships struct {
		Nodes    []TeamMembership `json:"nodes"`
		PageInfo PageInfo         `json:"pageInfo"`
	}
	States struct {
		Nodes    []WorkflowState `json:"nodes"`
		PageInfo PageInfo        `json:"pageInfo"`
	} `json:"states,omitempty"`
}

type Project struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	SlugID      string `json:"slugId"`
	URL         string `json:"url"`
	Teams       struct {
		Nodes    []Team   `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"teams"`
	Members struct {
		Nodes    []User   `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"members"`
}

type GraphQLError struct {
	Error  string `json:"error"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func (e *GraphQLError) Message() string {
	if e.Error != "" {
		return e.Error
	}
	if len(e.Errors) == 0 {
		return "unknown graphql error"
	}
	return e.Errors[0].Message
}

type ViewerPermissions struct {
	Guest bool   `json:"guest"`
	Admin bool   `json:"admin"`
	ID    string `json:"id"`
}

type PageState struct {
	Token          string `json:"token"`
	ResourceTypeID string `json:"resource_type_id"`
	ResourceID     string `json:"resource_id"`
}

type ProjectTokensState struct {
	States       []PageState `json:"states"`
	CurrentState PageState   `json:"current_state"`
}

type Tokens struct {
	UsersToken string `json:"usersToken,omitempty"`
	TeamsToken string `json:"teamsToken,omitempty"`
}

type TeamMembership struct {
	ID   string `json:"id"`
	User User   `json:"user"`
	Team Team   `json:"team"`
}

type WorkflowType string

const (
	Backlog   WorkflowType = "backlog"
	Unstarted WorkflowType = "unstarted"
	Started   WorkflowType = "started"
	Completed WorkflowType = "completed"
	Canceled  WorkflowType = "canceled"
)

type WorkflowState struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Color    string       `json:"color"`
	Type     WorkflowType `json:"type"`
	Position float64      `json:"position"`
}

type IssueField struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        IssueFieldType `json:"type"`
}

type IssueFieldType struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Kind        string                `json:"kind"`
	OfType      *IssueFieldType       `json:"ofType,omitempty"`
	EnumValues  []IssueFieldEnumValue `json:"enumValues,omitempty"`
}

type IssueFieldEnumValue struct {
	Name string `json:"name"`
}
