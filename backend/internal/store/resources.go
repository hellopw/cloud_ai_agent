package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListResources() ([]model.Resource, error) {
	rows, err := s.db.Query("SELECT id, name, type, config, created_at, updated_at FROM resources ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := make([]model.Resource, 0)
	for rows.Next() {
		var r model.Resource
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Config, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

func (s *Store) GetResource(id string) (*model.Resource, error) {
	var r model.Resource
	err := s.db.QueryRow("SELECT id, name, type, config, created_at, updated_at FROM resources WHERE id = ?", id).
		Scan(&r.ID, &r.Name, &r.Type, &r.Config, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) CreateResource(r *model.Resource) error {
	r.ID = uuid.New().String()
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO resources (id, name, type, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		r.ID, r.Name, r.Type, r.Config, r.CreatedAt, r.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateResource(r *model.Resource) error {
	r.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE resources SET name=?, type=?, config=?, updated_at=? WHERE id=?",
		r.Name, r.Type, r.Config, r.UpdatedAt, r.ID,
	)
	return err
}

func (s *Store) DeleteResource(id string) error {
	_, err := s.db.Exec("DELETE FROM resources WHERE id=?", id)
	return err
}
