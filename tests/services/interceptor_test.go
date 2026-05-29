package service_test

import (
	"context"
	"sync"
	"testing"

	"github.com/Vixel2006/panoptes/internal/app"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

type mockRequestPersister struct {
	mu   sync.Mutex
	reqs []model.Request
}

func (m *mockRequestPersister) Create(_ context.Context, req *model.Request) error {
	m.mu.Lock()
	m.reqs = append(m.reqs, *req)
	m.mu.Unlock()
	return nil
}

func (m *mockRequestPersister) Requests() []model.Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.Request, len(m.reqs))
	copy(out, m.reqs)
	return out
}

type mockResponsePersister struct {
	mu    sync.Mutex
	resps []model.Response
}

func (m *mockResponsePersister) Create(_ context.Context, resp *model.Response) error {
	m.mu.Lock()
	m.resps = append(m.resps, *resp)
	m.mu.Unlock()
	return nil
}

func (m *mockResponsePersister) Responses() []model.Response {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.Response, len(m.resps))
	copy(out, m.resps)
	return out
}

func TestInterceptRequest(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := app.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	modelReq := model.Request{
		ID:     "req-1",
		URL:    "http://example.com/foo?q=1",
		Method: "POST",
		Length: 16,
	}

	if err := ic.InterceptRequest(modelReq); err != nil {
		t.Fatal(err)
	}
	ic.Stop()

	persisted := reqPersist.Requests()
	if len(persisted) != 1 {
		t.Fatalf("expected 1 persisted request, got %d", len(persisted))
	}

	p := persisted[0]
	if p.URL != "http://example.com/foo?q=1" {
		t.Errorf("URL = %q, want %q", p.URL, "http://example.com/foo?q=1")
	}
	if p.Method != "POST" {
		t.Errorf("Method = %q, want %q", p.Method, "POST")
	}
	if p.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestInterceptResponseLinksToLastRequest(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := app.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	ic.InterceptRequest(model.Request{ID: "req-1", URL: "http://example.com/", Method: "GET"})
	ic.Stop()

	ic2 := app.NewInterceptor(nil, reqPersist, respPersist)
	defer ic2.Stop()
	ic2.InterceptRequest(model.Request{ID: "req-2", URL: "http://example.com/", Method: "GET"})

	ic2.InterceptResponse(model.Response{
		StatusCode: 200,
		Status:     "200 OK",
	})
	ic2.Stop()

	persistedResps := respPersist.Responses()
	if len(persistedResps) != 1 {
		t.Fatalf("expected 1 persisted response, got %d", len(persistedResps))
	}

	if persistedResps[0].StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", persistedResps[0].StatusCode)
	}
	if persistedResps[0].RequestID != "req-2" {
		t.Errorf("RequestID = %q, want req-2", persistedResps[0].RequestID)
	}
}

func TestInterceptRequestChannel(t *testing.T) {
	ch := make(chan model.Request, 1)
	ic := app.NewInterceptor(ch, nil, nil)
	defer ic.Stop()

	ic.InterceptRequest(model.Request{ID: "req-1", URL: "http://example.com/", Method: "GET"})

	select {
	case req := <-ch:
		if req.URL != "http://example.com/" {
			t.Errorf("ch req URL = %q", req.URL)
		}
	default:
		t.Error("expected request on channel")
	}
}

func TestInterceptResponseNoPanicWithoutPriorRequest(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := app.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	ic.InterceptResponse(model.Response{
		StatusCode: 404,
		Status:     "404 Not Found",
	})
	ic.Stop()

	persisted := respPersist.Responses()
	if len(persisted) != 1 {
		t.Fatalf("expected 1 persisted response, got %d", len(persisted))
	}
}

func TestInterceptorStopDrainsEvents(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := app.NewInterceptor(nil, reqPersist, respPersist)

	for i := 0; i < 5; i++ {
		ic.InterceptRequest(model.Request{
			ID:     "",
			URL:    "http://example.com/",
			Method: "GET",
		})
	}
	ic.Stop()

	if got := len(reqPersist.Requests()); got != 5 {
		t.Errorf("expected 5 persisted requests after Stop, got %d", got)
	}
}
