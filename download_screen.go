package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderDownloadScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	faintStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Bold(true)

	s.WriteString(titleStyle.Render("Active Downloads"))
	s.WriteString("\n\n")

	active := m.queue.GetActive()
	if len(active) == 0 {
		s.WriteString(faintStyle.Render("No active downloads"))
	} else {
		for _, entry := range active {
			label := entry.DisplayTitle()
			if entry.Playlist != nil {
				label = fmt.Sprintf("[%s %d/%d] %s",
					entry.Playlist.PlaylistTitle,
					entry.Playlist.Index,
					entry.Playlist.Total,
					label,
				)
			}
			s.WriteString(fmt.Sprintf("Downloading: %s\n", label))
			s.WriteString(m.currentProgress.ViewAs(entry.Progress / 100.0))
			s.WriteString(fmt.Sprintf(" %.1f%%\n\n", entry.Progress))
		}
	}

	s.WriteString("\n")
	s.WriteString(boldStyle.Render("Overall Progress:"))
	s.WriteString("\n")
	totalProg := m.queue.TotalProgress()
	s.WriteString(m.overallProgress.ViewAs(totalProg / 100.0))
	s.WriteString(fmt.Sprintf(" %.1f%%", totalProg))

	completed := len(m.queue.GetCompleted())
	total := len(m.queue.Entries)
	s.WriteString(fmt.Sprintf("\n\nCompleted: %d/%d", completed, total))

	return s.String()
}
