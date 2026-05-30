package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Vixel2006/panoptes/internal/core/models"
)

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
		decompressed, _ := m.decompressor.Decompress(resp.Header.Get("Content-Encoding"), respBody)

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
