package ui

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type panes int

const (
	paneSidebar panes = iota
	paneList
	paneDetail
)

type detailTab int

const (
	tabRequest detailTab = iota
	tabResponse
	tabNotes
)

type inputMode int

const (
	inputNone inputMode = iota
	inputNewSession
	inputNewGroup
	inputAssignGroup
	inputAddNoteTitle
	inputAddNoteBody
	inputEditNoteTitle
	inputEditNoteBody
	inputFilter
	inputRepeaterMethod
	inputRepeaterURL
	inputRepeaterHeaders
	inputRepeaterBody
	inputPreferredEditor
)

type eventItem struct {
	req  model.Request
	resp *model.Response
}

type inputState struct {
	mode   inputMode
	prompt string
	value  string
}

type modal int

const (
	noModal modal = iota
	modalSessions
	modalGroups
	modalAssignGroup
	modalHelp
)

type picker struct {
	title  string
	items  []string
	cursor int
}

type Config struct {
	Barrier            port.BarrierPort
	Interceptor        port.InterceptorPort
	RequestCh          <-chan model.Request
	Sessions           port.SessionUseCase
	ReqRepo            port.RequestRepo
	RespRepo           port.ResponseRepo
	GroupRepo          port.GroupRepo
	NoteRepo           port.NoteRepo
	IDGen              port.IDGenerator
	Decompressor       port.Decompressor
	InitialSessionName string
}

type Model struct {
	ready  bool
	width  int
	height int

	barrier      port.BarrierPort
	interceptor  port.InterceptorPort
	requestCh    <-chan model.Request
	sessions     port.SessionUseCase
	reqRepo      port.RequestRepo
	respRepo     port.ResponseRepo
	groupRepo    port.GroupRepo
	noteRepo     port.NoteRepo
	idGen        port.IDGenerator
	decompressor port.Decompressor

	events   []eventItem
	eventMap map[string]int
	cursor   int

	detailPort viewport.Model

	activePane    panes
	activeTab     detailTab
	sidebarCursor int

	curModal modal
	pick     picker

	showInput bool
	input     inputState

	currentSession *model.Session
	sessionList    []*model.Session
	currentGroup   *model.Group
	groupList      []*model.Group
	noteList       []*model.Note
	tempNoteID     string
	tempNoteTitle  string
	noteCursor     int
	tempAssignRequestID string

	paused bool
	filter string

	statusMsg    string
	statusExpiry time.Time

	detailLines string

	initialSessionName string

	repeaterActive  bool
	repeaterMethod  string
	repeaterURL     string
	repeaterHeaders string
	repeaterBody    string
	repeaterResp    *model.Response
	repeaterFocus   int
	repeaterLoading bool
	repeaterErr     string

	preferredEditor string
}

func NewModel(cfg Config) Model {
	return Model{
		barrier:            cfg.Barrier,
		interceptor:        cfg.Interceptor,
		requestCh:          cfg.RequestCh,
		sessions:           cfg.Sessions,
		reqRepo:            cfg.ReqRepo,
		respRepo:           cfg.RespRepo,
		groupRepo:          cfg.GroupRepo,
		noteRepo:           cfg.NoteRepo,
		idGen:              cfg.IDGen,
		decompressor:       cfg.Decompressor,
		events:             make([]eventItem, 0),
		eventMap:           make(map[string]int),
		detailPort:         viewport.New(60, 10),
		cursor:             -1,
		activePane:         paneList,
		activeTab:          tabRequest,
		sidebarCursor:      0,
		tempNoteTitle:      "",
		initialSessionName: cfg.InitialSessionName,
		preferredEditor:    os.Getenv("EDITOR"),
	}
}

type eventReceived struct {
	req model.Request
}

type responseReceived struct {
	reqID string
	resp  *model.Response
}

type sessionsLoaded struct {
	sessions []*model.Session
}

type groupsLoaded struct {
	groups []*model.Group
}

type notesLoaded struct {
	notes []*model.Note
}

type eventsBatch struct {
	events []eventItem
}

type statusMsg struct {
	text string
}

type errMsg struct {
	err error
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadSessions(), m.loadEvents())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		m.detailPort.Width = msg.Width - 50
		m.detailPort.Height = msg.Height - 10

	case tea.KeyMsg:
		return m.onKey(msg)

	case eventReceived:
		ev := eventItem{req: msg.req}
		m.events = append([]eventItem{ev}, m.events...)
		for id, idx := range m.eventMap {
			m.eventMap[id] = idx + 1
		}
		m.eventMap[msg.req.ID] = 0
		if m.cursor >= 0 {
			m.cursor++
		} else if m.currentGroup == nil || msg.req.GroupID == m.currentGroup.ID {
			m.cursor = 0
		}
		if m.currentSession != nil && msg.req.SessionID == m.currentSession.ID {
			return m, m.loadResp(msg.req.ID)
		}
		return m, nil

	case responseReceived:
		idx, ok := m.eventMap[msg.reqID]
		if ok && idx >= 0 && idx < len(m.events) {
			m.events[idx].resp = msg.resp
			if idx == m.cursor && m.activeTab == tabResponse {
				m.refreshDetail()
			}
		}

	case eventsBatch:
		m.events = msg.events
		m.eventMap = make(map[string]int, len(m.events))
		for i, ev := range msg.events {
			m.eventMap[ev.req.ID] = i
		}
		if len(m.events) > 0 && m.cursor < 0 {
			m.cursor = 0
		}
		m.refreshDetail()
		m.setStatus(fmt.Sprintf("Loaded %d events", len(m.events)))
		cmds := make([]tea.Cmd, len(m.events))
		for i, ev := range m.events {
			ev := ev
			cmds[i] = m.loadResp(ev.req.ID)
		}
		return m, tea.Batch(cmds...)

	case sessionsLoaded:
		m.sessionList = msg.sessions
		if m.initialSessionName != "" && len(msg.sessions) > 0 {
			var found *model.Session
			for _, s := range m.sessionList {
				if s.Name == m.initialSessionName {
					found = s
					break
				}
			}
			if found != nil {
				m.setStatus("Session: " + found.Name)
				return m, m.selectSession(found)
			} else {
				s, err := m.sessions.Create(m.initialSessionName)
				if err != nil {
					m.setStatus(fmt.Sprintf("Error creating session: %v", err))
					return m, m.makeDefaultSession()
				}
				m.setStatus("Session: " + s.Name)
				return m, tea.Batch(m.selectSession(s), m.loadSessions())
			}
		} else {
			if len(msg.sessions) > 0 {
				m.setStatus("Session: " + msg.sessions[0].Name)
				return m, m.selectSession(msg.sessions[0])
			} else {
				return m, m.makeDefaultSession()
			}
		}

	case groupsLoaded:
		m.groupList = msg.groups
		if m.sidebarCursor >= len(m.groupList)+1 {
			m.sidebarCursor = 0
		}
		return m, nil

	case notesLoaded:
		m.noteList = msg.notes
		if m.noteCursor >= len(m.noteList) {
			if len(m.noteList) > 0 {
				m.noteCursor = len(m.noteList) - 1
			} else {
				m.noteCursor = 0
			}
		}
		m.refreshDetail()
		return m, nil

	case repeaterResult:
		m.repeaterLoading = false
		if msg.err != nil {
			m.repeaterErr = msg.err.Error()
		} else {
			m.repeaterResp = msg.resp
		}
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.setStatus(fmt.Sprintf("Editor error: %v", msg.err))
			os.Remove(msg.filepath)
			return m, nil
		}
		contentBytes, err := os.ReadFile(msg.filepath)
		os.Remove(msg.filepath)
		if err != nil {
			m.setStatus(fmt.Sprintf("Error reading file: %v", err))
			return m, nil
		}
		val := string(contentBytes)
		switch msg.mode {
		case inputRepeaterHeaders:
			m.repeaterHeaders = val
			m.setStatus("Updated Headers from Editor")
		case inputRepeaterBody:
			m.repeaterBody = val
			m.setStatus("Updated Body from Editor")
		case inputAddNoteBody:
			var gid string
			var cmd tea.Cmd
			if m.currentGroup == nil {
				if m.currentSession == nil {
					m.setStatus("No active session")
					m.tempNoteTitle = ""
					return m, nil
				}
				g := &model.Group{
					ID:        m.idGen.New(),
					SessionID: m.currentSession.ID,
					Name:      "Notes Group",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				if err := m.groupRepo.Create(context.Background(), g); err != nil {
					m.setStatus(fmt.Sprintf("Error: %v", err))
					m.tempNoteTitle = ""
					return m, nil
				}
				m.currentGroup = g
				gid = g.ID
				cmd = m.loadGroups(m.currentSession.ID)
			} else {
				gid = m.currentGroup.ID
			}

			n := &model.Note{
				ID:        m.idGen.New(),
				GroupID:   gid,
				Title:     m.tempNoteTitle,
				Body:      val,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := m.noteRepo.Create(context.Background(), n); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
				m.tempNoteTitle = ""
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			}
			m.tempNoteTitle = ""
			m.setStatus("Note added")
			if cmd != nil {
				return m, tea.Batch(cmd, m.loadNotes(gid))
			}
			return m, m.loadNotes(gid)

		case inputEditNoteBody:
			n, err := m.noteRepo.GetByID(context.Background(), m.tempNoteID)
			if err != nil {
				m.setStatus(fmt.Sprintf("Error fetching note: %v", err))
				m.tempNoteID = ""
				m.tempNoteTitle = ""
				return m, nil
			}
			n.Title = m.tempNoteTitle
			n.Body = val
			n.UpdatedAt = time.Now()
			err = m.noteRepo.Update(context.Background(), n)
			m.tempNoteID = ""
			m.tempNoteTitle = ""
			if err != nil {
				m.setStatus(fmt.Sprintf("Error updating note: %v", err))
				return m, nil
			}
			m.setStatus("Note updated")
			if m.currentGroup != nil {
				return m, m.loadNotes(m.currentGroup.ID)
			}
			return m, nil
		}
		return m, nil

	case errMsg:
		m.setStatus(fmt.Sprintf("Error: %v", msg.err))
		return m, nil

	case statusMsg:
		m.setStatus(msg.text)
	}

	return m, nil
}

func (m *Model) refreshDetail() {
	if m.cursor < 0 || m.cursor >= len(m.events) {
		m.detailLines = ""
		m.detailPort.SetContent("")
		return
	}
	m.detailLines = m.buildDetail(m.cursor)
	m.detailPort.SetContent(m.detailLines)
	m.detailPort.GotoTop()
}
