package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListPrompts() ([]model.Prompt, error) {
	rows, err := s.db.Query("SELECT id, name, description, content, created_at, updated_at FROM prompts ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prompts := make([]model.Prompt, 0)
	for rows.Next() {
		var p model.Prompt
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Content, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		prompts = append(prompts, p)
	}
	return prompts, rows.Err()
}

func (s *Store) GetPrompt(id string) (*model.Prompt, error) {
	var p model.Prompt
	err := s.db.QueryRow("SELECT id, name, description, content, created_at, updated_at FROM prompts WHERE id = ?", id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Content, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) CreatePrompt(p *model.Prompt) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO prompts (id, name, description, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.Description, p.Content, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (s *Store) UpdatePrompt(p *model.Prompt) error {
	p.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE prompts SET name=?, description=?, content=?, updated_at=? WHERE id=?",
		p.Name, p.Description, p.Content, p.UpdatedAt, p.ID,
	)
	return err
}

func (s *Store) DeletePrompt(id string) error {
	_, err := s.db.Exec("DELETE FROM prompts WHERE id=?", id)
	return err
}
