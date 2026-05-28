package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/google/uuid"
)

type Interceptor struct {
	requestCh chan<- model.Request
}

func NewInterceptor(requestCh chan<- model.Request) *Interceptor {
	return &Interceptor{
		requestCh: requestCh,
	}
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

	if i.requestCh != nil {
		select {
		case i.requestCh <- req:
		default:
		}
	}

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

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		Headers: r.Header,
		Body:    io.NopCloser(bytes.NewReader(rawBody)),
	}

	_ = resp
	// TODO: persist dbResp and push to TUI

	return nil
}
