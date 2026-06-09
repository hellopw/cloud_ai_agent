package store

import (
	"testing"

	"cloud_ai_agent/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if err := s.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPromptCRUD(t *testing.T) {
	s := newTestStore(t)

	p := &model.Prompt{Name: "test", Description: "desc", Content: "hello"}
	if err := s.CreatePrompt(p); err != nil {
		t.Fatalf("CreatePrompt: %v", err)
	}

	got, err := s.GetPrompt(p.ID)
	if err != nil || got == nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}

	got.Description = "updated"
	if err := s.UpdatePrompt(got); err != nil {
		t.Fatalf("UpdatePrompt: %v", err)
	}

	all, err := s.ListPrompts()
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("ListPrompts len = %d, want 1", len(all))
	}

	if err := s.DeletePrompt(p.ID); err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}

	all, _ = s.ListPrompts()
	if len(all) != 0 {
		t.Errorf("ListPrompts len after delete = %d, want 0", len(all))
	}
}

func TestTemplateBindings(t *testing.T) {
	s := newTestStore(t)

	p := &model.Prompt{Name: "p1", Content: "test"}
	s.CreatePrompt(p)

	sk := &model.Skill{Name: "s1", Content: "skill"}
	s.CreateSkill(sk)

	tmpl := &model.Template{Name: "t1"}
	if err := s.CreateTemplate(tmpl); err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	if err := s.UpdateTemplateBindings(tmpl.ID, []string{p.ID}, []string{sk.ID}, nil); err != nil {
		t.Fatalf("UpdateTemplateBindings: %v", err)
	}

	got, err := s.GetTemplate(tmpl.ID)
	if err != nil {
		t.Fatalf("GetTemplate: %v", err)
	}
	if len(got.PromptIDs) != 1 || got.PromptIDs[0] != p.ID {
		t.Errorf("PromptIDs = %v, want [%s]", got.PromptIDs, p.ID)
	}
	if len(got.SkillIDs) != 1 || got.SkillIDs[0] != sk.ID {
		t.Errorf("SkillIDs = %v, want [%s]", got.SkillIDs, sk.ID)
	}
}
