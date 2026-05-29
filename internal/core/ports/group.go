package port

import "github.com/Vixel2006/panoptes/internal/core/models"

type GroupUseCase interface {
	Create(sessionID, name string) (*model.Group, error)
	Get(id string) (*model.Group, error)
	List(sessionID string) ([]*model.Group, error)
	Delete(id string) error
}
