package app

import (
	"context"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
)

type SessionManager struct {
	repo    port.SessionRepo
	idGen   port.IDGenerator
	current *model.Session
}

func NewSessionManager(repo port.SessionRepo, idGen port.IDGenerator) *SessionManager {
	return &SessionManager{repo: repo, idGen: idGen}
}

func (m *SessionManager) Create(name string) (*model.Session, error) {
	s := &model.Session{
		ID:        m.idGen.New(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	ctx := context.Background()
	if err := m.repo.Create(ctx, s); err != nil {
		return nil, err
	}
	m.current = s
	return s, nil
}

func (m *SessionManager) Load(id string) (*model.Session, error) {
	s, err := m.repo.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	m.current = s
	return s, nil
}

func (m *SessionManager) Current() *model.Session {
	return m.current
}

func (m *SessionManager) List() ([]*model.Session, error) {
	return m.repo.List(context.Background())
}

func (m *SessionManager) Rename(id, name string) error {
	s, err := m.repo.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	s.Name = name
	s.UpdatedAt = time.Now()
	return m.repo.Update(context.Background(), s)
}

func (m *SessionManager) Delete(id string) error {
	return m.repo.Delete(context.Background(), id)
}
