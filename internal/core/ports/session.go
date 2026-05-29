package port

import "github.com/Vixel2006/panoptes/internal/core/models"

type SessionUseCase interface {
	Create(name string) (*model.Session, error)
	Load(id string) (*model.Session, error)
	Current() *model.Session
	List() ([]*model.Session, error)
	Rename(id, name string) error
	Delete(id string) error
}
