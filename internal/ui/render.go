package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

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

		sidebarView := sStyle.Width(sidebarW - 2).Height(mh - 2).Render(m.renderSidebar(sidebarW-2, mh-2))
		listView := lStyle.Width(listW - 2).Height(mh - 2).Render(m.renderList(listW-2, mh-2))
		detailView := dStyle.Width(detailW - 2).Height(mh - 2).Render(m.renderDetail(detailW-2, mh-2))

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

func (m Model) renderRepeater(w, h int) string {
	colW := w / 2
	if colW < 20 {
		colW = 20
	}
	colH := h - 2

	var leftBuf strings.Builder
	leftBuf.WriteString(detailSection.Render(" REQUEST EDITOR ") + "\n\n")

	methodStyle := lipgloss.NewStyle()
	if m.repeaterFocus == 0 {
		methodStyle = methodStyle.Background(accentColor).Foreground(lipgloss.Color("#FAFAFA"))
	}
	leftBuf.WriteString("Method (Press Enter to edit):\n" + methodStyle.Render(" "+m.repeaterMethod+" ") + "\n\n")

	urlStyle := lipgloss.NewStyle()
	if m.repeaterFocus == 1 {
		urlStyle = urlStyle.Background(accentColor).Foreground(lipgloss.Color("#FAFAFA"))
	}
	leftBuf.WriteString("URL (Press Enter to edit):\n" + urlStyle.Render(" "+m.repeaterURL+" ") + "\n\n")

	headersStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#303030"))
	if m.repeaterFocus == 2 {
		headersStyle = headersStyle.BorderForeground(accentColor)
	}
	hdrVal := m.repeaterHeaders
	if hdrVal == "" {
		hdrVal = "(Empty Headers)"
	}
	leftBuf.WriteString("Headers (Press Enter to edit):\n" + headersStyle.Width(colW-4).Render(hdrVal) + "\n\n")

	bodyStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#303030"))
	if m.repeaterFocus == 3 {
		bodyStyle = bodyStyle.BorderForeground(accentColor)
	}
	bodyVal := m.repeaterBody
	if bodyVal == "" {
		bodyVal = "(Empty Body)"
	}
	leftBuf.WriteString("Body (Press Enter to edit):\n" + bodyStyle.Width(colW-4).Render(bodyVal) + "\n\n")

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
	leftBuf.WriteString("\n\n" + dim.Render(" ℹ Navigate: [↑↓/Tab] • Edit: [Enter] • Send: [Ctrl+S] • Exit: [Esc/q]"))

	var rightBuf strings.Builder
	rightBuf.WriteString(detailSection.Render(" RESPONSE VIEW ") + "\n\n")

	if m.repeaterResp != nil {
		resp := m.repeaterResp
		rightBuf.WriteString("Status:\n" + statusColor(resp.StatusCode).Render(fmt.Sprintf(" %d %s ", resp.StatusCode, resp.Status)) + "\n\n")

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
