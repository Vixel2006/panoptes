package port

import "github.com/Vixel2006/panoptes/internal/core/models"

type NoteUseCase interface {
	Create(groupID, title, body string) (*model.Note, error)
	List(groupID string) ([]*model.Note, error)
	Update(id, title, body string) error
	Delete(id string) error
}
