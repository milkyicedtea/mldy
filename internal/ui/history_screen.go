package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"mldy/internal/download"
)

func (m Model) renderHistoryScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))
	faintStyle := lipgloss.NewStyle().Faint(true)
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	playlistStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	s.WriteString(titleStyle.Render("Download History"))
	s.WriteString("\n\n")

	completed := m.queue.GetCompleted()
	if len(completed) == 0 {
		s.WriteString(faintStyle.Render("No completed downloads"))
		return s.String()
	}

	rows := calculateListRows(completed, m.expanded)

	// Viewport logic
	height := m.height - 10
	if height < 5 {
		height = 5
	}

	start := m.historyOffset
	end := start + height
	if end > len(rows) {
		end = len(rows)
	}
	if start > len(rows) {
		start = 0
	}

	for i := start; i < end; i++ {
		row := rows[i]
		isSelected := i == m.historyCursor

		prefix := "  "
		if isSelected {
			prefix = "> "
		}

		if row.IsHeader {
			expanded := m.expanded[row.BatchID]
			arrow := "▶"
			if expanded {
				arrow = "▼"
			}
			label := fmt.Sprintf("%s%s %s", prefix, arrow, row.PlaylistTitle)

			// Make label selectable (and toggleable ideally)
			label = zone.Mark(zoneSelectPlaylist(row.BatchID), label)

			if isSelected {
				s.WriteString(selectedStyle.Render(label))
				s.WriteString(faintStyle.Render(" [Enter: Expand/Collapse]"))
			} else {
				s.WriteString(playlistStyle.Render(label))
			}
			s.WriteString("\n")
		} else {
			indent := "  "
			if row.Entry.Playlist != nil {
				indent = "    "
			}

			icon := successStyle.Render("✓")
			if row.Entry.Status == download.StatusFailed {
				icon = failStyle.Render("✗")
			}

			title := row.Entry.DisplayTitle()
			line := fmt.Sprintf("%s%s%s %s", prefix, indent, icon, title)

			// Make line selectable
			line = zone.Mark(zoneSelectEntry(row.Entry.ID), line)

			if isSelected {
				s.WriteString(selectedStyle.Render(line))
			} else {
				s.WriteString(line)
			}

			// Clickable remove
			delBtn := zone.Mark(zoneRemoveEntry(row.Entry.ID), failStyle.Render(" ✕"))
			s.WriteString(" " + delBtn)

			s.WriteString("\n")

			// Error details (only if selected or always? Always is better but minimal space)
			// Or maybe only if expanded logic?
			// Let's keep existing logic: show error if failed.
			if row.Entry.Status == download.StatusFailed && row.Entry.Error != "" {
				// We render error lines below the item.
				// This complicates "Row" abstraction if error takes multiple lines.
				// For now, let's just indent it.
				// If we want navigable lines for error, we'd need them in rows.
				// Current implementation just prints it.
				// Caution: this might mess up the "Height" calculation if not accounted for.
				// But we are in a loop iterating rows.
				// Render it as part of this row's output.
				for _, line := range strings.Split(row.Entry.Error, "\n") {
					s.WriteString(indent + "  " + errorStyle.Render(line) + "\n")
				}
			} else if row.Entry.OutputPath != "" {
				s.WriteString(fmt.Sprintf("%s%s  Saved to: %s\n", prefix, indent, row.Entry.OutputPath))
			}
			// Add extra newline
			s.WriteString("\n")
		}
	}

	if len(rows) > end {
		s.WriteString(faintStyle.Render(fmt.Sprintf("... %d more ...", len(rows)-end)) + "\n")
	}

	return s.String()
}
