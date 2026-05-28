package repo_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Vixel2006/panoptes/internal/adapters/repo"
	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/testutil/dbtest"
)

func TestResponseCreateAndGet(t *testing.T) {
	db := dbtest.Open(t)
	rr := repo.NewRequestRepository(db)
	respR := repo.NewResponseRepository(db)

	rr.Create(ctx, &model.Request{ID: "req-1", URL: "http://example.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})

	resp := &model.Response{
		ID:         "resp-1",
		Status:     "200 OK",
		StatusCode: 200,
		Header:     json.RawMessage(`{"Content-Type":["text/plain"]}`),
		Payload:    json.RawMessage(`ok`),
		Length:     2,
		RequestID:  "req-1",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := respR.Create(ctx, resp); err != nil {
		t.Fatal(err)
	}

	got, err := respR.GetByID(ctx, "resp-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.StatusCode != 200 || got.RequestID != "req-1" {
		t.Errorf("got %+v", got)
	}
}

func TestResponseGetByRequestID(t *testing.T) {
	db := dbtest.Open(t)
	rr := repo.NewRequestRepository(db)
	respR := repo.NewResponseRepository(db)

	rr.Create(ctx, &model.Request{ID: "req-1", URL: "http://example.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})
	respR.Create(ctx, &model.Response{ID: "resp-1", Status: "200", StatusCode: 200, Header: json.RawMessage(`{}`), RequestID: "req-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	got, err := respR.GetByRequestID(ctx, "req-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "resp-1" {
		t.Errorf("ID = %q, want %q", got.ID, "resp-1")
	}
}

func TestResponseDelete(t *testing.T) {
	db := dbtest.Open(t)
	rr := repo.NewRequestRepository(db)
	respR := repo.NewResponseRepository(db)

	rr.Create(ctx, &model.Request{ID: "req-1", URL: "http://example.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})
	respR.Create(ctx, &model.Response{ID: "resp-1", Status: "200", StatusCode: 200, Header: json.RawMessage(`{}`), RequestID: "req-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	respR.Delete(ctx, "resp-1")

	_, err := respR.GetByID(ctx, "resp-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestResponseCascadeDeleteWithRequest(t *testing.T) {
	db := dbtest.Open(t)
	rr := repo.NewRequestRepository(db)
	respR := repo.NewResponseRepository(db)

	rr.Create(ctx, &model.Request{ID: "req-1", URL: "http://example.com", Method: "GET", Header: json.RawMessage(`{}`), CreatedAt: time.Now(), UpdatedAt: time.Now()})
	respR.Create(ctx, &model.Response{ID: "resp-1", Status: "200", StatusCode: 200, Header: json.RawMessage(`{}`), RequestID: "req-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	rr.Delete(ctx, "req-1")

	_, err := respR.GetByID(ctx, "resp-1")
	if err == nil {
		t.Fatal("expected response to cascade delete with request")
	}
}
