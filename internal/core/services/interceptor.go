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
	// Here we add the repo ports
}

func NewInterceptor() *Interceptor {
	return &Interceptor{}
}

func (i *Interceptor) InterceptRequest(req *http.Request) error {
	rawBody, _ := io.ReadAll(req.Body)
	req.Body.Close()

	headerJSON, _ := json.Marshal(req.Header)

	dbReq := model.Request{
		ID:      uuid.New().String(),
		URL:     req.URL.String(),
		Method:  req.Method,
		Header:  json.RawMessage(headerJSON),
		Payload: json.RawMessage(rawBody),
		Length:  req.ContentLength,

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		ParsedURL: req.URL,
		Host:      req.Host,
		Headers:   req.Header,
		Body:      io.NopCloser(bytes.NewReader(rawBody)),
		RawBody:   rawBody,
	}

	_ = dbReq
	// TODO: persist dbReq and push to TUI

	return nil
}

func (i *Interceptor) InterceptResponse(resp *http.Response) error {
	rawBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	headerJSON, _ := json.Marshal(resp.Header)

	dbResp := model.Response{
		ID:         uuid.New().String(),
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Header:     json.RawMessage(headerJSON),
		Payload:    json.RawMessage(rawBody),

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),

		Headers: resp.Header,
		Body:    io.NopCloser(bytes.NewReader(rawBody)),
	}

	_ = dbResp
	// TODO: persist dbResp and push to TUI

	return nil
}
