package app

import (
	"context"
	"sync"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
)

type Interceptor struct {
	requestCh chan<- model.Request
	lastReqID string

	persistReqCh  chan model.Request
	persistRespCh chan model.Response
	done          chan struct{}
	workerWg      sync.WaitGroup
	stopOnce      sync.Once

	activeSessionID string
	activeSessionMu sync.RWMutex
}

func NewInterceptor(requestCh chan<- model.Request, reqPersist port.RequestWriter, respPersist port.ResponseWriter) *Interceptor {
	i := &Interceptor{
		requestCh:    requestCh,
		persistReqCh: make(chan model.Request, 1024),
		persistRespCh: make(chan model.Response, 1024),
		done:         make(chan struct{}),
	}
	i.startWorker(reqPersist, respPersist)
	return i
}

func (i *Interceptor) startWorker(reqPersist port.RequestWriter, respPersist port.ResponseWriter) {
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

func (i *Interceptor) InterceptRequest(req model.Request) error {
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	req.SessionID = i.GetActiveSessionID()

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

func (i *Interceptor) InterceptResponse(resp model.Response) error {
	resp.CreatedAt = time.Now()
	resp.UpdatedAt = time.Now()
	resp.RequestID = i.lastReqID

	select {
	case i.persistRespCh <- resp:
	default:
	}

	return nil
}

func (i *Interceptor) SetActiveSession(sessionID string) {
	i.activeSessionMu.Lock()
	defer i.activeSessionMu.Unlock()
	i.activeSessionID = sessionID
}

func (i *Interceptor) GetActiveSessionID() string {
	i.activeSessionMu.RLock()
	defer i.activeSessionMu.RUnlock()
	return i.activeSessionID
}
