package port

import (
	"context"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type InterceptorPort interface {
	Intercept(ctx context.Context, req *model.Request) (*model.Response, error)
}
