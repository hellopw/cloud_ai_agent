package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListTools() ([]model.Tool, error) {
	rows, err := s.db.Query("SELECT id, name, label, description, dsl_definition, created_at, updated_at FROM tools ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make([]model.Tool, 0)
	for rows.Next() {
		var t model.Tool
		if err := rows.Scan(&t.ID, &t.Name, &t.Label, &t.Description, &t.DSLDefinition, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

func (s *Store) GetTool(id string) (*model.Tool, error) {
	var t model.Tool
	err := s.db.QueryRow("SELECT id, name, label, description, dsl_definition, created_at, updated_at FROM tools WHERE id = ?", id).
		Scan(&t.ID, &t.Name, &t.Label, &t.Description, &t.DSLDefinition, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) CreateTool(t *model.Tool) error {
	t.ID = uuid.New().String()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO tools (id, name, label, description, dsl_definition, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.Name, t.Label, t.Description, t.DSLDefinition, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateTool(t *model.Tool) error {
	t.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE tools SET name=?, label=?, description=?, dsl_definition=?, updated_at=? WHERE id=?",
		t.Name, t.Label, t.Description, t.DSLDefinition, t.UpdatedAt, t.ID,
	)
	return err
}

func (s *Store) DeleteTool(id string) error {
	_, err := s.db.Exec("DELETE FROM tools WHERE id=?", id)
	return err
}
