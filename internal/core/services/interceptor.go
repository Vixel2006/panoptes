package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/google/uuid"
)

type requestPersister interface {
	Create(ctx context.Context, req *model.Request) error
}

type responsePersister interface {
	Create(ctx context.Context, resp *model.Response) error
}

type Interceptor struct {
	requestCh chan<- model.Request
	lastReqID string

	persistReqCh  chan model.Request
	persistRespCh chan model.Response
	done          chan struct{}
	workerWg      sync.WaitGroup
	stopOnce      sync.Once
}

func NewInterceptor(requestCh chan<- model.Request, reqPersist requestPersister, respPersist responsePersister) *Interceptor {
	i := &Interceptor{
		requestCh:    requestCh,
		persistReqCh: make(chan model.Request, 1024),
		persistRespCh: make(chan model.Response, 1024),
		done:         make(chan struct{}),
	}
	i.startWorker(reqPersist, respPersist)
	return i
}

func (i *Interceptor) startWorker(reqPersist requestPersister, respPersist responsePersister) {
	i.workerWg.Add(1)
	go func() {
		defer i.workerWg.Done()
		for {
			select {
			case req := <-i.persistReqCh:
				if reqPersist != nil {
					reqPersist.Create(context.Background(), &req)
				}
			case resp := <-i.persistRespCh:
				if respPersist != nil {
					respPersist.Create(context.Background(), &resp)
				}
			case <-i.done:
				for {
					select {
					case req := <-i.persistReqCh:
						if reqPersist != nil {
							reqPersist.Create(context.Background(), &req)
						}
					case resp := <-i.persistRespCh:
						if respPersist != nil {
							respPersist.Create(context.Background(), &resp)
						}
					default:
						return
					}
				}
			}
		}
	}()
}

func (i *Interceptor) Stop() {
	i.stopOnce.Do(func() {
		close(i.done)
		i.workerWg.Wait()
	})
}

func (i *Interceptor) InterceptRequest(r *http.Request) error {
	rawBody, _ := io.ReadAll(r.Body)
	r.Body.Close()

	r.Body = io.NopCloser(bytes.NewReader(rawBody))
	r.ContentLength = int64(len(rawBody))
	r.Header.Del("Transfer-Encoding")

	headerJSON, _ := json.Marshal(r.Header)

	req := model.Request{
		ID:      uuid.New().String(),
		URL:     r.URL.String(),
		Method:  r.Method,
		Header:  json.RawMessage(headerJSON),
		Payload: json.RawMessage(rawBody),
		Length:  r.ContentLength,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		ParsedURL: r.URL,
		Host:      r.Host,
		Headers:   r.Header,
		Body:      io.NopCloser(bytes.NewReader(rawBody)),
		RawBody:   rawBody,
	}

	select {
	case i.persistReqCh <- req:
	default:
	}

	if i.requestCh != nil {
		select {
		case i.requestCh <- req:
		default:
		}
	}

	i.lastReqID = req.ID

	return nil
}

func (i *Interceptor) InterceptResponse(r *http.Response) error {
	rawBody, _ := io.ReadAll(r.Body)
	r.Body.Close()

	r.Body = io.NopCloser(bytes.NewReader(rawBody))
	r.ContentLength = int64(len(rawBody))
	r.Header.Del("Transfer-Encoding")

	headerJSON, _ := json.Marshal(r.Header)

	resp := model.Response{
		ID:         uuid.New().String(),
		Status:     r.Status,
		StatusCode: r.StatusCode,
		Header:     json.RawMessage(headerJSON),
		Payload:    json.RawMessage(rawBody),
		RequestID:  i.lastReqID,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		Headers: r.Header,
		Body:    io.NopCloser(bytes.NewReader(rawBody)),
	}

	select {
	case i.persistRespCh <- resp:
	default:
	}

	return nil
}
