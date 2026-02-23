package main

import (
	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

func (m Model) renderTabs() string {
	activeStyle := lipgloss.NewStyle().Bold(true).
		Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 2)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 2)

	tab := func(label, zoneID string, current Screen) string {
		var rendered string
		if m.screen == current {
			rendered = activeStyle.Render(label)
		} else {
			rendered = inactiveStyle.Render(label)
		}
		return zone.Mark(zoneID, rendered)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		tab("Input/Queue", zoneTabInput, ScreenInput),
		tab("Downloads", zoneTabDownload, ScreenDownload),
		tab("History", zoneTabHistory, ScreenHistory),
	)
}
