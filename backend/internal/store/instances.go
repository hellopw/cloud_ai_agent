package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListInstances() ([]model.Instance, error) {
	rows, err := s.db.Query("SELECT id, agent_id, container_id, host_port, status, created_at, updated_at FROM instances ORDER BY created_at DESC")
	if err != nil { return nil, err }
	defer rows.Close()
	instances := make([]model.Instance, 0)
	for rows.Next() {
		var i model.Instance
		if err := rows.Scan(&i.ID, &i.AgentID, &i.ContainerID, &i.HostPort, &i.Status, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		instances = append(instances, i)
	}
	return instances, rows.Err()
}

func (s *Store) GetInstance(id string) (*model.Instance, error) {
	var i model.Instance
	err := s.db.QueryRow("SELECT id, agent_id, container_id, host_port, status, created_at, updated_at FROM instances WHERE id = ?", id).
		Scan(&i.ID, &i.AgentID, &i.ContainerID, &i.HostPort, &i.Status, &i.CreatedAt, &i.UpdatedAt)
	if err == sql.ErrNoRows { return nil, nil }
	if err != nil { return nil, err }
	rows, err := s.db.Query("SELECT memory_id FROM instance_memories WHERE instance_id = ?", id)
	if err != nil { return &i, nil }
	defer rows.Close()
	for rows.Next() {
		var mid string
		if err := rows.Scan(&mid); err == nil { i.MemoryIDs = append(i.MemoryIDs, mid) }
	}
	return &i, nil
}

func (s *Store) CreateInstance(i *model.Instance) error {
	i.ID = uuid.New().String()
	i.Status = "starting"
	i.CreatedAt = time.Now()
	i.UpdatedAt = time.Now()
	_, err := s.db.Exec(
		"INSERT INTO instances (id, agent_id, container_id, host_port, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		i.ID, i.AgentID, i.ContainerID, i.HostPort, i.Status, i.CreatedAt, i.UpdatedAt,
	)
	if err != nil { return err }
	for _, mid := range i.MemoryIDs {
		s.db.Exec("INSERT OR IGNORE INTO instance_memories (instance_id, memory_id) VALUES (?, ?)", i.ID, mid)
	}
	return nil
}

func (s *Store) UpdateInstanceStatus(id, status, containerID string, hostPort int) error {
	_, err := s.db.Exec(
		"UPDATE instances SET status=?, container_id=COALESCE(NULLIF(?, ''), container_id), host_port=CASE WHEN ? > 0 THEN ? ELSE host_port END, updated_at=? WHERE id=?",
		status, containerID, hostPort, hostPort, time.Now(), id,
	)
	return err
}

func (s *Store) DeleteInstance(id string) error {
	s.db.Exec("DELETE FROM instance_memories WHERE instance_id = ?", id)
	_, err := s.db.Exec("DELETE FROM instances WHERE id=?", id)
	return err
}
