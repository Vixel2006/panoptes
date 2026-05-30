package port

import "github.com/Vixel2006/panoptes/internal/core/models"

type InterceptorPort interface {
	InterceptRequest(req model.Request) error
	InterceptResponse(resp model.Response, reqID string) error
	SetActiveSession(sessionID string)
	Stats() InterceptorStats
	Stop()
}

type InterceptorStats struct {
	DroppedRequests  uint64
	DroppedResponses uint64
}
