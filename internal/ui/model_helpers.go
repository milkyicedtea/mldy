package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"mldy/internal/download"
)

// Zone IDs for clickable regions.
const (
	zoneTabInput    = "tab-input"
	zoneTabDownload = "tab-download"
	zoneTabHistory  = "tab-history"
	zoneStartBtn    = "btn-start"
	zoneRemoveBtn   = "btn-remove-last"
	// Per-entry remove buttons use "btn-remove-<entry.ID>", built dynamically.
)

func zoneRemoveEntry(id int) string {
	return fmt.Sprintf("btn-remove-%d", id)
}

func zoneStartEntry(id int) string {
	return fmt.Sprintf("btn-start-%d", id)
}

func zoneTogglePlaylist(batchID string) string {
	return fmt.Sprintf("btn-toggle-%s", batchID)
}

func zoneStartPlaylist(batchID string) string {
	return fmt.Sprintf("btn-start-batch-%s", batchID)
}

func zoneRemovePlaylist(batchID string) string {
	return fmt.Sprintf("btn-remove-batch-%s", batchID)
}

func zoneSelectEntry(id int) string {
	return fmt.Sprintf("select-entry-%d", id)
}

func zoneSelectPlaylist(batchID string) string {
	return fmt.Sprintf("select-batch-%s", batchID)
}

func listenProgress(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m Model) startNextDownload() tea.Cmd {
	queued := m.queue.GetQueued()
	if len(queued) == 0 {
		m.isRunning = false
		return nil
	}
	entry := queued[0]
	m.queue.Update(entry.ID, func(e *download.DownloadEntry) { e.Status = download.StatusDownloading })
	return tea.Batch(
		m.downloader.StartDownload(m.queue.GetByID(entry.ID), m.progressCh),
		listenProgress(m.progressCh),
	)
}

func (m Model) tryStartDownloads() (tea.Model, tea.Cmd) {
	if !m.isRunning && m.resolvingCount == 0 && len(m.queue.GetQueued()) > 0 {
		m.isRunning = true
		return m, m.startNextDownload()
	}
	return m, nil
}

func (m Model) tryRemoveLast() (tea.Model, tea.Cmd) {
	queued := m.queue.GetQueued()
	if len(queued) > 0 {
		m.queue.Remove(queued[len(queued)-1].ID)
	}
	return m, nil
}

type ListRow struct {
	IsHeader      bool
	PlaylistTitle string
	BatchID       string
	EntryID       int                     // valid only if !IsHeader
	Entry         *download.DownloadEntry // valid only if !IsHeader
}

func calculateListRows(entries []download.DownloadEntry, expanded map[string]bool) []ListRow {
	var rows []ListRow
	lastBatchID := ""

	for i := range entries {
		entry := &entries[i]
		currentBatchID := ""
		if entry.Playlist != nil {
			currentBatchID = entry.Playlist.BatchID
		}

		// Handle playlist headers
		if currentBatchID != "" {
			if currentBatchID != lastBatchID {
				rows = append(rows, ListRow{
					IsHeader:      true,
					PlaylistTitle: entry.Playlist.PlaylistTitle,
					BatchID:       currentBatchID,
				})
				lastBatchID = currentBatchID
			}

			// If not expanded (default is collapsed), skip adding the entry
			if !expanded[currentBatchID] {
				continue
			}
		} else {
			lastBatchID = ""
		}

		rows = append(rows, ListRow{
			IsHeader: false,
			EntryID:  entry.ID,
			Entry:    entry,
		})
	}
	return rows
}
