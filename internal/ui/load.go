package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

func (m Model) loadSessions() tea.Cmd {
	return func() tea.Msg {
		s, err := m.sessions.List()
		if err != nil {
			return errMsg{err}
		}
		return sessionsLoaded{s}
	}
}

func (m Model) loadGroups(sid string) tea.Cmd {
	return func() tea.Msg {
		g, err := m.groupRepo.ListBySession(context.Background(), sid)
		if err != nil {
			return errMsg{err}
		}
		return groupsLoaded{g}
	}
}

func (m Model) loadNotes(gid string) tea.Cmd {
	return func() tea.Msg {
		n, err := m.noteRepo.ListByGroup(context.Background(), gid)
		if err != nil {
			return errMsg{err}
		}
		return notesLoaded{n}
	}
}

func (m Model) loadEvents() tea.Cmd {
	return func() tea.Msg {
		if m.currentSession == nil {
			return eventsBatch{nil}
		}
		r, err := m.reqRepo.ListBySession(context.Background(), m.currentSession.ID)
		if err != nil {
			return errMsg{err}
		}
		items := make([]eventItem, len(r))
		for i, x := range r {
			items[i] = eventItem{req: *x}
		}
		return eventsBatch{items}
	}
}

func (m Model) loadResp(id string) tea.Cmd {
	return func() tea.Msg {
		r, err := m.respRepo.GetByRequestID(context.Background(), id)
		if err != nil || r == nil {
			return nil
		}
		return responseReceived{id, r}
	}
}

func (m *Model) selectSession(s *model.Session) tea.Cmd {
	m.sessions.Load(s.ID)
	m.interceptor.SetActiveSession(s.ID)
	m.currentSession = s
	m.currentGroup = nil
	m.cursor = -1
	m.events = nil
	m.eventMap = map[string]int{}
	m.detailLines = ""
	m.detailPort.SetContent("")
	return tea.Batch(m.loadGroups(s.ID), m.loadEvents())
}

func (m *Model) makeDefaultSession() tea.Cmd {
	s, err := m.sessions.Create("default")
	if err != nil {
		m.setStatus(fmt.Sprintf("Error: %v", err))
		return nil
	}
	m.setStatus("Created default session")
	return m.selectSession(s)
}
