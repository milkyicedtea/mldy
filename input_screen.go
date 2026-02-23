package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	zone "github.com/lrstanley/bubblezone"
)

func (m Model) renderInputScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	faintStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Bold(true)

	s.WriteString(titleStyle.Render("Add URLs to Queue"))
	s.WriteString("\n\n")
	s.WriteString(m.urlInput.View())
	s.WriteString("\n\n")

	if m.resolvingCount > 0 {
		s.WriteString(faintStyle.Render(fmt.Sprintf("⟳ Resolving %d URL(s)...", m.resolvingCount)))
		s.WriteString("\n\n")
	}

	queued := m.queue.GetQueued()
	if len(queued) > 0 {
		s.WriteString(boldStyle.Render(fmt.Sprintf("Queued (%d):", len(queued))))
		s.WriteString("\n")

		playlistStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
		removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Faint(true)
		lastPlaylist := ""

		for i, entry := range queued {
			if entry.Playlist != nil && entry.Playlist.PlaylistTitle != lastPlaylist {
				lastPlaylist = entry.Playlist.PlaylistTitle
				s.WriteString(fmt.Sprintf("  %s\n", playlistStyle.Render("▶ "+lastPlaylist)))
			} else if entry.Playlist == nil {
				lastPlaylist = ""
			}

			indent := "  "
			if entry.Playlist != nil {
				indent = "    "
			}

			label := entry.DisplayTitle()
			if entry.Playlist != nil {
				label = fmt.Sprintf("%d/%d  %s", entry.Playlist.Index, entry.Playlist.Total, label)
			}

			// ✕ button, individually zoned per entry ID.
			removeBtn := zone.Mark(zoneRemoveEntry(entry.ID), removeStyle.Render(" ✕"))
			s.WriteString(fmt.Sprintf("%s%d. %s%s\n", indent, i+1, label, removeBtn))
		}

		s.WriteString("\n")

		// "Remove last" and "Start downloads" action buttons.
		removeBtnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
		startBtnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
		disabledBtnStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

		removeBtn := zone.Mark(zoneRemoveBtn, removeBtnStyle.Render("✕ Remove last"))

		canStart := !m.isRunning && m.resolvingCount == 0
		var startBtn string
		if canStart {
			startBtn = zone.Mark(zoneStartBtn, startBtnStyle.Render("▶ Start downloads"))
		} else if m.isRunning {
			startBtn = disabledBtnStyle.Render("⟳ Downloading...")
		} else {
			startBtn = disabledBtnStyle.Render("▶ Start downloads")
		}

		s.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, "  ", removeBtn, "  ", startBtn))
		s.WriteString("\n")
	} else if m.resolvingCount == 0 {
		s.WriteString(faintStyle.Render("No items in queue"))
	}

	s.WriteString("\n\n")
	s.WriteString(boldStyle.Render("Current Config:"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  Kind:          %s\n", m.config.Kind))
	s.WriteString(fmt.Sprintf("  Format:        %s\n", m.config.Format))
	s.WriteString(fmt.Sprintf("  Audio Quality: %s\n", m.config.AudioQuality))
	s.WriteString(fmt.Sprintf("  Video Quality: %s\n", m.config.VideoQuality))
	s.WriteString(fmt.Sprintf("  Output Folder: %s\n", m.config.OutputFolder))
	if m.runtime != "" {
		s.WriteString(fmt.Sprintf("  JS Runtime:    %s\n", m.runtime))
	} else {
		s.WriteString(faintStyle.Render("  JS Runtime:    none (some videos may fail)\n"))
	}

	return s.String()
}
