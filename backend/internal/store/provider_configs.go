package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListProviderConfigs() ([]model.ProviderConfig, error) {
	rows, err := s.db.Query("SELECT id, name, provider, model_id, api_key, base_url, created_at, updated_at FROM provider_configs ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]model.ProviderConfig, 0)
	for rows.Next() {
		var pc model.ProviderConfig
		var baseURL sql.NullString
		if err := rows.Scan(&pc.ID, &pc.Name, &pc.Provider, &pc.ModelID, &pc.APIKey, &baseURL, &pc.CreatedAt, &pc.UpdatedAt); err != nil {
			return nil, err
		}
		if baseURL.Valid {
			pc.BaseURL = baseURL.String
		}
		configs = append(configs, pc)
	}
	return configs, rows.Err()
}

func (s *Store) GetProviderConfig(id string) (*model.ProviderConfig, error) {
	var pc model.ProviderConfig
	var baseURL sql.NullString
	err := s.db.QueryRow("SELECT id, name, provider, model_id, api_key, base_url, created_at, updated_at FROM provider_configs WHERE id = ?", id).
		Scan(&pc.ID, &pc.Name, &pc.Provider, &pc.ModelID, &pc.APIKey, &baseURL, &pc.CreatedAt, &pc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if baseURL.Valid {
		pc.BaseURL = baseURL.String
	}
	return &pc, nil
}

func (s *Store) CreateProviderConfig(pc *model.ProviderConfig) error {
	pc.ID = uuid.New().String()
	pc.CreatedAt = time.Now()
	pc.UpdatedAt = time.Now()
	baseURL := sql.NullString{String: pc.BaseURL, Valid: pc.BaseURL != ""}
	_, err := s.db.Exec(
		"INSERT INTO provider_configs (id, name, provider, model_id, api_key, base_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		pc.ID, pc.Name, pc.Provider, pc.ModelID, pc.APIKey, baseURL, pc.CreatedAt, pc.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateProviderConfig(pc *model.ProviderConfig) error {
	pc.UpdatedAt = time.Now()
	baseURL := sql.NullString{String: pc.BaseURL, Valid: pc.BaseURL != ""}
	_, err := s.db.Exec(
		"UPDATE provider_configs SET name=?, provider=?, model_id=?, api_key=?, base_url=?, updated_at=? WHERE id=?",
		pc.Name, pc.Provider, pc.ModelID, pc.APIKey, baseURL, pc.UpdatedAt, pc.ID,
	)
	return err
}

func (s *Store) DeleteProviderConfig(id string) error {
	_, err := s.db.Exec("DELETE FROM provider_configs WHERE id=?", id)
	return err
}
