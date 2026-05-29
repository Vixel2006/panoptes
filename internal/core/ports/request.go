package port

import (
	"context"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type RequestRepo interface {
	RequestWriter
	GetByID(ctx context.Context, id string) (*model.Request, error)
	ListByGroup(ctx context.Context, groupID string) ([]*model.Request, error)
	ListBySession(ctx context.Context, sessionID string) ([]*model.Request, error)
	ListAll(ctx context.Context) ([]*model.Request, error)
	UpdateGroup(ctx context.Context, id, groupID string) error
	Delete(ctx context.Context, id string) error
}
