package repo_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/testutil/dbtest"
)

func TestRequestCreateAndGet(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewRequestRepository(db)

	req := &model.Request{
		ID:        "req-1",
		URL:       "http://example.com",
		Method:    "GET",
		Header:    json.RawMessage(`{"Accept":["*/*"]}`),
		Payload:   json.RawMessage(`{}`),
		Length:    2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.Create(ctx, req); err != nil {
		t.Fatal(err)
	}

	got, err := r.GetByID(ctx, "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "req-1" || got.URL != "http://example.com" || got.Method != "GET" {
		t.Errorf("got %+v", got)
	}
}

func TestRequestListAll(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewRequestRepository(db)

	r.Create(ctx, &model.Request{ID: "r1", URL: "http://a.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})
	r.Create(ctx, &model.Request{ID: "r2", URL: "http://b.com", Method: "POST", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
}

func TestRequestUpdateGroup(t *testing.T) {
	db := dbtest.Open(t)

	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	rr := repo.NewRequestRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	rr.Create(ctx, &model.Request{ID: "r1", URL: "http://a.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})

	if err := rr.UpdateGroup(ctx, "r1", "grp-1"); err != nil {
		t.Fatal(err)
	}

	req, _ := rr.GetByID(ctx, "r1")
	if req.GroupID != "grp-1" {
		t.Errorf("GroupID = %q, want %q", req.GroupID, "grp-1")
	}
}

func TestRequestListByGroup(t *testing.T) {
	db := dbtest.Open(t)

	sr := repo.NewSessionRepository(db)
	gr := repo.NewGroupRepository(db)
	rr := repo.NewRequestRepository(db)

	sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	gr.Create(ctx, &model.Group{ID: "grp-1", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	rr.Create(ctx, &model.Request{ID: "r1", URL: "http://a.com", Method: "GET", Header: json.RawMessage(`{}`), GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	rr.Create(ctx, &model.Request{ID: "r2", URL: "http://b.com", Method: "GET", Header: json.RawMessage(`{}`), GroupID: "grp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	reqs, err := rr.ListByGroup(ctx, "grp-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2, got %d", len(reqs))
	}
}

func TestRequestDelete(t *testing.T) {
	db := dbtest.Open(t)
	r := repo.NewRequestRepository(db)

	r.Create(ctx, &model.Request{ID: "r1", URL: "http://a.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})

	if err := r.Delete(ctx, "r1"); err != nil {
		t.Fatal(err)
	}

	_, err := r.GetByID(ctx, "r1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestRequestListBySession(t *testing.T) {
	db := dbtest.Open(t)
	sr := repo.NewSessionRepository(db)
	r := repo.NewRequestRepository(db)

	_ = sr.Create(ctx, &model.Session{ID: "sess-1", Name: "s1", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	_ = sr.Create(ctx, &model.Session{ID: "sess-2", Name: "s2", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	now := time.Now()
	if err := r.Create(ctx, &model.Request{ID: "r1", URL: "http://a.com", Method: "GET", Header: json.RawMessage(`{}`), SessionID: "sess-1", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, &model.Request{ID: "r2", URL: "http://b.com", Method: "POST", Header: json.RawMessage(`{}`), SessionID: "sess-1", CreatedAt: now.Add(time.Second), UpdatedAt: now.Add(time.Second)}); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, &model.Request{ID: "r3", URL: "http://c.com", Method: "GET", Header: json.RawMessage(`{}`), SessionID: "sess-2", CreatedAt: now.Add(2 * time.Second), UpdatedAt: now.Add(2 * time.Second)}); err != nil {
		t.Fatal(err)
	}

	sess1Reqs, err := r.ListBySession(ctx, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(sess1Reqs) != 2 {
		t.Fatalf("expected 2 requests for sess-1, got %d", len(sess1Reqs))
	}
	if sess1Reqs[0].ID != "r1" {
		t.Errorf("expected first request for sess-1 to be r1, got %s", sess1Reqs[0].ID)
	}
	if sess1Reqs[1].ID != "r2" {
		t.Errorf("expected second request for sess-1 to be r2, got %s", sess1Reqs[1].ID)
	}

	sess2Reqs, err := r.ListBySession(ctx, "sess-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(sess2Reqs) != 1 {
		t.Fatalf("expected 1 request for sess-2, got %d", len(sess2Reqs))
	}
	if sess2Reqs[0].ID != "r3" {
		t.Errorf("got request ID %q, want r3", sess2Reqs[0].ID)
	}
}
