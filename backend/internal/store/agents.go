package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListAgents() ([]model.Agent, error) {
	rows, err := s.db.Query("SELECT id, name, template_id, repo_url, COALESCE(git_username,''), COALESCE(git_password,''), branch, image_tag, status, COALESCE(error_msg,''), created_at, updated_at FROM agents ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := make([]model.Agent, 0)
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.TemplateID, &a.RepoURL, &a.GitUsername, &a.GitPassword, &a.Branch, &a.ImageTag, &a.Status, &a.ErrorMsg, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, rows.Err()
}

func (s *Store) GetAgent(id string) (*model.Agent, error) {
	var a model.Agent
	err := s.db.QueryRow("SELECT id, name, template_id, repo_url, COALESCE(git_username,''), COALESCE(git_password,''), branch, image_tag, status, COALESCE(error_msg,''), created_at, updated_at FROM agents WHERE id = ?", id).
		Scan(&a.ID, &a.Name, &a.TemplateID, &a.RepoURL, &a.GitUsername, &a.GitPassword, &a.Branch, &a.ImageTag, &a.Status, &a.ErrorMsg, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) CreateAgent(a *model.Agent) error {
	a.ID = uuid.New().String()
	a.Status = "draft"
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO agents (id, name, template_id, repo_url, git_username, git_password, branch, image_tag, status, error_msg, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		a.ID, a.Name, a.TemplateID, a.RepoURL, a.GitUsername, a.GitPassword, a.Branch, a.ImageTag, a.Status, a.ErrorMsg, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateAgentStatus(id, status, imageTag, errorMsg string) error {
	_, err := s.db.Exec(
		"UPDATE agents SET status=?, image_tag=COALESCE(NULLIF(?, ''), image_tag), error_msg=?, updated_at=? WHERE id=?",
		status, imageTag, errorMsg, time.Now(), id,
	)
	return err
}

func (s *Store) UpdateAgent(a *model.Agent) error {
	a.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE agents SET name=?, template_id=?, repo_url=?, git_username=?, git_password=?, branch=?, updated_at=? WHERE id=?",
		a.Name, a.TemplateID, a.RepoURL, a.GitUsername, a.GitPassword, a.Branch, a.UpdatedAt, a.ID,
	)
	return err
}

func (s *Store) DeleteAgent(id string) error {
	_, err := s.db.Exec("DELETE FROM agents WHERE id=?", id)
	return err
}
