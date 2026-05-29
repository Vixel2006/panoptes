package app_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Vixel2006/panoptes/internal/app"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

type mockSessionRepo struct {
	mu   sync.Mutex
	data map[string]*model.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{data: make(map[string]*model.Session)}
}

func (m *mockSessionRepo) Create(_ context.Context, s *model.Session) error {
	m.mu.Lock()
	m.data[s.ID] = s
	m.mu.Unlock()
	return nil
}

func (m *mockSessionRepo) GetByID(_ context.Context, id string) (*model.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.data[id]
	if !ok {
		return nil, &notFoundError{"session"}
	}
	return s, nil
}

func (m *mockSessionRepo) List(_ context.Context) ([]*model.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*model.Session, 0, len(m.data))
	for _, s := range m.data {
		out = append(out, s)
	}
	return out, nil
}

func (m *mockSessionRepo) Update(_ context.Context, s *model.Session) error {
	m.mu.Lock()
	m.data[s.ID] = s
	m.mu.Unlock()
	return nil
}

func (m *mockSessionRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	delete(m.data, id)
	m.mu.Unlock()
	return nil
}

func TestSessionCreateAndCurrent(t *testing.T) {
	m := app.NewSessionManager(newMockSessionRepo(), &mockIDGenerator{})

	s, err := m.Create("test-session")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "test-session" {
		t.Errorf("Name = %q, want %q", s.Name, "test-session")
	}
	if s.ID == "" {
		t.Error("ID is empty")
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	cur := m.Current()
	if cur == nil || cur.ID != s.ID {
		t.Error("Current() does not return the created session")
	}
}

func TestSessionLoad(t *testing.T) {
	repo := newMockSessionRepo()
	m := app.NewSessionManager(repo, &mockIDGenerator{})

	s, _ := m.Create("original")

	m2 := app.NewSessionManager(repo, &mockIDGenerator{})
	loaded, err := m2.Load(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "original" {
		t.Errorf("Name = %q after Load", loaded.Name)
	}
	if m2.Current().ID != s.ID {
		t.Error("Current() not set after Load")
	}
}

func TestSessionList(t *testing.T) {
	m := app.NewSessionManager(newMockSessionRepo(), &mockIDGenerator{})
	m.Create("a")
	m.Create("b")

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}
}

func TestSessionRename(t *testing.T) {
	m := app.NewSessionManager(newMockSessionRepo(), &mockIDGenerator{})

	s, _ := m.Create("old")
	if err := m.Rename(s.ID, "new"); err != nil {
		t.Fatal(err)
	}

	cur := m.Current()
	if cur.Name != "new" {
		t.Errorf("Name = %q after Rename", cur.Name)
	}
}

func TestSessionDelete(t *testing.T) {
	m := app.NewSessionManager(newMockSessionRepo(), &mockIDGenerator{})

	s, _ := m.Create("to-delete")
	m.Delete(s.ID)

	if m.Current() == nil {
		t.Error("Current() should still return the session object")
	}

	list, _ := m.List()
	if len(list) != 0 {
		t.Error("expected empty list after delete")
	}
}
