package repo_test

import (
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/testutil/dbtest"
)

func TestSessionCreateAndGet(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewSessionRepository(db)

	s := &model.Session{
		ID:        "sess-1",
		Name:      "test session",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.Create(ctx, s); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetByID(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != s.ID {
		t.Errorf("ID = %q, want %q", got.ID, s.ID)
	}
	if got.Name != s.Name {
		t.Errorf("Name = %q, want %q", got.Name, s.Name)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestSessionGetNotFound(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewSessionRepository(db)

	_, err := r.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
}

func TestSessionList(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewSessionRepository(db)

	r.Create(ctx, &model.Session{ID: "a", Name: "A", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	r.Create(ctx, &model.Session{ID: "b", Name: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	sessions, err := r.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestSessionUpdate(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewSessionRepository(db)

	s := &model.Session{ID: "sess-1", Name: "old", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.Create(ctx, s)

	s.Name = "updated"
	s.UpdatedAt = time.Now()
	if err := r.Update(ctx, s); err != nil {
		t.Fatal(err)
	}

	got, _ := r.GetByID(ctx, "sess-1")
	if got.Name != "updated" {
		t.Errorf("Name = %q, want %q", got.Name, "updated")
	}
}

func TestSessionDelete(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewSessionRepository(db)

	r.Create(ctx, &model.Session{ID: "sess-1", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	if err := r.Delete(ctx, "sess-1"); err != nil {
		t.Fatal(err)
	}

	_, err := r.GetByID(ctx, "sess-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}
