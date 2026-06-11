package store

import (
	"database/sql"
	"time"

	"cloud_ai_agent/internal/model"

	"github.com/google/uuid"
)

func (s *Store) ListAgentTeams() ([]model.AgentTeam, error) {
	rows, err := s.db.Query("SELECT id, name, template_id, repo_url, COALESCE(git_username,''), COALESCE(git_password,''), branch, image_tag, status, COALESCE(error_msg,''), created_at, updated_at FROM agent_teams ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := make([]model.AgentTeam, 0)
	for rows.Next() {
		var t model.AgentTeam
		if err := rows.Scan(&t.ID, &t.Name, &t.TemplateID, &t.RepoURL, &t.GitUsername, &t.GitPassword, &t.Branch, &t.ImageTag, &t.Status, &t.ErrorMsg, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (s *Store) GetAgentTeam(id string) (*model.AgentTeam, error) {
	var t model.AgentTeam
	err := s.db.QueryRow("SELECT id, name, template_id, repo_url, COALESCE(git_username,''), COALESCE(git_password,''), branch, image_tag, status, COALESCE(error_msg,''), created_at, updated_at FROM agent_teams WHERE id = ?", id).
		Scan(&t.ID, &t.Name, &t.TemplateID, &t.RepoURL, &t.GitUsername, &t.GitPassword, &t.Branch, &t.ImageTag, &t.Status, &t.ErrorMsg, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.loadTeamBindings(&t); err != nil {
		return nil, err
	}
	if err := s.loadTeamMembers(&t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) loadTeamBindings(t *model.AgentTeam) error {
	rows, err := s.db.Query("SELECT prompt_id FROM agent_team_prompts WHERE team_id = ?", t.ID)
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

	rows2, err := s.db.Query("SELECT skill_id FROM agent_team_skills WHERE team_id = ?", t.ID)
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
	return nil
}

func (s *Store) loadTeamMembers(t *model.AgentTeam) error {
	rows, err := s.db.Query("SELECT id, team_id, name, role, COALESCE(agent_template_id,''), COALESCE(provider_config_id,''), COALESCE(system_prompt_override,''), sequence, created_at, updated_at FROM team_members WHERE team_id = ? ORDER BY sequence", t.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var m model.TeamMember
		if err := rows.Scan(&m.ID, &m.TeamID, &m.Name, &m.Role, &m.AgentTemplateID, &m.ProviderConfigID, &m.SystemPromptOverride, &m.Sequence, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return err
		}

		// load member prompts
		pr, err := s.db.Query("SELECT prompt_id FROM team_member_prompts WHERE team_member_id = ?", m.ID)
		if err != nil {
			return err
		}
		for pr.Next() {
			var pid string
			pr.Scan(&pid)
			m.PromptIDs = append(m.PromptIDs, pid)
		}
		pr.Close()

		// load member skills
		sk, err := s.db.Query("SELECT skill_id FROM team_member_skills WHERE team_member_id = ?", m.ID)
		if err != nil {
			return err
		}
		for sk.Next() {
			var sid string
			sk.Scan(&sid)
			m.SkillIDs = append(m.SkillIDs, sid)
		}
		sk.Close()

		// load member tools
		tl, err := s.db.Query("SELECT tool_id FROM team_member_tools WHERE team_member_id = ?", m.ID)
		if err != nil {
			return err
		}
		for tl.Next() {
			var tid string
			tl.Scan(&tid)
			m.ToolIDs = append(m.ToolIDs, tid)
		}
		tl.Close()

		t.Members = append(t.Members, m)
	}
	return rows.Err()
}

func (s *Store) CreateAgentTeam(t *model.AgentTeam) error {
	t.ID = uuid.New().String()
	t.Status = "draft"
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"INSERT INTO agent_teams (id, name, template_id, repo_url, git_username, git_password, branch, image_tag, status, error_msg, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.Name, t.TemplateID, t.RepoURL, t.GitUsername, t.GitPassword, t.Branch, t.ImageTag, t.Status, t.ErrorMsg, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return err
	}

	if err := s.insertTeamMembers(tx, t.ID, t.Members); err != nil {
		return err
	}

	if err := s.insertTeamBindings(tx, t.ID, t.PromptIDs, t.SkillIDs); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) insertTeamMembers(tx *sql.Tx, teamID string, members []model.TeamMember) error {
	for i := range members {
		m := &members[i]
		m.ID = uuid.New().String()
		m.TeamID = teamID
		m.CreatedAt = time.Now()
		m.UpdatedAt = time.Now()
		_, err := tx.Exec(
			"INSERT INTO team_members (id, team_id, name, role, agent_template_id, provider_config_id, system_prompt_override, sequence, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			m.ID, m.TeamID, m.Name, m.Role, m.AgentTemplateID, m.ProviderConfigID, m.SystemPromptOverride, m.Sequence, m.CreatedAt, m.UpdatedAt,
		)
		if err != nil {
			return err
		}

		for _, pid := range m.PromptIDs {
			tx.Exec("INSERT OR IGNORE INTO team_member_prompts (team_member_id, prompt_id) VALUES (?, ?)", m.ID, pid)
		}
		for _, sid := range m.SkillIDs {
			tx.Exec("INSERT OR IGNORE INTO team_member_skills (team_member_id, skill_id) VALUES (?, ?)", m.ID, sid)
		}
		for _, tid := range m.ToolIDs {
			tx.Exec("INSERT OR IGNORE INTO team_member_tools (team_member_id, tool_id) VALUES (?, ?)", m.ID, tid)
		}
	}
	return nil
}

func (s *Store) insertTeamBindings(tx *sql.Tx, teamID string, promptIDs, skillIDs []string) error {
	for _, pid := range promptIDs {
		tx.Exec("INSERT OR IGNORE INTO agent_team_prompts (team_id, prompt_id) VALUES (?, ?)", teamID, pid)
	}
	for _, sid := range skillIDs {
		tx.Exec("INSERT OR IGNORE INTO agent_team_skills (team_id, skill_id) VALUES (?, ?)", teamID, sid)
	}
	return nil
}

func (s *Store) UpdateAgentTeamStatus(id, status, imageTag, errorMsg string) error {
	_, err := s.db.Exec(
		"UPDATE agent_teams SET status=?, image_tag=COALESCE(NULLIF(?, ''), image_tag), error_msg=?, updated_at=? WHERE id=?",
		status, imageTag, errorMsg, time.Now(), id,
	)
	return err
}

func (s *Store) UpdateAgentTeam(t *model.AgentTeam) error {
	t.UpdatedAt = time.Now()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		"UPDATE agent_teams SET name=?, template_id=?, repo_url=?, git_username=?, git_password=?, branch=?, updated_at=? WHERE id=?",
		t.Name, t.TemplateID, t.RepoURL, t.GitUsername, t.GitPassword, t.Branch, t.UpdatedAt, t.ID,
	)
	if err != nil {
		return err
	}

	// Delete and re-insert members and their bindings
	tx.Exec("DELETE FROM team_members WHERE team_id = ?", t.ID)

	// Delete old team-level bindings
	tx.Exec("DELETE FROM agent_team_prompts WHERE team_id = ?", t.ID)
	tx.Exec("DELETE FROM agent_team_skills WHERE team_id = ?", t.ID)

	if err := s.insertTeamMembers(tx, t.ID, t.Members); err != nil {
		return err
	}
	if err := s.insertTeamBindings(tx, t.ID, t.PromptIDs, t.SkillIDs); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) DeleteAgentTeam(id string) error {
	_, err := s.db.Exec("DELETE FROM agent_teams WHERE id=?", id)
	return err
}
