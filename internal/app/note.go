package app

import (
	"context"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
)

type NoteManager struct {
	repo  port.NoteRepo
	idGen port.IDGenerator
}

func NewNoteManager(repo port.NoteRepo, idGen port.IDGenerator) *NoteManager {
	return &NoteManager{repo: repo, idGen: idGen}
}

func (m *NoteManager) Create(groupID, title, body string) (*model.Note, error) {
	n := &model.Note{
		ID:        m.idGen.New(),
		Title:     title,
		Body:      body,
		GroupID:   groupID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx := context.Background()
	if err := m.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (m *NoteManager) List(groupID string) ([]*model.Note, error) {
	return m.repo.ListByGroup(context.Background(), groupID)
}

func (m *NoteManager) Update(id, title, body string) error {
	n, err := m.repo.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	n.Title = title
	n.Body = body
	n.UpdatedAt = time.Now()
	return m.repo.Update(context.Background(), n)
}

func (m *NoteManager) Delete(id string) error {
	return m.repo.Delete(context.Background(), id)
}
