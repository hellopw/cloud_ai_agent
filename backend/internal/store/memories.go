package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListMemories() ([]model.Memory, error) {
	rows, err := s.db.Query("SELECT id, name, description, content, source, created_at, updated_at FROM memories ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memories := make([]model.Memory, 0)
	for rows.Next() {
		var m model.Memory
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (s *Store) GetMemory(id string) (*model.Memory, error) {
	var m model.Memory
	err := s.db.QueryRow("SELECT id, name, description, content, source, created_at, updated_at FROM memories WHERE id = ?", id).
		Scan(&m.ID, &m.Name, &m.Description, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) CreateMemory(m *model.Memory) error {
	m.ID = uuid.New().String()
	if m.Source == "" {
		m.Source = "manual"
	}
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO memories (id, name, description, content, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.Name, m.Description, m.Content, m.Source, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateMemory(m *model.Memory) error {
	m.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"UPDATE memories SET name=?, description=?, content=?, source=?, updated_at=? WHERE id=?",
		m.Name, m.Description, m.Content, m.Source, m.UpdatedAt, m.ID,
	)
	return err
}

func (s *Store) DeleteMemory(id string) error {
	_, err := s.db.Exec("DELETE FROM memories WHERE id=?", id)
	return err
}

func (s *Store) GetMemoriesByIDs(ids []string) ([]model.Memory, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := "SELECT id, name, description, content, source, created_at, updated_at FROM memories WHERE id IN ("
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id
	}
	query += ") ORDER BY created_at DESC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memories []model.Memory
	for rows.Next() {
		var m model.Memory
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (s *Store) LinkInstanceMemory(instanceID, memoryID string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO instance_memories (instance_id, memory_id) VALUES (?, ?)", instanceID, memoryID)
	return err
}

func (s *Store) GetInstanceMemories(instanceID string) ([]model.Memory, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.name, m.description, m.content, m.source, m.created_at, m.updated_at
		 FROM memories m INNER JOIN instance_memories im ON im.memory_id = m.id
		 WHERE im.instance_id = ? ORDER BY m.created_at DESC`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var memories []model.Memory
	for rows.Next() {
		var m model.Memory
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Content, &m.Source, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}
