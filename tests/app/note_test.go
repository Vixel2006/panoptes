package app_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Vixel2006/panoptes/internal/app"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

type mockNoteRepo struct {
	mu   sync.Mutex
	data map[string]*model.Note
}

func newMockNoteRepo() *mockNoteRepo {
	return &mockNoteRepo{data: make(map[string]*model.Note)}
}

func (m *mockNoteRepo) Create(_ context.Context, n *model.Note) error {
	m.mu.Lock()
	m.data[n.ID] = n
	m.mu.Unlock()
	return nil
}

func (m *mockNoteRepo) GetByID(_ context.Context, id string) (*model.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.data[id]
	if !ok {
		return nil, &notFoundError{"note"}
	}
	return n, nil
}

func (m *mockNoteRepo) ListByGroup(_ context.Context, groupID string) ([]*model.Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*model.Note
	for _, n := range m.data {
		if n.GroupID == groupID {
			out = append(out, n)
		}
	}
	return out, nil
}

func (m *mockNoteRepo) Update(_ context.Context, n *model.Note) error {
	m.mu.Lock()
	m.data[n.ID] = n
	m.mu.Unlock()
	return nil
}

func (m *mockNoteRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	delete(m.data, id)
	m.mu.Unlock()
	return nil
}

func TestNoteCreate(t *testing.T) {
	m := app.NewNoteManager(newMockNoteRepo(), &mockIDGenerator{})

	n, err := m.Create("grp-1", "Test Note", "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if n.Title != "Test Note" {
		t.Errorf("Title = %q, want %q", n.Title, "Test Note")
	}
	if n.Body != "hello world" {
		t.Errorf("Body = %q, want %q", n.Body, "hello world")
	}
	if n.GroupID != "grp-1" {
		t.Errorf("GroupID = %q, want %q", n.GroupID, "grp-1")
	}
	if n.ID == "" {
		t.Error("ID is empty")
	}
}

func TestNoteList(t *testing.T) {
	m := app.NewNoteManager(newMockNoteRepo(), &mockIDGenerator{})

	m.Create("grp-1", "A", "a")
	m.Create("grp-1", "B", "b")
	m.Create("grp-2", "C", "c")

	list, err := m.List("grp-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 notes for grp-1, got %d", len(list))
	}
}

func TestNoteUpdate(t *testing.T) {
	m := app.NewNoteManager(newMockNoteRepo(), &mockIDGenerator{})

	n, _ := m.Create("grp-1", "old", "old body")
	if err := m.Update(n.ID, "new", "new body"); err != nil {
		t.Fatal(err)
	}

	list, _ := m.List("grp-1")
	if len(list) != 1 {
		t.Fatalf("expected 1 note, got %d", len(list))
	}
	if list[0].Title != "new" || list[0].Body != "new body" {
		t.Errorf("got title=%q body=%q", list[0].Title, list[0].Body)
	}
}

func TestNoteDelete(t *testing.T) {
	m := app.NewNoteManager(newMockNoteRepo(), &mockIDGenerator{})

	n, _ := m.Create("grp-1", "x", "y")
	if err := m.Delete(n.ID); err != nil {
		t.Fatal(err)
	}

	list, _ := m.List("grp-1")
	if len(list) != 0 {
		t.Error("expected empty list after delete")
	}
}

func TestNoteUpdateNotFound(t *testing.T) {
	m := app.NewNoteManager(newMockNoteRepo(), &mockIDGenerator{})

	err := m.Update("nonexistent", "title", "body")
	if err == nil {
		t.Fatal("expected error for nonexistent note")
	}
}
