package port

import "github.com/Vixel2006/panoptes/internal/core/models"

type InterceptorPort interface {
	InterceptRequest(req model.Request) error
	InterceptResponse(resp model.Response) error
	SetActiveSession(sessionID string)
	Stop()
}
