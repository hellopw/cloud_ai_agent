package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud_ai_agent/internal/model"
	"cloud_ai_agent/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := NewHandler(s)
	ts := httptest.NewServer(NewRouter(h))
	t.Cleanup(func() { ts.Close() })

	return ts, s
}

func doJSON(t *testing.T, method, url string, body interface{}) *http.Response {
	t.Helper()
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func readResp(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return data
}

func decodeJSONResp[T any](t *testing.T, data []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("Unmarshal: %v (body=%s)", err, string(data))
	}
	return v
}

func TestAgentTeamsAPI_CRUD(t *testing.T) {
	ts, store := newTestServer(t)

	// Setup dependencies
	tmpl := &model.Template{Name: "test-template"}
	if err := store.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	pc := &model.ProviderConfig{Name: "test-config", Provider: "openai-codex", ModelID: "gpt-5"}
	if err := store.CreateProviderConfig(pc); err != nil {
		t.Fatalf("CreateProviderConfig: %v", err)
	}

	prompt := &model.Prompt{Name: "team-prompt", Content: "test"}
	if err := store.CreatePrompt(prompt); err != nil {
		t.Fatalf("CreatePrompt: %v", err)
	}

	skill := &model.Skill{Name: "team-skill", Content: "test"}
	if err := store.CreateSkill(skill); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	// --- CREATE team ---
	createBody := model.AgentTeam{
		Name:       "test-team",
		TemplateID: tmpl.ID,
		RepoURL:    "https://github.com/test/repo.git",
		Branch:     "main",
		PromptIDs:  []string{prompt.ID},
		SkillIDs:   []string{skill.ID},
		Members: []model.TeamMember{
			{
				Name:             "leader",
				Role:             "leader",
				AgentTemplateID:  tmpl.ID,
				ProviderConfigID: pc.ID,
				Sequence:         0,
			},
			{
				Name:                 "code-reviewer",
				Role:                 "worker",
				AgentTemplateID:      tmpl.ID,
				ProviderConfigID:     pc.ID,
				SystemPromptOverride: "You are a code reviewer",
				Sequence:             1,
			},
		},
	}

	resp := doJSON(t, "POST", ts.URL+"/api/agent-teams", createBody)
	data := readResp(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", resp.StatusCode, string(data))
	}
	created := decodeJSONResp[model.AgentTeam](t, data)
	if created.ID == "" {
		t.Fatal("created team id is empty")
	}
	if created.Status != "draft" {
		t.Errorf("status = %q, want draft", created.Status)
	}
	if len(created.Members) != 2 {
		t.Fatalf("members len = %d, want 2", len(created.Members))
	}

	// --- LIST teams ---
	resp = doJSON(t, "GET", ts.URL+"/api/agent-teams", nil)
	data = readResp(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET list status = %d, body = %s", resp.StatusCode, string(data))
	}
	teams := decodeJSONResp[[]model.AgentTeam](t, data)
	if len(teams) != 1 {
		t.Fatalf("list len = %d, want 1", len(teams))
	}
	if teams[0].Name != "test-team" {
		t.Errorf("list[0].Name = %q, want test-team", teams[0].Name)
	}

	// --- GET team detail ---
	resp = doJSON(t, "GET", ts.URL+"/api/agent-teams/"+created.ID, nil)
	data = readResp(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET detail status = %d, body = %s", resp.StatusCode, string(data))
	}
	got := decodeJSONResp[model.AgentTeam](t, data)
	if got.Name != "test-team" {
		t.Errorf("Name = %q", got.Name)
	}
	if len(got.Members) != 2 {
		t.Fatalf("members len = %d, want 2", len(got.Members))
	}
	if len(got.PromptIDs) != 1 || got.PromptIDs[0] != prompt.ID {
		t.Errorf("PromptIDs = %v, want [%s]", got.PromptIDs, prompt.ID)
	}
	if len(got.SkillIDs) != 1 || got.SkillIDs[0] != skill.ID {
		t.Errorf("SkillIDs = %v, want [%s]", got.SkillIDs, skill.ID)
	}

	// Verify leader and worker
	leader := got.Members[0]
	if leader.Name != "leader" || leader.Role != "leader" {
		t.Errorf("leader = %s (%s), want leader (leader)", leader.Name, leader.Role)
	}
	worker := got.Members[1]
	if worker.Name != "code-reviewer" || worker.Role != "worker" {
		t.Errorf("worker = %s (%s), want code-reviewer (worker)", worker.Name, worker.Role)
	}
	if worker.SystemPromptOverride != "You are a code reviewer" {
		t.Errorf("system_prompt_override = %q", worker.SystemPromptOverride)
	}

	// --- UPDATE team ---
	updateBody := model.AgentTeam{
		Name:       "test-team-renamed",
		TemplateID: tmpl.ID,
		RepoURL:    "https://github.com/test/repo2.git",
		Branch:     "develop",
		Members: []model.TeamMember{
			{
				Name:             "new-leader",
				Role:             "leader",
				AgentTemplateID:  tmpl.ID,
				ProviderConfigID: pc.ID,
				Sequence:         0,
			},
		},
	}

	resp = doJSON(t, "PUT", ts.URL+"/api/agent-teams/"+created.ID, updateBody)
	data = readResp(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", resp.StatusCode, string(data))
	}
	updated := decodeJSONResp[model.AgentTeam](t, data)
	if updated.Name != "test-team-renamed" {
		t.Errorf("updated name = %q", updated.Name)
	}
	if updated.Branch != "develop" {
		t.Errorf("updated branch = %q", updated.Branch)
	}
	if len(updated.Members) != 1 {
		t.Fatalf("updated members len = %d, want 1", len(updated.Members))
	}
	if updated.Members[0].Name != "new-leader" {
		t.Errorf("updated member name = %q", updated.Members[0].Name)
	}

	// --- DELETE team ---
	resp = doJSON(t, "DELETE", ts.URL+"/api/agent-teams/"+created.ID, nil)
	data = readResp(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d, body = %s", resp.StatusCode, string(data))
	}

	// Verify gone
	resp = doJSON(t, "GET", ts.URL+"/api/agent-teams/"+created.ID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after delete status = %d, want 404", resp.StatusCode)
	}

	// List should be empty
	resp = doJSON(t, "GET", ts.URL+"/api/agent-teams", nil)
	data = readResp(t, resp)
	teams = decodeJSONResp[[]model.AgentTeam](t, data)
	if len(teams) != 0 {
		t.Errorf("list len after delete = %d, want 0", len(teams))
	}
}

func TestAgentTeamsAPI_Validation(t *testing.T) {
	ts, _ := newTestServer(t)

	// GET non-existent team
	resp := doJSON(t, "GET", ts.URL+"/api/agent-teams/nonexistent", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET nonexistent status = %d, want 404", resp.StatusCode)
	}

	// PUT non-existent team
	resp = doJSON(t, "PUT", ts.URL+"/api/agent-teams/nonexistent", model.AgentTeam{Name: "x"})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("PUT nonexistent status = %d, want 404", resp.StatusCode)
	}
}

func TestAgentTeamsAPI_CannotEditAfterReady(t *testing.T) {
	ts, store := newTestServer(t)

	tmpl := &model.Template{Name: "ready-template"}
	store.CreateTemplate(tmpl)

	pc := &model.ProviderConfig{Name: "pc", Provider: "openai-codex", ModelID: "gpt"}
	store.CreateProviderConfig(pc)

	// Insert a team and set its status to "ready"
	team := &model.AgentTeam{
		Name:       "ready-team",
		TemplateID: tmpl.ID,
		RepoURL:    "https://github.com/test/repo.git",
		Branch:     "main",
		Members: []model.TeamMember{{
			Name:             "leader",
			Role:             "leader",
			AgentTemplateID:  tmpl.ID,
			ProviderConfigID: pc.ID,
			Sequence:         0,
		}},
	}
	if err := store.CreateAgentTeam(team); err != nil {
		t.Fatalf("CreateAgentTeam: %v", err)
	}
	store.UpdateAgentTeamStatus(team.ID, "ready", "some-image:latest", "")

	// Attempt to edit a ready team
	resp := doJSON(t, "PUT", ts.URL+"/api/agent-teams/"+team.ID, model.AgentTeam{Name: "changed"})
	data := readResp(t, resp)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("PUT ready team status = %d, body = %s", resp.StatusCode, string(data))
	}
}

func TestAgentTeamsAPI_DeleteCleansUp(t *testing.T) {
	ts, store := newTestServer(t)

	tmpl := &model.Template{Name: "cleanup-template"}
	store.CreateTemplate(tmpl)

	pc := &model.ProviderConfig{Name: "pc", Provider: "openai-codex", ModelID: "gpt"}
	store.CreateProviderConfig(pc)

	prompt := &model.Prompt{Name: "p1", Content: "test"}
	store.CreatePrompt(prompt)

	skill := &model.Skill{Name: "s1", Content: "test"}
	store.CreateSkill(skill)

	tool := &model.Tool{Name: "t1", DSLDefinition: "{}"}
	store.CreateTool(tool)

	team := &model.AgentTeam{
		Name:       "cleanup-team",
		TemplateID: tmpl.ID,
		RepoURL:    "https://github.com/test/repo.git",
		Branch:     "main",
		PromptIDs:  []string{prompt.ID},
		SkillIDs:   []string{skill.ID},
		Members: []model.TeamMember{
			{
				Name:             "member1",
				Role:             "leader",
				AgentTemplateID:  tmpl.ID,
				ProviderConfigID: pc.ID,
				PromptIDs:        []string{prompt.ID},
				SkillIDs:         []string{skill.ID},
				ToolIDs:          []string{tool.ID},
				Sequence:         0,
			},
		},
	}
	if err := store.CreateAgentTeam(team); err != nil {
		t.Fatalf("CreateAgentTeam: %v", err)
	}

	// Verify it exists
	resp := doJSON(t, "GET", ts.URL+"/api/agent-teams/"+team.ID, nil)
	data := readResp(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET after create: status = %d, body = %s", resp.StatusCode, string(data))
	}

	// Delete
	resp = doJSON(t, "DELETE", ts.URL+"/api/agent-teams/"+team.ID, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status = %d", resp.StatusCode)
	}

	// Verify gone
	resp = doJSON(t, "GET", ts.URL+"/api/agent-teams/"+team.ID, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after delete status = %d, want 404", resp.StatusCode)
	}
}
