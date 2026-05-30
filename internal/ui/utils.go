package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

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

func repeaterFocusLabel(focus int) string {
	switch focus {
	case 0:
		return "Method"
	case 1:
		return "URL"
	case 2:
		return "Headers"
	case 3:
		return "Body"
	case 4:
		return "Send"
	default:
		return ""
	}
}
