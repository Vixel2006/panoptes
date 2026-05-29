package app

import (
	"context"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
)

type GroupManager struct {
	repo  port.GroupRepo
	idGen port.IDGenerator
}

func NewGroupManager(repo port.GroupRepo, idGen port.IDGenerator) *GroupManager {
	return &GroupManager{repo: repo, idGen: idGen}
}

func (m *GroupManager) Create(sessionID, name string) (*model.Group, error) {
	g := &model.Group{
		ID:        m.idGen.New(),
		SessionID: sessionID,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx := context.Background()
	if err := m.repo.Create(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (m *GroupManager) Get(id string) (*model.Group, error) {
	return m.repo.GetByID(context.Background(), id)
}

func (m *GroupManager) List(sessionID string) ([]*model.Group, error) {
	return m.repo.ListBySession(context.Background(), sessionID)
}

func (m *GroupManager) Delete(id string) error {
	return m.repo.Delete(context.Background(), id)
}
