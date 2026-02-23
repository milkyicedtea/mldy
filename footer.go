package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helps := []string{"tab/click: switch screen"}

	switch m.screen {
	case ScreenInput:
		helps = append(helps, "enter: add URL")
		if m.resolvingCount > 0 {
			helps = append(helps, "resolving...")
		} else if m.isRunning {
			helps = append(helps, "d: downloading...")
		} else if len(m.queue.GetQueued()) > 0 {
			helps = append(helps, "d: start  •  backspace: remove last")
		}
	case ScreenDownload:
		if m.isRunning {
			helps = append(helps, "downloading...")
		} else if len(m.queue.GetQueued()) > 0 {
			helps = append(helps, "d: start downloads")
		}
	case ScreenHistory:
		if len(m.queue.GetCompleted()) == 0 {
			helps = append(helps, "no history yet")
		}
	}

	helps = append(helps, "q: quit")
	return helpStyle.Render(strings.Join(helps, " • "))
}
