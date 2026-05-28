package repo_test

import (
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/testutil/dbtest"
)

func TestNoteCreateAndGet(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	nr := repo.NewNoteRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	n := &model.Note{
		ID:        "note-1",
		Title:     "Test Note",
		Body:      "hello world",
		GroupID:   "grp-1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := nr.Create(ctx, n); err != nil {
		t.Fatal(err)
	}

	got, err := nr.GetByID(ctx, "note-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Test Note" || got.Body != "hello world" || got.GroupID != "grp-1" {
		t.Errorf("got %+v", got)
	}
}

func TestNoteListByGroup(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	nr := repo.NewNoteRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	nr.Create(ctx, &model.Note{ID: "n1", Title: "A", Body: "a", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	nr.Create(ctx, &model.Note{ID: "n2", Title: "B", Body: "b", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	notes, err := nr.ListByGroup(ctx, "grp-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestNoteUpdate(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	nr := repo.NewNoteRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	nr.Create(ctx, &model.Note{ID: "note-1", Title: "old", Body: "old body", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	nr.Update(ctx, &model.Note{ID: "note-1", Title: "new", Body: "new body", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	got, _ := nr.GetByID(ctx, "note-1")
	if got.Title != "new" || got.Body != "new body" {
		t.Errorf("got title=%q body=%q", got.Title, got.Body)
	}
}

func TestNoteDelete(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	nr := repo.NewNoteRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	nr.Create(ctx, &model.Note{ID: "note-1", Title: "x", Body: "y", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	nr.Delete(ctx, "note-1")

	_, err := nr.GetByID(ctx, "note-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestNoteCascadeDeleteWithGroup(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	nr := repo.NewNoteRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	nr.Create(ctx, &model.Note{ID: "note-1", Title: "x", Body: "y", GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	gr.Delete(ctx, "grp-1")

	_, err := nr.GetByID(ctx, "note-1")
	if err == nil {
		t.Fatal("expected note to cascade delete with group")
	}
}
