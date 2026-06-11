package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListTemplates() ([]model.Template, error) {
	rows, err := s.db.Query("SELECT id, name, description, agent_type, dockerfile_content, created_at, updated_at FROM templates ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		var t model.Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.AgentType, &t.DockerfileContent, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

func (s *Store) GetTemplate(id string) (*model.Template, error) {
	var t model.Template
	err := s.db.QueryRow("SELECT id, name, description, agent_type, dockerfile_content, created_at, updated_at FROM templates WHERE id = ?", id).
		Scan(&t.ID, &t.Name, &t.Description, &t.AgentType, &t.DockerfileContent, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadTemplateBindings(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) loadTemplateBindings(t *model.Template) error {
	rows, err := s.db.Query("SELECT prompt_id FROM template_prompts WHERE template_id = ?", t.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		t.PromptIDs = append(t.PromptIDs, id)
	}

	rows2, err := s.db.Query("SELECT skill_id FROM template_skills WHERE template_id = ?", t.ID)
	if err != nil {
		return err
	}
	defer rows2.Close()
	for rows2.Next() {
		var id string
		if err := rows2.Scan(&id); err != nil {
			return err
		}
		t.SkillIDs = append(t.SkillIDs, id)
	}

	rows3, err := s.db.Query("SELECT tool_id FROM template_tools WHERE template_id = ?", t.ID)
	if err != nil {
		return err
	}
	defer rows3.Close()
	for rows3.Next() {
		var id string
		if err := rows3.Scan(&id); err != nil {
			return err
		}
		t.ToolIDs = append(t.ToolIDs, id)
	}
	return nil
}

func (s *Store) CreateTemplate(t *model.Template) error {
	t.ID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO templates (id, name, description, agent_type, dockerfile_content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.Name, t.Description, t.AgentType, t.DockerfileContent, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateTemplate(t *model.Template) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE templates SET name=?, description=?, agent_type=?, dockerfile_content=?, updated_at=? WHERE id=?",
		t.Name, t.Description, t.AgentType, t.DockerfileContent, t.UpdatedAt, t.ID,
	)
	return err
}

func (s *Store) DeleteTemplate(id string) error {
	_, err := s.db.Exec("DELETE FROM templates WHERE id=?", id)
	return err
}

func (s *Store) UpdateTemplateBindings(templateID string, promptIDs, skillIDs, toolIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tx.Exec("DELETE FROM template_prompts WHERE template_id = ?", templateID)
	tx.Exec("DELETE FROM template_skills WHERE template_id = ?", templateID)
	tx.Exec("DELETE FROM template_tools WHERE template_id = ?", templateID)

	for _, pid := range promptIDs {
		tx.Exec("INSERT OR IGNORE INTO template_prompts (template_id, prompt_id) VALUES (?, ?)", templateID, pid)
	}
	for _, sid := range skillIDs {
		tx.Exec("INSERT OR IGNORE INTO template_skills (template_id, skill_id) VALUES (?, ?)", templateID, sid)
	}
	for _, tid := range toolIDs {
		tx.Exec("INSERT OR IGNORE INTO template_tools (template_id, tool_id) VALUES (?, ?)", templateID, tid)
	}

	return tx.Commit()
}
