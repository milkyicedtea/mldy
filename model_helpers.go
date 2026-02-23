package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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

func listenProgress(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m *Model) startNextDownload() tea.Cmd {
	queued := m.queue.GetQueued()
	if len(queued) == 0 {
		m.isRunning = false
		return nil
	}
	entry := queued[0]
	m.queue.Update(entry.ID, func(e *DownloadEntry) { e.Status = StatusDownloading })
	return tea.Batch(
		m.downloader.StartDownload(m.queue.GetByID(entry.ID), m.progressCh),
		listenProgress(m.progressCh),
	)
}

func (m *Model) tryStartDownloads() (tea.Model, tea.Cmd) {
	if !m.isRunning && m.resolvingCount == 0 && len(m.queue.GetQueued()) > 0 {
		m.isRunning = true
		return m, m.startNextDownload()
	}
	return m, nil
}

func (m *Model) tryRemoveLast() (tea.Model, tea.Cmd) {
	queued := m.queue.GetQueued()
	if len(queued) > 0 {
		m.queue.Remove(queued[len(queued)-1].ID)
	}
	return m, nil
}
