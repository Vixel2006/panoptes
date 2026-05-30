package adapter_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	adapter "github.com/Vixel2006/panoptes/internal/adapters"
	"github.com/Vixel2006/panoptes/internal/app"
	model "github.com/Vixel2006/panoptes/internal/core/models"
	port "github.com/Vixel2006/panoptes/internal/core/ports"
	service "github.com/Vixel2006/panoptes/internal/core/services"
)

type fakeCertIssuer struct{}

func (f *fakeCertIssuer) IssueLeaf(hostname string) (*tls.Certificate, error) {
	return &tls.Certificate{}, nil
}

type fakeForwarder struct {
	mu   sync.Mutex
	resp *http.Response
	err  error
	reqs []*http.Request
}

func (f *fakeForwarder) RoundTrip(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.reqs = append(f.reqs, req)
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakeForwarder) Requests() []*http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*http.Request, len(f.reqs))
	copy(out, f.reqs)
	return out
}

type mockPersister struct {
	mu    sync.Mutex
	reqs  []model.Request
	resps []model.Response
}

func (m *mockPersister) CreateReq(_ context.Context, req *model.Request) error {
	m.mu.Lock()
	m.reqs = append(m.reqs, *req)
	m.mu.Unlock()
	return nil
}

func (m *mockPersister) CreateResp(_ context.Context, resp *model.Response) error {
	m.mu.Lock()
	m.resps = append(m.resps, *resp)
	m.mu.Unlock()
	return nil
}

type reqWriter struct {
	p *mockPersister
}

func (w *reqWriter) Create(ctx context.Context, req *model.Request) error {
	return w.p.CreateReq(ctx, req)
}

type respWriter struct {
	p *mockPersister
}

func (w *respWriter) Create(ctx context.Context, resp *model.Response) error {
	return w.p.CreateResp(ctx, resp)
}

func newTestAdapter(forwarder port.HTTPForwarder) *adapter.InterceptAdapter {
	persister := &mockPersister{}
	interceptor := app.NewInterceptor(
		make(chan model.Request, 100),
		&reqWriter{p: persister},
		&respWriter{p: persister},
	)

	return adapter.NewInterceptAdapter(
		&fakeCertIssuer{},
		service.NewBarrier(),
		interceptor,
		adapter.NewDecompressor(),
		adapter.NewUUIDGenerator(),
		forwarder,
		nil,
	)
}

func TestInterceptRoundTripForwardsRequest(t *testing.T) {
	fwd := &fakeForwarder{
		resp: &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     http.Header{},
		},
	}
	a := newTestAdapter(fwd)
	t.Cleanup(func() { a.Interceptor().Stop() })

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	var buf bytes.Buffer
	if err := a.InterceptRoundTrip(req, &buf); err != nil {
		t.Fatal(err)
	}

	requests := fwd.Requests()
	if len(requests) != 1 {
		t.Fatalf("expected 1 forwarded request, got %d", len(requests))
	}
	if requests[0].URL.String() != "http://example.com/foo" {
		t.Errorf("URL = %q", requests[0].URL.String())
	}
}

func TestInterceptRoundTripWritesResponseToWriter(t *testing.T) {
	fwd := &fakeForwarder{
		resp: &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("hello from upstream")),
			Header:     http.Header{},
		},
	}
	a := newTestAdapter(fwd)
	t.Cleanup(func() { a.Interceptor().Stop() })

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	var buf bytes.Buffer
	if err := a.InterceptRoundTrip(req, &buf); err != nil {
		t.Fatal(err)
	}

	respBytes := buf.Bytes()
	if !bytes.Contains(respBytes, []byte("200 OK")) {
		t.Errorf("response missing status line, got: %s", string(respBytes))
	}
	if !bytes.Contains(respBytes, []byte("hello from upstream")) {
		t.Errorf("response missing body, got: %s", string(respBytes))
	}
}

func TestInterceptRoundTripReturns502OnForwarderError(t *testing.T) {
	fwd := &fakeForwarder{
		err: io.ErrUnexpectedEOF,
	}
	a := newTestAdapter(fwd)
	t.Cleanup(func() { a.Interceptor().Stop() })

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	var buf bytes.Buffer
	if err := a.InterceptRoundTrip(req, &buf); err != nil {
		t.Fatal(err)
	}

	respBytes := buf.Bytes()
	if !bytes.Contains(respBytes, []byte("502 Bad Gateway")) {
		t.Errorf("expected 502, got: %s", string(respBytes))
	}
}

func TestInterceptRoundTripBarrierDropReturns403(t *testing.T) {
	fwd := &fakeForwarder{
		resp: &http.Response{
			StatusCode: 200,
			Status:     "200 OK",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     http.Header{},
		},
	}
	a := newTestAdapter(fwd)
	t.Cleanup(func() { a.Interceptor().Stop() })

	a.Barrier().SetActive(true)

	go func() {
		time.Sleep(5 * time.Millisecond)
		a.Barrier().Release(false)
	}()

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	var buf bytes.Buffer
	if err := a.InterceptRoundTrip(req, &buf); err != nil {
		t.Fatal(err)
	}

	respBytes := buf.Bytes()
	if !bytes.Contains(respBytes, []byte("403")) {
		t.Errorf("expected 403, got: %s", string(respBytes))
	}
}
