package app_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Vixel2006/panoptes/internal/app"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

type mockGroupRepo struct {
	mu   sync.Mutex
	data map[string]*model.Group
}

func newMockGroupRepo() *mockGroupRepo {
	return &mockGroupRepo{data: make(map[string]*model.Group)}
}

func (m *mockGroupRepo) Create(_ context.Context, g *model.Group) error {
	m.mu.Lock()
	m.data[g.ID] = g
	m.mu.Unlock()
	return nil
}

func (m *mockGroupRepo) GetByID(_ context.Context, id string) (*model.Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.data[id]
	if !ok {
		return nil, &notFoundError{"group"}
	}
	return g, nil
}

func (m *mockGroupRepo) ListBySession(_ context.Context, sessionID string) ([]*model.Group, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*model.Group
	for _, g := range m.data {
		if g.SessionID == sessionID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *mockGroupRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	delete(m.data, id)
	m.mu.Unlock()
	return nil
}

func TestGroupCreate(t *testing.T) {
	m := app.NewGroupManager(newMockGroupRepo(), &mockIDGenerator{})

	g, err := m.Create("sess-1", "Test Group")
	if err != nil {
		t.Fatal(err)
	}
	if g.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", g.SessionID, "sess-1")
	}
	if g.Name != "Test Group" {
		t.Errorf("Name = %q, want %q", g.Name, "Test Group")
	}
	if g.ID == "" {
		t.Error("ID is empty")
	}
}

func TestGroupGet(t *testing.T) {
	m := app.NewGroupManager(newMockGroupRepo(), &mockIDGenerator{})

	created, _ := m.Create("sess-1", "Test Group")
	got, err := m.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %q", got.ID)
	}
}

func TestGroupList(t *testing.T) {
	m := app.NewGroupManager(newMockGroupRepo(), &mockIDGenerator{})

	m.Create("sess-1", "Group 1")
	m.Create("sess-1", "Group 2")
	m.Create("sess-2", "Group 3")

	list, err := m.List("sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 groups for sess-1, got %d", len(list))
	}

	list2, _ := m.List("sess-2")
	if len(list2) != 1 {
		t.Fatalf("expected 1 group for sess-2, got %d", len(list2))
	}
}

func TestGroupDelete(t *testing.T) {
	m := app.NewGroupManager(newMockGroupRepo(), &mockIDGenerator{})

	g, _ := m.Create("sess-1", "Delete Group")
	if err := m.Delete(g.ID); err != nil {
		t.Fatal(err)
	}

	_, err := m.Get(g.ID)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
