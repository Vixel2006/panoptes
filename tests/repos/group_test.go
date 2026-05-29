package repo_test

import (
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/testutil/dbtest"
)

func TestGroupCreateAndGet(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	g := &model.Group{ID: "grp-1", SessionID: "sess-1", Name: "g1", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	if err := gr.Create(ctx, g); err != nil {
		t.Fatal(err)
	}

	got, err := gr.GetByID(ctx, "grp-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "grp-1" || got.SessionID != "sess-1" || got.Name != "g1" {
		t.Errorf("got %+v", got)
	}
}

func TestGroupListBySession(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "g1", SessionID: "sess-1", Name: "g1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "g2", SessionID: "sess-1", Name: "g2", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	groups, err := gr.ListBySession(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
}

func TestGroupDelete(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "g1", SessionID: "sess-1", Name: "g1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	gr.Delete(ctx, "g1")

	_, err := gr.GetByID(ctx, "g1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestGroupDeleteCascadesFromSession(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "g1", SessionID: "sess-1", Name: "g1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	sr.Delete(ctx, "sess-1")

	groups, _ := gr.ListBySession(ctx, "sess-1")
	if len(groups) != 0 {
		t.Error("expected groups to cascade delete with session")
	}
}
