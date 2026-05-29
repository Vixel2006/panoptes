package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

func Run(cfg Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := NewModel(cfg)
	p := tea.NewProgram(m)

	go func() {
		for {
			select {
			case req, ok := <-cfg.RequestCh:
				if !ok {
					return
				}
				p.Send(eventReceived{req: req})
			case <-ctx.Done():
				return
			}
		}
	}()

	_, err := p.Run()
	return err
}
