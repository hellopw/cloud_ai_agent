package model

import "time"

type Prompt struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Skill struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Tool struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Label         string    `json:"label"`
	Description   string    `json:"description"`
	DSLDefinition string    `json:"dsl_definition"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Memory struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Source      string    `json:"source"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Template struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	AgentType         string    `json:"agent_type"`
	DockerfileContent string    `json:"dockerfile_content"`
	EffectiveDockerfile string   `json:"effective_dockerfile,omitempty"`
	PromptIDs         []string  `json:"prompt_ids,omitempty"`
	SkillIDs          []string  `json:"skill_ids,omitempty"`
	ToolIDs           []string  `json:"tool_ids,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TemplateID  string    `json:"template_id"`
	RepoURL     string    `json:"repo_url"`
	Branch      string    `json:"branch"`
	GitUsername string    `json:"git_username,omitempty"`
	GitPassword string    `json:"git_password,omitempty"`
	ImageTag    string    `json:"image_tag"`
	Status      string    `json:"status"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AgentTeam struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	TemplateID  string       `json:"template_id"`
	RepoURL     string       `json:"repo_url"`
	Branch      string       `json:"branch"`
	GitUsername string       `json:"git_username,omitempty"`
	GitPassword string       `json:"git_password,omitempty"`
	ImageTag    string       `json:"image_tag"`
	Status      string       `json:"status"`
	ErrorMsg    string       `json:"error_msg,omitempty"`
	PromptIDs   []string     `json:"prompt_ids,omitempty"`
	SkillIDs    []string     `json:"skill_ids,omitempty"`
	Members     []TeamMember `json:"members,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type TeamMember struct {
	ID                   string    `json:"id"`
	TeamID               string    `json:"team_id"`
	Name                 string    `json:"name"`
	Role                 string    `json:"role"`
	AgentTemplateID      string    `json:"agent_template_id"`
	ProviderConfigID     string    `json:"provider_config_id"`
	PromptIDs            []string  `json:"prompt_ids,omitempty"`
	SkillIDs             []string  `json:"skill_ids,omitempty"`
	ToolIDs              []string  `json:"tool_ids,omitempty"`
	SystemPromptOverride string    `json:"system_prompt_override,omitempty"`
	Sequence             int       `json:"sequence"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
type Instance struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	TeamID      string    `json:"team_id,omitempty"`
	ContainerID string    `json:"container_id"`
	HostPort    int       `json:"host_port"`
	Status      string    `json:"status"`
	ErrorMsg    string    `json:"error_msg,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChatMessage struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instance_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	ToolCall   string    `json:"tool_call,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Resource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProviderConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	ModelID   string    `json:"model_id"`
	APIKey    string    `json:"api_key,omitempty"`
	BaseURL   string    `json:"base_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
