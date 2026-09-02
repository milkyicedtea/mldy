package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m Model) renderInputScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	faintStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	playlistStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	startStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	dimmedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	s.WriteString(titleStyle.Render("Add URLs to Queue"))
	s.WriteString("\n\n")
	s.WriteString(m.urlInput.View()) // Input field
	s.WriteString("\n\n")

	if m.resolvingCount > 0 {
		s.WriteString(faintStyle.Render(fmt.Sprintf("⟳ Resolving %d URL(s)...", m.resolvingCount)))
		s.WriteString("\n\n")
	}

	queued := m.queue.GetQueued()
	if len(queued) > 0 {
		s.WriteString(boldStyle.Render(fmt.Sprintf("Queued (%d):", len(queued))))
		s.WriteString("\n")

		rows := calculateListRows(queued, m.expanded)

		// Viewport logic
		height := m.height - 15 // Approx space for header/footer
		if height < 5 {
			height = 5
		}

		start := m.queueOffset
		end := start + height
		if end > len(rows) {
			end = len(rows)
		}
		if start > len(rows) {
			start = 0 // Reset if out of bounds
		}

		for i := start; i < end; i++ {
			row := rows[i]
			isSelected := !m.urlInput.Focused() && i == m.queueCursor

			prefix := "  "
			if isSelected {
				prefix = "> "
			}

			if row.IsHeader {
				expanded := m.expanded[row.BatchID]
				icon := "▶"
				if expanded {
					icon = "▼"
				}

				// Clickable toggle region
				toggleZone := zone.Mark(zoneSelectPlaylist(row.BatchID), icon+" "+row.PlaylistTitle)

				line := fmt.Sprintf("%s%s", prefix, toggleZone)

				if isSelected {
					s.WriteString(selectedStyle.Render(line))
				} else {
					s.WriteString(playlistStyle.Render(line))
				}

				// Always visible actions
				startBtn := zone.Mark(zoneStartPlaylist(row.BatchID), startStyle.Render(" ▶"))
				delBtn := zone.Mark(zoneRemovePlaylist(row.BatchID), removeStyle.Render(" ✕"))
				s.WriteString(" " + startBtn + delBtn)

				s.WriteString("\n")
			} else {
				// Indent items if they belong to a visible playlist (which they always do if filtered)
				// Actually if it's a single item, Indent is small. If part of playlist, larger.
				indent := "  "
				if row.Entry.Playlist != nil {
					indent = "    "
				}

				label := row.Entry.DisplayTitle()
				if row.Entry.Playlist != nil {
					label = fmt.Sprintf("%d/%d  %s", row.Entry.Playlist.Index, row.Entry.Playlist.Total, label)
				}

				line := fmt.Sprintf("%s%s%s", prefix, indent, label)

				// Wrap label region as clickable for selection
				line = zone.Mark(zoneSelectEntry(row.Entry.ID), line)

				if isSelected {
					s.WriteString(selectedStyle.Render(line))
				} else {
					s.WriteString(line)
				}

				// Always visible actions (small)
				startBtn := zone.Mark(zoneStartEntry(row.Entry.ID), startStyle.Render(" ▶"))
				delBtn := zone.Mark(zoneRemoveEntry(row.Entry.ID), removeStyle.Render(" ✕"))
				s.WriteString(" " + startBtn + delBtn)

				s.WriteString("\n")
			}
		}

		if len(rows) > end {
			s.WriteString(faintStyle.Render(fmt.Sprintf("... %d more ...", len(rows)-end)) + "\n")
		}
		s.WriteString("\n")

	} else if m.resolvingCount == 0 {
		s.WriteString(faintStyle.Render("No items in queue"))
		s.WriteString("\n\n")
	}

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
		s.WriteString(dimmedStyle.Render("  JS Runtime:    none (some videos may fail)\n"))
	}

	return s.String()
}
