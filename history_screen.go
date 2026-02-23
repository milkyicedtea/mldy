package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderHistoryScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	faintStyle := lipgloss.NewStyle().Faint(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	playlistStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)

	s.WriteString(titleStyle.Render("Download History"))
	s.WriteString("\n\n")

	completed := m.queue.GetCompleted()
	if len(completed) == 0 {
		s.WriteString(faintStyle.Render("No completed downloads"))
		return s.String()
	}

	lastPlaylist := ""
	for _, entry := range completed {
		if entry.Playlist != nil && entry.Playlist.PlaylistTitle != lastPlaylist {
			lastPlaylist = entry.Playlist.PlaylistTitle
			s.WriteString(playlistStyle.Render("▶ "+lastPlaylist) + "\n")
		} else if entry.Playlist == nil {
			lastPlaylist = ""
		}

		indent := ""
		if entry.Playlist != nil {
			indent = "  "
		}

		icon := successStyle.Render("✓")
		if entry.Status == StatusFailed {
			icon = failStyle.Render("✗")
		}
		s.WriteString(fmt.Sprintf("%s%s %s\n", indent, icon, entry.DisplayTitle()))

		if entry.Status == StatusFailed && entry.Error != "" {
			for _, line := range strings.Split(entry.Error, "\n") {
				s.WriteString(indent + "  " + errorStyle.Render(line) + "\n")
			}
		} else if entry.OutputPath != "" {
			s.WriteString(fmt.Sprintf("%s  Saved to: %s\n", indent, entry.OutputPath))
		}
		s.WriteString("\n")
	}

	return s.String()
}
