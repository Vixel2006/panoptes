package port

import (
	"context"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type ResponseRepo interface {
	ResponseWriter
	GetByRequestID(ctx context.Context, requestID string) (*model.Response, error)
	Delete(ctx context.Context, id string) error
}
