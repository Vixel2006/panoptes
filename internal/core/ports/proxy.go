package port

import (
	"context"
	"net"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type ProxyPort interface {
	Intercept(ctx context.Context, req *model.Request) (*model.Response, error)
}
