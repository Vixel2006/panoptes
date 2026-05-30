package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

func (m *Model) onKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showInput {
		return m.onInputKey(msg)
	}
	if m.curModal != noModal {
		return m.onModalKey(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		if m.repeaterActive {
			m.repeaterActive = false
			m.repeaterResp = nil
			m.repeaterErr = ""
			m.setStatus("Repeater closed")
			return m, nil
		}
		return m, tea.Quit

	case "tab":
		if m.repeaterActive {
			m.repeaterFocus = (m.repeaterFocus + 1) % 5
			m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			return m, nil
		}
		m.activePane = (m.activePane + 1) % 3
		m.setStatus(ifelse(m.activePane == paneSidebar, "Sidebar", ifelse(m.activePane == paneList, "List Pane", "Detail Pane")))
		if m.activePane == paneDetail {
			m.refreshDetail()
		}

	case "shift+tab":
		if m.repeaterActive {
			m.repeaterFocus = (m.repeaterFocus + 4) % 5
			m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			return m, nil
		}
		m.activePane = (m.activePane + 2) % 3
		m.setStatus(ifelse(m.activePane == paneSidebar, "Sidebar", ifelse(m.activePane == paneList, "List Pane", "Detail Pane")))

	case "left", "h":
		if m.repeaterActive {
			if m.repeaterFocus > 0 {
				m.repeaterFocus--
				m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			}
			return m, nil
		}
		if m.activePane > 0 {
			m.activePane--
			m.setStatus(ifelse(m.activePane == paneSidebar, "Sidebar", ifelse(m.activePane == paneList, "List Pane", "Detail Pane")))
		}

	case "right", "l":
		if m.repeaterActive {
			if m.repeaterFocus < 4 {
				m.repeaterFocus++
				m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			}
			return m, nil
		}
		if m.activePane < 2 {
			m.activePane++
			m.setStatus(ifelse(m.activePane == paneSidebar, "Sidebar", ifelse(m.activePane == paneList, "List Pane", "Detail Pane")))
		}

	case "1":
		m.activeTab = tabRequest
		m.refreshDetail()
	case "2":
		m.activeTab = tabResponse
		m.refreshDetail()
	case "3":
		m.activeTab = tabNotes
		m.refreshDetail()

	case "e":
		if m.cursor >= 0 && m.cursor < len(m.events) {
			ev := m.events[m.cursor]
			m.repeaterActive = true
			m.repeaterMethod = ev.req.Method
			m.repeaterURL = ev.req.URL
			// Parse headers from stored JSON
			var hdrs map[string]string
			m.repeaterHeaders = ""
			if json.Unmarshal(ev.req.Header, &hdrs) == nil {
				for k, v := range hdrs {
					m.repeaterHeaders += k + ": " + v + "\n"
				}
			}
			m.repeaterBody = string(ev.req.Payload)
			m.repeaterResp = ev.resp
			m.repeaterFocus = 0
			m.repeaterLoading = false
			m.repeaterErr = ""
			m.setStatus("Repeater: Edit request and send (Ctrl+S)")
		}

	case "E":
		m.showInput = true
		m.input = inputState{
			mode:   inputPreferredEditor,
			prompt: "Editor command (e.g., nano, vim, code): ",
			value:  m.preferredEditor,
		}

	// Up/Down navigation based on active pane
	case "j", "down":
		if m.repeaterActive {
			m.repeaterFocus = (m.repeaterFocus + 1) % 5
			m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			return m, nil
		}
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
		if m.repeaterActive {
			m.repeaterFocus = (m.repeaterFocus + 4) % 5
			m.setStatus(repeaterFocusLabel(m.repeaterFocus))
			return m, nil
		}
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
		if m.repeaterActive {
			switch m.repeaterFocus {
			case 0:
				m.showInput = true
				m.input = inputState{mode: inputRepeaterMethod, prompt: "Method: ", value: m.repeaterMethod}
			case 1:
				m.showInput = true
				m.input = inputState{mode: inputRepeaterURL, prompt: "URL: ", value: m.repeaterURL}
			case 2:
				return m, m.openExternalEditor(inputRepeaterHeaders, m.repeaterHeaders)
			case 3:
				return m, m.openExternalEditor(inputRepeaterBody, m.repeaterBody)
			case 4:
				if !m.repeaterLoading {
					m.repeaterLoading = true
					m.repeaterResp = nil
					m.repeaterErr = ""
					return m, m.sendRepeaterRequest(m.repeaterMethod, m.repeaterURL, m.repeaterHeaders, m.repeaterBody)
				}
			}
			return m, nil
		}
		switch m.activePane {
		case paneSidebar:
			if m.sidebarCursor == 0 {
				m.currentGroup = nil
				m.noteList = nil
				m.noteCursor = 0
				visible := m.getVisibleEvents()
				if len(visible) > 0 {
					m.cursor = visible[0].idx
				} else {
					m.cursor = -1
				}
				m.activePane = paneList
				m.setStatus("All Requests selected")
				m.refreshDetail()
			} else if m.sidebarCursor > 0 && m.sidebarCursor <= len(m.groupList) {
				m.currentGroup = m.groupList[m.sidebarCursor-1]
				m.noteCursor = 0
				visible := m.getVisibleEvents()
				if len(visible) > 0 {
					m.cursor = visible[0].idx
				} else {
					m.cursor = -1
				}
				m.activePane = paneList
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
			err := m.noteRepo.Delete(context.Background(), n.ID)
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
	case "n":
		if m.currentSession == nil {
			return m, m.makeDefaultSession()
		}
		m.showInput = true
		m.input = inputState{
			mode:   inputNewSession,
			prompt: "Session name: ",
			value:  "",
		}

	case "s":
		if len(m.sessionList) == 0 {
			m.setStatus("No sessions (create one with n)")
			return m, nil
		}
		m.curModal = modalSessions
		items := make([]string, len(m.sessionList))
		for i, s := range m.sessionList {
			items[i] = s.Name
		}
		m.pick = picker{title: "Select Session", items: items, cursor: 0}

	case "c":
		if m.currentSession == nil {
			m.setStatus("No session (press n)")
			return m, nil
		}
		m.showInput = true
		m.input = inputState{
			mode:   inputNewGroup,
			prompt: "Group name: ",
			value:  "",
		}

	case "g":
		if len(m.groupList) > 0 {
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

		groupCounts := make(map[string]int)
		for _, ev := range m.events {
			if ev.req.GroupID != "" {
				groupCounts[ev.req.GroupID]++
			}
		}

		var items []string
		for _, g := range m.groupList {
			shortID := g.ID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			count := groupCounts[g.ID]
			displayName := g.Name
			if displayName == "" {
				displayName = fmt.Sprintf("Group %d", len(items)+1)
			}
			items = append(items, fmt.Sprintf("%s (%s) [%d req]", displayName, shortID, count))
		}
		items = append(items, "Unassign from group")
		items = append(items, "Create New Group and Assign")
		m.curModal = modalAssignGroup
		m.pick = picker{title: "Assign to Group", items: items, cursor: 0}

	case "A":
		if m.cursor < 0 || m.cursor >= len(m.events) {
			m.setStatus("No request selected")
			return m, nil
		}
		if m.currentSession == nil {
			m.setStatus("No active session")
			return m, nil
		}

		reqID := m.events[m.cursor].req.ID
		if m.currentGroup != nil && m.events[m.cursor].req.GroupID == m.currentGroup.ID {
			if err := m.reqRepo.UpdateGroup(context.Background(), reqID, ""); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = ""
				m.refreshDetail()
				m.setStatus("Unassigned from group")
			}
		} else if m.currentGroup != nil {
			if err := m.reqRepo.UpdateGroup(context.Background(), reqID, m.currentGroup.ID); err != nil {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			} else {
				m.events[m.cursor].req.GroupID = m.currentGroup.ID
				m.refreshDetail()
				m.setStatus("Assigned to " + m.currentGroup.Name)
			}
		} else {
			m.setStatus("No active group — use a/Sidebar/g to select one")
		}

	case "N":
		if m.currentGroup == nil {
			m.setStatus("Select a group first (sidebar or press g)")
			return m, nil
		}
		m.showInput = true
		m.input = inputState{
			mode:   inputAddNoteTitle,
			prompt: "Note title: ",
			value:  "",
		}

	case "/":
		m.showInput = true
		m.input = inputState{
			mode:   inputFilter,
			prompt: "Filter (regex not supported, plain text): ",
			value:  m.filter,
		}

	case "r":
		m.barrier.SetActive(true)
		m.setStatus("Drop mode — requests will be dropped")
	case "R":
		m.barrier.SetActive(true)
		m.setStatus("Forward mode — requests will be forwarded")

	case "ctrl+s":
		if m.repeaterActive && !m.repeaterLoading {
			m.repeaterLoading = true
			m.repeaterResp = nil
			m.repeaterErr = ""
			return m, m.sendRepeaterRequest(m.repeaterMethod, m.repeaterURL, m.repeaterHeaders, m.repeaterBody)
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
				g := &model.Group{
					ID:        m.idGen.New(),
					SessionID: m.currentSession.ID,
					Name:      gName,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				if err := m.groupRepo.Create(context.Background(), g); err != nil {
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
		m.noteCursor = 0
		visible := m.getVisibleEvents()
		if len(visible) > 0 {
			m.cursor = visible[0].idx
		} else {
			m.cursor = -1
		}
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
