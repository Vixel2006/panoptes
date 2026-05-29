package port

import (
	"context"

	"github.com/Vixel2006/panoptes/internal/core/models"
)

type SessionRepo interface {
	Create(ctx context.Context, s *model.Session) error
	GetByID(ctx context.Context, id string) (*model.Session, error)
	List(ctx context.Context) ([]*model.Session, error)
	Update(ctx context.Context, s *model.Session) error
	Delete(ctx context.Context, id string) error
}

type GroupRepo interface {
	Create(ctx context.Context, g *model.Group) error
	GetByID(ctx context.Context, id string) (*model.Group, error)
	ListBySession(ctx context.Context, sessionID string) ([]*model.Group, error)
	Delete(ctx context.Context, id string) error
}

type NoteRepo interface {
	Create(ctx context.Context, n *model.Note) error
	GetByID(ctx context.Context, id string) (*model.Note, error)
	ListByGroup(ctx context.Context, groupID string) ([]*model.Note, error)
	Update(ctx context.Context, n *model.Note) error
	Delete(ctx context.Context, id string) error
}

type RequestWriter interface {
	Create(ctx context.Context, req *model.Request) error
}

type ResponseWriter interface {
	Create(ctx context.Context, resp *model.Response) error
}
