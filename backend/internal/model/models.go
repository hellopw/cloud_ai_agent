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

type Template struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	DockerfileContent string    `json:"dockerfile_content"`
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

type Instance struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agent_id"`
	ContainerID string    `json:"container_id"`
	HostPort    int       `json:"host_port"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Resource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}