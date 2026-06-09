package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListSkills() ([]model.Skill, error) {
	rows, err := s.db.Query("SELECT id, name, description, content, created_at, updated_at FROM skills ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []model.Skill
	for rows.Next() {
		var sk model.Skill
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Content, &sk.CreatedAt, &sk.UpdatedAt); err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

func (s *Store) GetSkill(id string) (*model.Skill, error) {
	var sk model.Skill
	err := s.db.QueryRow("SELECT id, name, description, content, created_at, updated_at FROM skills WHERE id = ?", id).
		Scan(&sk.ID, &sk.Name, &sk.Description, &sk.Content, &sk.CreatedAt, &sk.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sk, nil
}

func (s *Store) CreateSkill(sk *model.Skill) error {
	sk.ID = uuid.New().String()
	sk.CreatedAt = time.Now()
	sk.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO skills (id, name, description, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		sk.ID, sk.Name, sk.Description, sk.Content, sk.CreatedAt, sk.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateSkill(sk *model.Skill) error {
	sk.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE skills SET name=?, description=?, content=?, updated_at=? WHERE id=?",
		sk.Name, sk.Description, sk.Content, sk.UpdatedAt, sk.ID,
	)
	return err
}

func (s *Store) DeleteSkill(id string) error {
	_, err := s.db.Exec("DELETE FROM skills WHERE id=?", id)
	return err
}
