package store

import (
	"cloud_ai_agent/internal/model"
)

func (s *Store) CreateChatMessage(msg *model.ChatMessage) error {
	_, err := s.db.Exec(
		`INSERT INTO chat_messages (id, instance_id, role, content, tool_call)
		 VALUES (?, ?, ?, ?, ?)`,
		msg.ID, msg.InstanceID, msg.Role, msg.Content, msg.ToolCall,
	)
	return err
}

func (s *Store) ListChatMessages(instanceID string) ([]model.ChatMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, instance_id, role, content, COALESCE(tool_call,''), created_at
		 FROM chat_messages WHERE instance_id = ? ORDER BY created_at ASC`, instanceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []model.ChatMessage
	for rows.Next() {
		var m model.ChatMessage
		if err := rows.Scan(&m.ID, &m.InstanceID, &m.Role, &m.Content, &m.ToolCall, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}
