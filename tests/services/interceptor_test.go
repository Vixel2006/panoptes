package service_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/services"
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
	ic := service.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	body := `{"key": "value"}`
	httpReq, err := http.NewRequest("POST", "http://example.com/foo?q=1", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Custom", "test")

	if err := ic.InterceptRequest(httpReq); err != nil {
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
	if p.ID == "" {
		t.Error("ID is empty")
	}
	if p.Length != int64(len(body)) {
		t.Errorf("Length = %d, want %d", p.Length, len(body))
	}
	if p.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestInterceptResponseLinksToLastRequest(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := service.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	httpReq, _ := http.NewRequest("GET", "http://example.com/", http.NoBody)
	ic.InterceptRequest(httpReq)
	ic.Stop()

	ic2 := service.NewInterceptor(nil, reqPersist, respPersist)
	defer ic2.Stop()
	// Set lastReqID by calling InterceptRequest
	httpReq2, _ := http.NewRequest("GET", "http://example.com/", http.NoBody)
	ic2.InterceptRequest(httpReq2)

	httpResp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
	ic2.InterceptResponse(httpResp)
	ic2.Stop()

	persistedResps := respPersist.Responses()
	if len(persistedResps) != 1 {
		t.Fatalf("expected 1 persisted response, got %d", len(persistedResps))
	}

	if persistedResps[0].StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", persistedResps[0].StatusCode)
	}
}

func TestInterceptRequestChannel(t *testing.T) {
	ch := make(chan model.Request, 1)
	ic := service.NewInterceptor(ch, nil, nil)
	defer ic.Stop()

	httpReq, _ := http.NewRequest("GET", "http://example.com/", http.NoBody)
	ic.InterceptRequest(httpReq)

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
	ic := service.NewInterceptor(nil, reqPersist, respPersist)
	defer ic.Stop()

	httpResp := &http.Response{
		StatusCode: 404,
		Status:     "404 Not Found",
		Body:       io.NopCloser(strings.NewReader("not found")),
	}
	ic.InterceptResponse(httpResp)
	ic.Stop()

	persisted := respPersist.Responses()
	if len(persisted) != 1 {
		t.Fatalf("expected 1 persisted response, got %d", len(persisted))
	}
}

func TestInterceptorStopDrainsEvents(t *testing.T) {
	reqPersist := &mockRequestPersister{}
	respPersist := &mockResponsePersister{}
	ic := service.NewInterceptor(nil, reqPersist, respPersist)

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "http://example.com/", http.NoBody)
		ic.InterceptRequest(req)
	}
	ic.Stop()

	if got := len(reqPersist.Requests()); got != 5 {
		t.Errorf("expected 5 persisted requests after Stop, got %d", got)
	}
}
