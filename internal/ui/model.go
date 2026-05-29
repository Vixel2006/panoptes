package ui

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Vixel2006/panoptes/internal/core/models"
	"github.com/Vixel2006/panoptes/internal/core/ports"
	"github.com/andybalholm/brotli"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Groups             port.GroupUseCase
	Notes              port.NoteUseCase
	ReqRepo            port.RequestRepo
	RespRepo           port.ResponseRepo
	InitialSessionName string
}

type Model struct {
	ready  bool
	width  int
	height int

	barrier     port.BarrierPort
	interceptor port.InterceptorPort
	requestCh   <-chan model.Request
	sessions    port.SessionUseCase
	groups      port.GroupUseCase
	notes       port.NoteUseCase
	reqRepo     port.RequestRepo
	respRepo    port.ResponseRepo

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
	repeaterFocus   int // 0: Method, 1: URL, 2: Headers, 3: Body, 4: Send
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
		groups:             cfg.Groups,
		notes:              cfg.Notes,
		reqRepo:            cfg.ReqRepo,
		respRepo:           cfg.RespRepo,
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
		g, err := m.groups.List(sid)
		if err != nil {
			return errMsg{err}
		}
		return groupsLoaded{g}
	}
}

func (m Model) loadNotes(gid string) tea.Cmd {
	return func() tea.Msg {
		n, err := m.notes.List(gid)
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Dynamically compute viewport size based on the layout
		sidebarW := 25
		if m.width < 60 {
			sidebarW = 15
		}
		listW := (m.width - sidebarW) / 2
		if listW < 20 {
			listW = 20
		}
		detailW := m.width - sidebarW - listW

		mh := m.height - 2
		tabHeaderHeight := 2
		m.detailPort.Width = max(detailW-2, 10)
		m.detailPort.Height = max(mh-tabHeaderHeight-2, 3)

		m.refreshDetail()
		return m, nil

	case tea.KeyMsg:
		return m.onKey(msg)

	case eventReceived:
		m.events = append(m.events, eventItem{req: msg.req})
		m.eventMap[msg.req.ID] = len(m.events) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
		if m.cursor == len(m.events)-1 || len(m.events) == 1 {
			m.refreshDetail()
		}
		m.setStatus(fmt.Sprintf("← %s %s", msg.req.Method, shorten(msg.req.URL, 50)))
		return m, m.loadResp(msg.req.ID)

	case responseReceived:
		idx, ok := m.eventMap[msg.reqID]
		if !ok {
			return m, nil
		}
		m.events[idx].resp = msg.resp
		if idx == m.cursor {
			m.refreshDetail()
		}
		return m, nil

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
		if m.currentSession == nil {
			if m.initialSessionName != "" {
				var found *model.Session
				for _, s := range msg.sessions {
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
		}
		return m, m.loadGroups(m.currentSession.ID)

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
				g, err := m.groups.Create(m.currentSession.ID, "Notes Group")
				if err != nil {
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

			_, err = m.notes.Create(gid, m.tempNoteTitle, val)
			m.tempNoteTitle = ""
			if err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
				if cmd != nil {
					return m, cmd
				}
				return m, nil
			}
			m.setStatus("Note added")
			if cmd != nil {
				return m, tea.Batch(cmd, m.loadNotes(gid))
			}
			return m, m.loadNotes(gid)

		case inputEditNoteBody:
			err = m.notes.Update(m.tempNoteID, m.tempNoteTitle, val)
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
		}
		return m, nil

	case errMsg:
		m.setStatus(fmt.Sprintf("Error: %v", msg.err))
		return m, nil

	case statusMsg:
		m.setStatus(msg.text)
		return m, nil
	}

	return m, nil
}

type repeaterResult struct {
	resp *model.Response
	err  error
}

type editorFinishedMsg struct {
	mode     inputMode
	filepath string
	err      error
}

func (m *Model) openExternalEditor(mode inputMode, initialContent string) tea.Cmd {
	f, err := os.CreateTemp("", "panoptes-repeater-*.txt")
	if err != nil {
		m.setStatus(fmt.Sprintf("Error creating temp file: %v", err))
		return nil
	}
	defer f.Close()

	if _, err := f.WriteString(initialContent); err != nil {
		m.setStatus(fmt.Sprintf("Error writing temp file: %v", err))
		os.Remove(f.Name())
		return nil
	}

	editor := m.preferredEditor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		if _, err := exec.LookPath("nano"); err == nil {
			editor = "nano"
		} else if _, err := exec.LookPath("vim"); err == nil {
			editor = "vim"
		} else {
			editor = "vi"
		}
	}

	c := exec.Command(editor, f.Name())
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{
			mode:     mode,
			filepath: f.Name(),
			err:      err,
		}
	})
}

func (m Model) sendRepeaterRequest(method, targetURL, rawHeaders, body string) tea.Cmd {
	return func() tea.Msg {
		req, err := http.NewRequest(method, targetURL, strings.NewReader(body))
		if err != nil {
			return repeaterResult{nil, err}
		}
		lines := strings.Split(rawHeaders, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}
		}
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			return repeaterResult{nil, err}
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		headerJSON, _ := json.Marshal(resp.Header)
		decompressed := decompressBody(resp.Header.Get("Content-Encoding"), respBody)

		return repeaterResult{
			resp: &model.Response{
				Status:     resp.Status,
				StatusCode: resp.StatusCode,
				Header:     json.RawMessage(headerJSON),
				Payload:    json.RawMessage(decompressed),
			},
			err: nil,
		}
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

func prettyPrintJSON(raw []byte) string {
	var obj any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}

func (m *Model) buildDetail(idx int) string {
	ev := m.events[idx]
	var b strings.Builder

	switch m.activeTab {
	case tabRequest:
		b.WriteString(detailSection.Render("Request URL & Method") + "\n")
		b.WriteString(methodColor(ev.req.Method).Render(ev.req.Method) + " " + ev.req.URL + "\n\n")

		if ev.req.GroupID != "" {
			groupName := "Unknown Group"
			for _, g := range m.groupList {
				if g.ID == ev.req.GroupID {
					if g.Name != "" {
						groupName = g.Name
					} else {
						groupName = "Group " + g.ID[:min(8, len(g.ID))]
					}
					break
				}
			}
			b.WriteString(detailSection.Render("Group") + "\n")
			b.WriteString("  " + green.Render(groupName) + "\n\n")
		}

		b.WriteString(detailSection.Render("Headers") + "\n")
		var ct string
		if len(ev.req.Header) > 0 {
			var hdrs map[string]json.RawMessage
			if json.Unmarshal(ev.req.Header, &hdrs) == nil {
				for k, v := range hdrs {
					val := strings.Trim(string(v), `"`)
					b.WriteString(fmt.Sprintf("  %s: %s\n", k, val))
					if strings.ToLower(k) == "content-type" {
						ct = val
					}
				}
			}
		} else {
			b.WriteString("  (None)\n")
		}
		b.WriteString("\n")

		b.WriteString(detailSection.Render("Body") + "\n")
		if len(ev.req.Payload) > 0 {
			payloadBytes := []byte(ev.req.Payload)
			if isBinary(ct, payloadBytes) {
				b.WriteString(renderBinaryPlaceholder(ct, len(payloadBytes)))
			} else {
				body := prettyPrintJSON(payloadBytes)
				if len(body) > 10000 {
					body = body[:10000] + "\n… 10k chars truncated"
				}
				b.WriteString(block(body, 2))
			}
			b.WriteString("\n")
		} else {
			b.WriteString("  (Empty Body)\n")
		}

	case tabResponse:
		if ev.resp != nil {
			b.WriteString(detailSection.Render("Status") + "\n")
			b.WriteString(fmt.Sprintf("  %s %s\n\n", statusColor(ev.resp.StatusCode).Render(fmt.Sprintf("%d", ev.resp.StatusCode)), ev.resp.Status))

			b.WriteString(detailSection.Render("Headers") + "\n")
			var ct string
			if len(ev.resp.Header) > 0 {
				var hdrs map[string]json.RawMessage
				if json.Unmarshal(ev.resp.Header, &hdrs) == nil {
					for k, v := range hdrs {
						val := strings.Trim(string(v), `"`)
						b.WriteString(fmt.Sprintf("  %s: %s\n", k, val))
						if strings.ToLower(k) == "content-type" {
							ct = val
						}
					}
				}
			} else {
				b.WriteString("  (None)\n")
			}
			b.WriteString("\n")

			b.WriteString(detailSection.Render("Body") + "\n")
			if len(ev.resp.Payload) > 0 {
				payloadBytes := []byte(ev.resp.Payload)
				if isBinary(ct, payloadBytes) {
					b.WriteString(renderBinaryPlaceholder(ct, len(payloadBytes)))
				} else {
					body := prettyPrintJSON(payloadBytes)
					if len(body) > 10000 {
						body = body[:10000] + "\n… 10k chars truncated"
					}
					b.WriteString(block(body, 2))
				}
				b.WriteString("\n")
			} else {
				b.WriteString("  (Empty Body)\n")
			}
		} else {
			b.WriteString("\n  " + dim.Render("Waiting for response..."))
		}

	case tabNotes:
		b.WriteString(detailSection.Render("Group Notes") + "\n")
		if m.currentGroup != nil {
			if len(m.noteList) > 0 {
				for i, n := range m.noteList {
					var noteStyle lipgloss.Style
					var cursorStr string
					if i == m.noteCursor {
						cursorStr = ">"
						if m.activePane == paneDetail {
							noteStyle = selBg
						} else {
							noteStyle = selBgDim
						}
					} else {
						cursorStr = "•"
					}

					titleLine := fmt.Sprintf(" %s %s ", cursorStr, n.Title)
					if i == m.noteCursor {
						titleLine = "  " + noteStyle.Render(titleLine)
					} else {
						titleLine = "   " + cursorStr + " " + n.Title
					}
					b.WriteString(titleLine + "\n")

					if n.Body != "" {
						bodyLines := strings.Split(n.Body, "\n")
						for _, bl := range bodyLines {
							b.WriteString(fmt.Sprintf("      %s\n", bl))
						}
					}
					b.WriteString("\n")
				}
			} else {
				b.WriteString(dim.Render("  No notes for this group yet.\n  Press 'N' to add a note."))
			}
		} else {
			b.WriteString(dim.Render("  No active group selected.\n  Select a group in the sidebar to view/add notes."))
		}
	}

	return b.String()
}

func (m *Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showInput {
		return m.onInputKey(msg)
	}
	if m.curModal != noModal {
		if m.curModal == modalHelp {
			m.curModal = noModal
			return m, nil
		}
		return m.onModalKey(msg)
	}

	keyStr := msg.String()
	if m.repeaterActive {
		switch keyStr {
		case "esc", "q":
			m.repeaterActive = false
			m.setStatus("Returned to dashboard")
			return m, nil
		case "tab", "down":
			m.repeaterFocus = (m.repeaterFocus + 1) % 5
			return m, nil
		case "up", "shift+tab":
			m.repeaterFocus = (m.repeaterFocus - 1 + 5) % 5
			return m, nil
		case "enter":
			switch m.repeaterFocus {
			case 0: // Method
				m.showInput = true
				m.input = inputState{mode: inputRepeaterMethod, prompt: "Method: ", value: m.repeaterMethod}
			case 1: // URL
				m.showInput = true
				m.input = inputState{mode: inputRepeaterURL, prompt: "URL: ", value: m.repeaterURL}
			case 2: // Headers
				return m, m.openExternalEditor(inputRepeaterHeaders, m.repeaterHeaders)
			case 3: // Body
				return m, m.openExternalEditor(inputRepeaterBody, m.repeaterBody)
			case 4: // Send
				m.repeaterLoading = true
				m.repeaterErr = ""
				return m, m.sendRepeaterRequest(m.repeaterMethod, m.repeaterURL, m.repeaterHeaders, m.repeaterBody)
			}
			return m, nil
		case "ctrl+s":
			m.repeaterLoading = true
			m.repeaterErr = ""
			return m, m.sendRepeaterRequest(m.repeaterMethod, m.repeaterURL, m.repeaterHeaders, m.repeaterBody)
		}
		return m, nil
	}
	switch keyStr {
	case "ctrl+c", "q":
		if m.barrier != nil {
			m.barrier.SetActive(false)
		}
		return m, tea.Quit

	// Pane focus navigation
	case "h", "left":
		if m.activePane == paneDetail {
			m.activePane = paneList
			m.setStatus("List Pane")
		} else if m.activePane == paneList {
			m.activePane = paneSidebar
			m.setStatus("Sidebar Pane")
		}
	case "l", "right":
		if m.activePane == paneSidebar {
			m.activePane = paneList
			m.setStatus("List Pane")
		} else if m.activePane == paneList {
			m.activePane = paneDetail
			m.setStatus("Detail Pane")
		}
	case "tab":
		switch m.activePane {
		case paneSidebar:
			m.activePane = paneList
			m.setStatus("List Pane")
		case paneList:
			m.activePane = paneDetail
			m.setStatus("Detail Pane")
		case paneDetail:
			m.activePane = paneSidebar
			m.setStatus("Sidebar Pane")
		}
	case "shift+tab", "backtab":
		switch m.activePane {
		case paneSidebar:
			m.activePane = paneDetail
			m.setStatus("Detail Pane")
		case paneList:
			m.activePane = paneSidebar
			m.setStatus("Sidebar Pane")
		case paneDetail:
			m.activePane = paneList
			m.setStatus("List Pane")
		}

	// Tab selection inside detail pane
	case "1":
		m.activeTab = tabRequest
		m.refreshDetail()
	case "2":
		m.activeTab = tabResponse
		m.refreshDetail()
	case "3":
		m.activeTab = tabNotes
		m.refreshDetail()

	// Up/Down navigation based on active pane
	case "j", "down":
		switch m.activePane {
		case paneSidebar:
			totalSidebarItems := len(m.groupList) + 1
			if m.sidebarCursor < totalSidebarItems-1 {
				m.sidebarCursor++
			}
		case paneList:
			visible := m.getVisibleEvents()
			if len(visible) > 0 {
				cpos := -1
				for i, v := range visible {
					if v.idx == m.cursor {
						cpos = i
						break
					}
				}
				if cpos >= 0 && cpos < len(visible)-1 {
					m.cursor = visible[cpos+1].idx
					m.refreshDetail()
				}
			}
		case paneDetail:
			if m.activeTab == tabNotes && len(m.noteList) > 0 {
				if m.noteCursor < len(m.noteList)-1 {
					m.noteCursor++
					m.refreshDetail()
				}
			} else {
				m.detailPort.LineDown(3)
			}
		}

	case "k", "up":
		switch m.activePane {
		case paneSidebar:
			if m.sidebarCursor > 0 {
				m.sidebarCursor--
			}
		case paneList:
			visible := m.getVisibleEvents()
			if len(visible) > 0 {
				cpos := -1
				for i, v := range visible {
					if v.idx == m.cursor {
						cpos = i
						break
					}
				}
				if cpos > 0 {
					m.cursor = visible[cpos-1].idx
					m.refreshDetail()
				}
			}
		case paneDetail:
			if m.activeTab == tabNotes && len(m.noteList) > 0 {
				if m.noteCursor > 0 {
					m.noteCursor--
					m.refreshDetail()
				}
			} else {
				m.detailPort.LineUp(3)
			}
		}

	case "enter":
		switch m.activePane {
		case paneSidebar:
			if m.sidebarCursor == 0 {
				m.currentGroup = nil
				m.noteList = nil
				m.noteCursor = 0
				m.setStatus("All Requests selected")
				m.refreshDetail()
			} else if m.sidebarCursor > 0 && m.sidebarCursor <= len(m.groupList) {
				m.currentGroup = m.groupList[m.sidebarCursor-1]
				m.noteCursor = 0
				m.setStatus("Group selected")
				m.refreshDetail()
				return m, m.loadNotes(m.currentGroup.ID)
			}
		case paneList:
			m.activePane = paneDetail
			m.setStatus("Detail Pane")
		case paneDetail:
			if m.activeTab == tabNotes && len(m.noteList) > 0 {
				n := m.noteList[m.noteCursor]
				m.tempNoteID = n.ID
				m.tempNoteTitle = n.Title
				m.showInput = true
				m.input = inputState{
					mode:   inputEditNoteTitle,
					prompt: "Edit note title: ",
					value:  n.Title,
				}
				return m, nil
			}
		}

	case "esc":
		if m.activePane == paneDetail {
			m.activePane = paneList
			m.setStatus("List Pane")
		}

	case "d", "x":
		if m.activePane == paneDetail && m.activeTab == tabNotes && len(m.noteList) > 0 {
			n := m.noteList[m.noteCursor]
			err := m.notes.Delete(n.ID)
			if err != nil {
				m.setStatus(fmt.Sprintf("Error deleting note: %v", err))
				return m, nil
			}
			m.setStatus("Note deleted")
			if m.currentGroup != nil {
				return m, m.loadNotes(m.currentGroup.ID)
			}
			return m, nil
		}

	// Modal / Session creation / Note creation keys
	case "E":
		m.showInput = true
		m.input = inputState{mode: inputPreferredEditor, prompt: "Preferred Editor: ", value: m.preferredEditor}
	case "e":
		if m.activePane == paneDetail && m.activeTab == tabNotes && len(m.noteList) > 0 {
			n := m.noteList[m.noteCursor]
			m.tempNoteID = n.ID
			m.tempNoteTitle = n.Title
			m.showInput = true
			m.input = inputState{
				mode:   inputEditNoteTitle,
				prompt: "Edit note title: ",
				value:  n.Title,
			}
			return m, nil
		}
		if m.cursor >= 0 && m.cursor < len(m.events) {
			req := m.events[m.cursor].req
			m.repeaterMethod = req.Method
			m.repeaterURL = req.URL
			// Format headers
			var hdrs map[string]json.RawMessage
			var hdrLines []string
			if json.Unmarshal(req.Header, &hdrs) == nil {
				for k, v := range hdrs {
					val := strings.Trim(string(v), `"`)
					hdrLines = append(hdrLines, fmt.Sprintf("%s: %s", k, val))
				}
			}
			m.repeaterHeaders = strings.Join(hdrLines, "\n")
			m.repeaterBody = string(req.Payload)
			m.repeaterResp = nil
			m.repeaterActive = true
			m.repeaterFocus = 0
			m.repeaterLoading = false
			m.repeaterErr = ""
			m.setStatus("Loaded request in Repeater")
		} else {
			m.setStatus("No request selected")
		}
	case "n":
		m.showInput = true
		m.input = inputState{mode: inputNewSession, prompt: "Session name: "}
	case "s":
		if len(m.sessionList) > 0 {
			items := make([]string, len(m.sessionList))
			for i, s := range m.sessionList {
				items[i] = s.Name
			}
			m.curModal = modalSessions
			m.pick = picker{title: "Sessions", items: items}
		}
	case "c":
		if m.currentSession != nil {
			m.showInput = true
			m.input = inputState{mode: inputNewGroup, prompt: "Group name: "}
		} else {
			m.setStatus("No session (press n)")
		}
	case "g":
		if len(m.groupList) > 0 {
			// Calculate request count per group
			groupCounts := make(map[string]int)
			for _, ev := range m.events {
				if ev.req.GroupID != "" {
					groupCounts[ev.req.GroupID]++
				}
			}

			items := make([]string, len(m.groupList))
			for i, g := range m.groupList {
				shortID := g.ID
				if len(shortID) > 8 {
					shortID = shortID[:8]
				}
				count := groupCounts[g.ID]
				displayName := g.Name
				if displayName == "" {
					displayName = fmt.Sprintf("Group %d", i+1)
				}
				items[i] = fmt.Sprintf("%s (%s) [%d requests]", displayName, shortID, count)
			}
			m.curModal = modalGroups
			m.pick = picker{title: "Select Group", items: items, cursor: 0}
		} else {
			m.setStatus("No groups (press c)")
		}
	case "a":
		if m.cursor < 0 || m.cursor >= len(m.events) {
			m.setStatus("No request selected")
			return m, nil
		}
		if m.currentSession == nil {
			m.setStatus("No active session")
			return m, nil
		}

		// Calculate request count per group
		groupCounts := make(map[string]int)
		for _, ev := range m.events {
			if ev.req.GroupID != "" {
				groupCounts[ev.req.GroupID]++
			}
		}

		var items []string
		for i, g := range m.groupList {
			shortID := g.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			count := groupCounts[g.ID]
			displayName := g.Name
			if displayName == "" {
				displayName = fmt.Sprintf("Group %d", i+1)
			}
			items = append(items, fmt.Sprintf("%s (%s) [%d requests]", displayName, shortID, count))
		}
		items = append(items, "(Unassign from Group)")
		items = append(items, "+ Create New Group")

		m.curModal = modalAssignGroup
		m.pick = picker{title: "Assign Request to Group", items: items, cursor: 0}
		return m, nil
	case "A":
		if m.cursor < 0 || m.cursor >= len(m.events) {
			m.setStatus("No request selected")
			return m, nil
		}
		reqID := m.events[m.cursor].req.ID
		if m.currentGroup != nil {
			if err := m.reqRepo.UpdateGroup(context.Background(), reqID, m.currentGroup.ID); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = m.currentGroup.ID
				m.refreshDetail()
				displayName := m.currentGroup.Name
				if displayName == "" {
					displayName = "Group " + m.currentGroup.ID[:min(8, len(m.currentGroup.ID))]
				}
				m.setStatus("Assigned to active group: " + displayName)
			}
		} else {
			// Unassign
			if err := m.reqRepo.UpdateGroup(context.Background(), reqID, ""); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = ""
				m.refreshDetail()
				m.setStatus("Unassigned from group")
			}
		}
		return m, nil
	case "N":
		m.activeTab = tabNotes
		m.showInput = true
		m.input = inputState{mode: inputAddNoteTitle, prompt: "Note title: "}

	case "/":
		m.showInput = true
		m.input = inputState{mode: inputFilter, prompt: "Filter: "}
	case "p":
		m.paused = !m.paused
		m.setStatus(ifelse(m.paused, "PAUSED", "Resumed"))
	case "r":
		if m.barrier != nil {
			m.barrier.SetActive(true)
			m.setStatus("DROP mode")
		}
	case "R":
		if m.barrier != nil {
			m.barrier.SetActive(false)
			m.setStatus("FORWARD mode")
		}
	case "?":
		m.curModal = modalHelp
	}

	return m, nil
}

func (m *Model) onInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.showInput = false
		m.input = inputState{}
		m.tempNoteTitle = ""
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.input.value)
		mode := m.input.mode
		m.showInput = false
		m.input = inputState{}

		switch mode {
		case inputRepeaterMethod:
			m.repeaterMethod = val
			m.setStatus("Updated Method")
			return m, nil
		case inputRepeaterURL:
			m.repeaterURL = val
			m.setStatus("Updated URL")
			return m, nil
		case inputRepeaterHeaders:
			m.repeaterHeaders = strings.ReplaceAll(val, "\\n", "\n")
			m.setStatus("Updated Headers")
			return m, nil
		case inputRepeaterBody:
			m.repeaterBody = strings.ReplaceAll(val, "\\n", "\n")
			m.setStatus("Updated Body")
			return m, nil

		case inputPreferredEditor:
			m.preferredEditor = val
			m.setStatus("Preferred editor set to: " + val)
			return m, nil

		case inputNewSession:
			if val == "" {
				return m, nil
			}
			s, err := m.sessions.Create(val)
			if err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
				return m, nil
			} else {
				m.setStatus("Session: " + val)
				return m, tea.Batch(m.selectSession(s), m.loadSessions())
			}

		case inputFilter:
			m.filter = val
			if val != "" {
				visible := m.getVisibleEvents()
				if len(visible) > 0 {
					m.cursor = visible[0].idx
				}
			}
			m.refreshDetail()
			m.setStatus(ifelse(val != "", "Filter: "+val, "Filter cleared"))

		case inputNewGroup:
			if m.currentSession != nil {
				gName := val
				if gName == "" {
					gName = "New Group"
				}
				g, err := m.groups.Create(m.currentSession.ID, gName)
				if err != nil {
					m.setStatus(fmt.Sprintf("Error: %v", err))
				} else {
					m.currentGroup = g
					m.setStatus("Group created: " + gName)
					if m.tempAssignRequestID != "" {
						reqID := m.tempAssignRequestID
						m.tempAssignRequestID = ""
						if err := m.reqRepo.UpdateGroup(context.Background(), reqID, g.ID); err != nil {
							m.setStatus(fmt.Sprintf("Error assigning request: %v", err))
						} else {
							// Update local event group ID
							for i, ev := range m.events {
								if ev.req.ID == reqID {
									m.events[i].req.GroupID = g.ID
									break
								}
							}
							m.refreshDetail()
							m.setStatus("Created group & assigned request")
						}
					}
					return m, m.loadGroups(m.currentSession.ID)
				}
			}

		case inputAssignGroup:
			if val == "" || len(m.groupList) == 0 || m.cursor < 0 || m.cursor >= len(m.events) {
				return m, nil
			}
			var gi int
			if _, err := fmt.Sscanf(val, "%d", &gi); err != nil || gi < 0 || gi >= len(m.groupList) {
				m.setStatus("Invalid index")
				return m, nil
			}
			g := m.groupList[gi]
			if err := m.reqRepo.UpdateGroup(context.Background(), m.events[m.cursor].req.ID, g.ID); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = g.ID
				m.refreshDetail()
				displayName := g.Name
				if displayName == "" {
					displayName = "Group " + g.ID[:min(8, len(g.ID))]
				}
				m.setStatus("Assigned to group " + displayName)
			}

		case inputAddNoteTitle:
			if val == "" {
				return m, nil
			}
			m.tempNoteTitle = val
			return m, m.openExternalEditor(inputAddNoteBody, "")

		case inputEditNoteTitle:
			if val == "" {
				return m, nil
			}
			m.tempNoteTitle = val
			var initialContent string
			for _, n := range m.noteList {
				if n.ID == m.tempNoteID {
					initialContent = n.Body
					break
				}
			}
			return m, m.openExternalEditor(inputEditNoteBody, initialContent)
		}
		return m, nil

	default:
		key := msg.String()
		if key == "backspace" {
			if len(m.input.value) > 0 {
				m.input.value = m.input.value[:len(m.input.value)-1]
			}
		} else if key == "space" || key == " " {
			m.input.value += " "
		} else if len(key) == 1 {
			m.input.value += key
		}
		return m, nil
	}
}

func (m *Model) onModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	total := len(m.pick.items)
	switch msg.String() {
	case "j", "down":
		if m.pick.cursor < total-1 {
			m.pick.cursor++
		}
	case "k", "up":
		if m.pick.cursor > 0 {
			m.pick.cursor--
		}
	case "enter":
		sel := m.pick.cursor
		modal := m.curModal
		m.curModal = noModal
		return m, m.pickAction(modal, sel)
	case "esc", "q":
		m.curModal = noModal
	}
	return m, nil
}

func (m *Model) pickAction(modal modal, sel int) tea.Cmd {
	switch modal {
	case modalSessions:
		if sel < 0 || sel >= len(m.sessionList) {
			return nil
		}
		s := m.sessionList[sel]
		m.sessions.Load(s.ID)
		m.interceptor.SetActiveSession(s.ID)
		m.currentSession = s
		m.currentGroup = nil
		m.cursor = -1
		m.events = nil
		m.eventMap = map[string]int{}
		m.detailLines = ""
		m.detailPort.SetContent("")
		m.setStatus("Session: " + s.Name)
		return tea.Batch(m.loadGroups(s.ID), m.loadEvents())

	case modalGroups:
		if sel < 0 || sel >= len(m.groupList) {
			return nil
		}
		m.currentGroup = m.groupList[sel]
		m.setStatus("Group selected")
		m.refreshDetail()
		return m.loadNotes(m.currentGroup.ID)

	case modalAssignGroup:
		if m.cursor < 0 || m.cursor >= len(m.events) {
			return nil
		}
		reqID := m.events[m.cursor].req.ID
		N := len(m.groupList)

		if sel == N {
			// Unassign from group
			if err := m.reqRepo.UpdateGroup(context.Background(), reqID, ""); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = ""
				m.refreshDetail()
				m.setStatus("Unassigned from group")
			}
			return nil
		}

		if sel == N+1 {
			// Create New Group and Assign
			if m.currentSession == nil {
				m.setStatus("No active session")
				return nil
			}
			m.tempAssignRequestID = reqID
			m.showInput = true
			m.input = inputState{mode: inputNewGroup, prompt: "Group name: "}
			return nil
		}

		if sel < 0 || sel >= N {
			return nil
		}
		g := m.groupList[sel]
		if err := m.reqRepo.UpdateGroup(context.Background(), reqID, g.ID); err != nil {
			m.setStatus(fmt.Sprintf("Error: %v", err))
		} else {
			m.events[m.cursor].req.GroupID = g.ID
			m.refreshDetail()
			displayName := g.Name
			if displayName == "" {
				displayName = "Group " + g.ID[:min(8, len(g.ID))]
			}
			m.setStatus("Assigned to group " + displayName)
		}
	}
	return nil
}

func (m *Model) matchFilter(ev eventItem) bool {
	f := strings.ToLower(m.filter)
	if strings.Contains(strings.ToLower(ev.req.URL), f) {
		return true
	}
	if strings.Contains(strings.ToLower(ev.req.Method), f) {
		return true
	}
	if ev.resp != nil && strings.Contains(fmt.Sprint(ev.resp.StatusCode), f) {
		return true
	}
	return false
}

func (m *Model) setStatus(text string) {
	m.statusMsg = text
	m.statusExpiry = time.Now().Add(4 * time.Second)
}

func (m Model) getVisibleEvents() []visibleRow {
	var visible []visibleRow
	for i, ev := range m.events {
		if m.currentGroup != nil && ev.req.GroupID != m.currentGroup.ID {
			continue
		}
		if m.filter != "" && !m.matchFilter(ev) {
			continue
		}
		visible = append(visible, visibleRow{idx: i, item: ev})
	}
	return visible
}

// ── View ─────────────────────────────────────────────────

func (m Model) View() string {
	if !m.ready {
		return "Loading…"
	}
	w := m.width
	h := m.height
	mh := h - 2

	var buf strings.Builder
	buf.WriteString(m.topBar(w))
	buf.WriteString("\n")

	if m.curModal == modalHelp {
		buf.WriteString(m.renderHelp(min(w-4, 60), min(mh-2, 22)))
		buf.WriteString("\n")
	} else if m.curModal != noModal {
		buf.WriteString(m.renderPicker(mh))
		buf.WriteString("\n")
	} else if m.showInput {
		buf.WriteString(m.renderInput(w, mh))
		buf.WriteString("\n")
	} else if m.repeaterActive {
		buf.WriteString(m.renderRepeater(w, mh))
		buf.WriteString("\n")
	} else {
		// Calculate widths for the 3 columns
		sidebarW := 25
		if w < 60 {
			sidebarW = 15
		}
		listW := (w - sidebarW) / 2
		if listW < 20 {
			listW = 20
		}
		detailW := w - sidebarW - listW
		if detailW < 20 {
			detailW = 20
			listW = w - sidebarW - detailW
		}

		var sStyle, lStyle, dStyle lipgloss.Style
		if m.activePane == paneSidebar {
			sStyle = focusedBorder
		} else {
			sStyle = unfocusedBorder
		}

		if m.activePane == paneList {
			lStyle = focusedBorder
		} else {
			lStyle = unfocusedBorder
		}

		if m.activePane == paneDetail {
			dStyle = focusedBorder
		} else {
			dStyle = unfocusedBorder
		}

		sidebarView := sStyle.Width(sidebarW - 2).Height(mh - 2).Render(m.renderSidebar(sidebarW - 2, mh - 2))
		listView := lStyle.Width(listW - 2).Height(mh - 2).Render(m.renderList(listW - 2, mh - 2))
		detailView := dStyle.Width(detailW - 2).Height(mh - 2).Render(m.renderDetail(detailW - 2, mh - 2))

		body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, listView, detailView)
		buf.WriteString(body)
		buf.WriteString("\n")
	}

	buf.WriteString(m.statusBar(w))
	return buf.String()
}

func (m Model) topBar(w int) string {
	left := topBar.Render(" panoptes ")
	if m.currentSession != nil {
		left += topBar.Render(accent.Render(" ◆ " + m.currentSession.Name + " "))
	}
	if m.currentGroup != nil {
		displayName := m.currentGroup.Name
		if displayName == "" {
			displayName = m.currentGroup.ID[:min(8, len(m.currentGroup.ID))]
		}
		left += topBar.Render(green.Render(" ▸ " + displayName + " "))
	}
	left += topBar.Render(blue.Render(fmt.Sprintf(" %d events ", len(m.events))))

	var right string
	if m.paused {
		right += topBar.Render(" PAUSED ")
	}
	right += topBar.Render(" ?help ")

	fill := w - lipgloss.Width(left) - lipgloss.Width(right)
	if fill < 0 {
		fill = 0
	}
	return left + topBar.Render(strings.Repeat(" ", fill)) + right
}

func (m Model) statusBar(w int) string {
	var left, right strings.Builder

	left.WriteString(bar.Render(" Tab:Focus "))
	left.WriteString(bar.Render(" ←/→/nav "))
	left.WriteString(bar.Render(" j/k:scroll "))
	left.WriteString(bar.Render(" 1/2/3:Tabs "))
	left.WriteString(bar.Render(" n:ses "))
	left.WriteString(bar.Render(" c:grp "))
	left.WriteString(bar.Render(" N:note "))
	left.WriteString(bar.Render(" /:filt "))
	left.WriteString(bar.Render(" r:drop "))
	left.WriteString(bar.Render(" R:fwd "))
	left.WriteString(bar.Render(" ?:help "))
	left.WriteString(bar.Render(" q:quit "))

	if m.statusMsg != "" && time.Now().Before(m.statusExpiry) {
		right.WriteString(bar.Render(" " + m.statusMsg + " "))
	}

	fill := w - lipgloss.Width(left.String()) - lipgloss.Width(right.String())
	if fill < 0 {
		fill = 0
	}
	return left.String() + bar.Render(strings.Repeat(" ", fill)) + right.String()
}

func (m Model) renderSidebar(w, h int) string {
	var b strings.Builder

	b.WriteString(sidebarHeaderStyle.Render("● SESSIONS") + "\n")
	if m.currentSession != nil {
		b.WriteString(fmt.Sprintf("  ◆ %s\n\n", m.currentSession.Name))
	} else {
		b.WriteString("  (None)\n\n")
	}

	b.WriteString(sidebarHeaderStyle.Render("● GROUPS") + "\n")

	var items []string
	items = append(items, "All Requests")
	for _, g := range m.groupList {
		displayName := g.Name
		if displayName == "" {
			id := g.ID
			if len(id) > 8 {
				id = id[:8]
			}
			displayName = "Group " + id
		}
		items = append(items, displayName)
	}

	for i, item := range items {
		isActive := false
		if m.currentGroup == nil && i == 0 {
			isActive = true
		} else if m.currentGroup != nil && i > 0 && m.groupList[i-1].ID == m.currentGroup.ID {
			isActive = true
		}

		cursorStr := "  "
		if i == m.sidebarCursor {
			cursorStr = "▸ "
		}

		displayStr := item
		if isActive {
			displayStr = "✔ " + item
		} else {
			displayStr = "  " + item
		}

		line := cursorStr + displayStr
		if len(line) > w {
			line = line[:w]
		}

		if i == m.sidebarCursor && m.activePane == paneSidebar {
			b.WriteString(invBg.Render(line) + "\n")
		} else if i == m.sidebarCursor {
			b.WriteString(selBgDim.Width(w).Render(line) + "\n")
		} else if isActive {
			b.WriteString(accent.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}

func (m Model) renderList(w, h int) string {
	visible := m.getVisibleEvents()

	if len(visible) == 0 {
		msg := "  No events"
		if m.filter != "" {
			msg = "  No matches"
		} else if m.currentGroup != nil {
			msg = "  No events in group"
		}
		return msg + "\n" + strings.Repeat("\n", h-2)
	}

	cpos := 0
	found := false
	for i, v := range visible {
		if v.idx == m.cursor {
			cpos = i
			found = true
			break
		}
	}
	if !found && len(visible) > 0 {
		m.cursor = visible[0].idx
		cpos = 0
	}

	half := h / 2
	start := cpos - half
	if start < 0 {
		start = 0
	}
	end := start + h
	if end > len(visible) {
		end = len(visible)
		start = end - h
		if start < 0 {
			start = 0
		}
	}

	var buf strings.Builder
	for _, v := range visible[start:end] {
		ev := v.item
		code := ""
		if ev.resp != nil {
			code = fmt.Sprintf("%d", ev.resp.StatusCode)
		}
		url := ev.req.URL
		if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			if p := strings.SplitN(url, "/", 4); len(p) >= 4 {
				url = "/" + p[3]
			}
		}
		url = shorten(url, w-18)

		line := fmt.Sprintf(" %s %s %s",
			methodColor(ev.req.Method).Render(fmt.Sprintf("%-6s", ev.req.Method)),
			code,
			url,
		)

		if v.idx == m.cursor && m.activePane == paneList {
			line = selBg.Width(w).Render(line)
		} else if v.idx == m.cursor {
			line = selBgDim.Width(w).Render(line)
		}
		buf.WriteString(line + "\n")
	}

	for i := end - start; i < h; i++ {
		buf.WriteString("\n")
	}

	return buf.String()
}

type visibleRow struct {
	idx  int
	item eventItem
}

func (m Model) renderDetail(w, h int) string {
	if m.curModal != noModal || m.showInput {
		return ""
	}

	if m.cursor < 0 || m.cursor >= len(m.events) {
		return dim.Render("  Select an event to view details")
	}

	var b strings.Builder

	reqTab := " [1] Request "
	respTab := " [2] Response "
	notesTab := " [3] Notes "

	if m.activeTab == tabRequest {
		reqTab = activeTabStyle.Render("[1] Request")
	} else {
		reqTab = inactiveTabStyle.Render("[1] Request")
	}

	if m.activeTab == tabResponse {
		respTab = activeTabStyle.Render("[2] Response")
	} else {
		respTab = inactiveTabStyle.Render("[2] Response")
	}

	if m.activeTab == tabNotes {
		notesTab = activeTabStyle.Render("[3] Notes")
	} else {
		notesTab = inactiveTabStyle.Render("[3] Notes")
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Top, reqTab, " ", respTab, " ", notesTab)
	b.WriteString(tabs + "\n")
	b.WriteString(subtleFore.Render(strings.Repeat("─", w)) + "\n")

	m.detailPort.Width = max(w, 10)
	m.detailPort.Height = max(h-3, 3)

	m.detailPort.SetContent(m.detailLines)
	return b.String() + m.detailPort.View()
}

func (m Model) renderHelp(w, h int) string {
	var b strings.Builder
	b.WriteString(accent.Render("   Panoptes Help & Shortcut Keys") + "\n")
	b.WriteString(subtleFore.Render(strings.Repeat("─", w-4)) + "\n\n")

	keys := [][]string{
		{"Tab / Shift+Tab", "Cycle active pane focus"},
		{"h/l or Left/Right", "Move focus Sidebar/List/Detail"},
		{"j/k or Up/Down", "Navigate list / Sidebar, scroll detail"},
		{"1 / 2 / 3", "Switch to Request/Response/Notes tab"},
		{"e", "Open selected request in Repeater"},
		{"E", "Configure preferred editor"},
		{"n", "Create new session"},
		{"s", "Switch sessions"},
		{"c", "Create new group"},
		{"g", "Select group"},
		{"a", "Assign request to a group (modal)"},
		{"A", "Assign request to active group / Unassign"},
		{"N", "Add a note to active group"},
		{"e / Enter", "Edit note (on Notes tab)"},
		{"d / x", "Delete note (on Notes tab)"},
		{"/", "Filter requests by text"},
		{"r", "Enable Drop mode"},
		{"R", "Enable Forward mode"},
		{"q / Ctrl+C", "Quit application"},
	}

	for _, k := range keys {
		b.WriteString(fmt.Sprintf("  %s%-18s%s%s\n", accent.Render("• "), k[0], dim.Render("─ "), k[1]))
	}

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Render(b.String())
}

func (m Model) renderPicker(h int) string {
	maxW := min(m.width-8, 48)
	maxH := min(h-2, len(m.pick.items)+3)

	var b strings.Builder
	b.WriteString(accent.Render(m.pick.title) + "\n")
	b.WriteString(strings.Repeat("─", maxW-2) + "\n")
	for i, item := range m.pick.items {
		if i > maxH-4 {
			b.WriteString("  …")
			break
		}
		cursor := "  "
		if i == m.pick.cursor {
			cursor = "▸ "
		}
		line := cursor + item
		if i == m.pick.cursor {
			b.WriteString(invBg.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return lipgloss.NewStyle().
		Width(maxW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1).
		Render(b.String())
}

func (m Model) renderInput(w, h int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(0, 1).
		Render(m.input.prompt + m.input.value + "▌")
}

// ── Styles ───────────────────────────────────────────────

var (
	accentColor = lipgloss.Color("#7B56DB")
	accent      = lipgloss.NewStyle().Foreground(accentColor)
	accentFore  = lipgloss.NewStyle().Foreground(accentColor)
	subtleFore  = lipgloss.NewStyle().Foreground(lipgloss.Color("#383838"))

	topBar = lipgloss.NewStyle().
		Height(1).Padding(0, 1).Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#2D2D2D"))

	bar = lipgloss.NewStyle().
		Height(1).Padding(0, 1).
		Foreground(lipgloss.Color("#BBBBBB")).
		Background(lipgloss.Color("#1A1A1A"))

	selBg = lipgloss.NewStyle().
		Background(lipgloss.Color("#3D3D3D")).
		Foreground(lipgloss.Color("#FAFAFA"))

	selBgDim = lipgloss.NewStyle().
		Background(lipgloss.Color("#2A2A2A")).
		Foreground(lipgloss.Color("#CCCCCC"))

	invBg = lipgloss.NewStyle().
		Background(lipgloss.Color("#7B56DB")).
		Foreground(lipgloss.Color("#FAFAFA"))

	dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	detailSection = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7B56DB"))

	green  = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	blue   = lipgloss.NewStyle().Foreground(lipgloss.Color("#64B5F6"))
	orange = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB74D"))
	red    = lipgloss.NewStyle().Foreground(lipgloss.Color("#F06292"))
	grey   = lipgloss.NewStyle().Foreground(lipgloss.Color("#BBBBBB"))

	statusGood = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	statusMid  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB74D"))
	statusBad  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F06292"))

	focusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)

	unfocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#303030"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(accentColor).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Background(lipgloss.Color("#222222")).
			Padding(0, 1)

	sidebarHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor)
)

func methodColor(m string) lipgloss.Style {
	switch m {
	case "GET":
		return green
	case "POST":
		return blue
	case "PUT":
		return orange
	case "DELETE":
		return red
	default:
		return grey
	}
}

func statusColor(c int) lipgloss.Style {
	switch {
	case c >= 200 && c < 300:
		return statusGood
	case c >= 300 && c < 400:
		return statusMid
	case c >= 400:
		return statusBad
	default:
		return statusGood
	}
}

func shorten(s string, n int) string {
	if n < 1 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func block(s string, indent int) string {
	pre := strings.Repeat(" ", indent)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pre + l
	}
	return strings.Join(lines, "\n")
}

func ifelse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func decompressBody(encoding string, body []byte) []byte {
	encoding = strings.ToLower(encoding)
	if strings.Contains(encoding, "gzip") {
		gr, err := gzip.NewReader(bytes.NewReader(body))
		if err == nil {
			defer gr.Close()
			decompressed, err := io.ReadAll(gr)
			if err == nil {
				return decompressed
			}
		}
	} else if strings.Contains(encoding, "deflate") {
		zr, err := zlib.NewReader(bytes.NewReader(body))
		if err == nil {
			defer zr.Close()
			decompressed, err := io.ReadAll(zr)
			if err == nil {
				return decompressed
			}
		}
		fr := flate.NewReader(bytes.NewReader(body))
		defer fr.Close()
		decompressed, err := io.ReadAll(fr)
		if err == nil {
			return decompressed
		}
	} else if strings.Contains(encoding, "br") {
		br := brotli.NewReader(bytes.NewReader(body))
		decompressed, err := io.ReadAll(br)
		if err == nil {
			return decompressed
		}
	}
	return body
}

func (m Model) renderRepeater(w, h int) string {
	colW := w / 2
	if colW < 20 {
		colW = 20
	}
	colH := h - 2

	var leftBuf strings.Builder
	leftBuf.WriteString(detailSection.Render(" REQUEST EDITOR ") + "\n\n")

	// Method
	methodStyle := lipgloss.NewStyle()
	if m.repeaterFocus == 0 {
		methodStyle = methodStyle.Background(accentColor).Foreground(lipgloss.Color("#FAFAFA"))
	}
	leftBuf.WriteString("Method (Press Enter to edit):\n" + methodStyle.Render(" "+m.repeaterMethod+" ") + "\n\n")

	// URL
	urlStyle := lipgloss.NewStyle()
	if m.repeaterFocus == 1 {
		urlStyle = urlStyle.Background(accentColor).Foreground(lipgloss.Color("#FAFAFA"))
	}
	leftBuf.WriteString("URL (Press Enter to edit):\n" + urlStyle.Render(" "+m.repeaterURL+" ") + "\n\n")

	// Headers
	headersStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#303030"))
	if m.repeaterFocus == 2 {
		headersStyle = headersStyle.BorderForeground(accentColor)
	}
	hdrVal := m.repeaterHeaders
	if hdrVal == "" {
		hdrVal = "(Empty Headers)"
	}
	leftBuf.WriteString("Headers (Press Enter to edit):\n" + headersStyle.Width(colW-4).Render(hdrVal) + "\n\n")

	// Body
	bodyStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#303030"))
	if m.repeaterFocus == 3 {
		bodyStyle = bodyStyle.BorderForeground(accentColor)
	}
	bodyVal := m.repeaterBody
	if bodyVal == "" {
		bodyVal = "(Empty Body)"
	}
	leftBuf.WriteString("Body (Press Enter to edit):\n" + bodyStyle.Width(colW-4).Render(bodyVal) + "\n\n")

	// Send Button / Status
	sendStyle := lipgloss.NewStyle().Padding(0, 2)
	if m.repeaterFocus == 4 {
		sendStyle = sendStyle.Background(lipgloss.Color("#73F59F")).Foreground(lipgloss.Color("#121212")).Bold(true)
	} else {
		sendStyle = sendStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#73F59F")).Foreground(lipgloss.Color("#73F59F"))
	}
	leftBuf.WriteString(sendStyle.Render("SEND REQUEST (Ctrl+S)"))
	if m.repeaterLoading {
		leftBuf.WriteString("  " + orange.Render("Loading..."))
	}
	if m.repeaterErr != "" {
		leftBuf.WriteString("  " + red.Render("Error: "+m.repeaterErr))
	}
	leftBuf.WriteString("\n\n" + dim.Render(" ℹ Navigate: [Tab/Arrows] • Edit: [Enter] • Send: [Ctrl+S] • Exit: [Esc/q]"))

	// Right Column: Response
	var rightBuf strings.Builder
	rightBuf.WriteString(detailSection.Render(" RESPONSE VIEW ") + "\n\n")

	if m.repeaterResp != nil {
		resp := m.repeaterResp
		rightBuf.WriteString("Status:\n" + statusColor(resp.StatusCode).Render(fmt.Sprintf(" %d %s ", resp.StatusCode, resp.Status)) + "\n\n")

		// Headers
		var hdrs map[string]json.RawMessage
		var ct string
		rightBuf.WriteString("Headers:\n")
		if json.Unmarshal(resp.Header, &hdrs) == nil {
			for k, v := range hdrs {
				val := strings.Trim(string(v), `"`)
				rightBuf.WriteString(fmt.Sprintf("  %s: %s\n", k, val))
				if strings.ToLower(k) == "content-type" {
					ct = val
				}
			}
		}
		rightBuf.WriteString("\n")

		// Body
		rightBuf.WriteString("Body:\n")
		payloadBytes := []byte(resp.Payload)
		if isBinary(ct, payloadBytes) {
			rightBuf.WriteString(renderBinaryPlaceholder(ct, len(payloadBytes)) + "\n")
		} else {
			bodyStr := prettyPrintJSON(payloadBytes)
			rightBuf.WriteString(block(bodyStr, 2) + "\n")
		}
	} else {
		rightBuf.WriteString(dim.Render("No response yet. Focus Send and press Enter or press Ctrl+S to fire."))
	}

	leftCol := focusedBorder.Width(colW - 2).Height(colH).Render(leftBuf.String())
	rightCol := unfocusedBorder.Width(w - colW - 2).Height(colH).Render(rightBuf.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)
}

func isBinary(contentType string, payload []byte) bool {
	if len(payload) == 0 {
		return false
	}
	ct := strings.ToLower(contentType)
	if strings.HasPrefix(ct, "image/") ||
		strings.HasPrefix(ct, "audio/") ||
		strings.HasPrefix(ct, "video/") ||
		strings.Contains(ct, "octet-stream") ||
		strings.Contains(ct, "zip") ||
		strings.Contains(ct, "pdf") ||
		strings.Contains(ct, "binary") {
		return true
	}
	if !utf8.Valid(payload) {
		return true
	}
	checkLen := len(payload)
	if checkLen > 1024 {
		checkLen = 1024
	}
	nonPrintable := 0
	for i := 0; i < checkLen; i++ {
		c := payload[i]
		if c == 0 {
			return true
		}
		if (c < 32 && c != '\n' && c != '\r' && c != '\t') || c > 126 {
			nonPrintable++
		}
	}
	if checkLen > 0 && float64(nonPrintable)/float64(checkLen) > 0.10 {
		return true
	}
	return false
}

func renderBinaryPlaceholder(contentType string, size int) string {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	sizeStr := formatSize(size)
	var b strings.Builder
	b.WriteString("  ┌──────────────────────────────────────────────────┐\n")
	b.WriteString("  │                  BINARY CONTENT                  │\n")
	b.WriteString("  │                                                  │\n")
	b.WriteString(fmt.Sprintf("  │  Type: %-42s│\n", contentType))
	b.WriteString(fmt.Sprintf("  │  Size: %-42s│\n", sizeStr))
	b.WriteString("  │                                                  │\n")
	b.WriteString("  │  [Binary payload cannot be displayed as text]    │\n")
	b.WriteString("  └──────────────────────────────────────────────────┘\n")
	return b.String()
}

func formatSize(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
