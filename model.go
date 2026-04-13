package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Screen int

const (
	ScreenInput Screen = iota
	ScreenDownload
	ScreenHistory
)

type Model struct {
	screen     Screen
	config     Config
	queue      *Queue
	downloader *Downloader
	runtime    string

	isRunning      bool
	resolvingCount int

	progressCh chan tea.Msg

	urlInput        textinput.Model
	currentProgress progress.Model
	overallProgress progress.Model

	width  int
	height int

	// Navigation state
	queueCursor   int
	historyCursor int
	queueOffset   int
	historyOffset int
	expanded      map[string]bool // key: formatted BatchID
}

func initialModel(runtime string) Model {
	config, _ := loadConfig()

	ti := textinput.New()
	ti.Placeholder = "Enter YouTube URL or playlist..."
	ti.Focus()
	ti.CharLimit = 500
	ti.SetWidth(80) // fixes the placeholder not showing entirely

	prog := progress.New()
	prog.SetWidth(80)

	return Model{
		screen:          ScreenInput,
		config:          config,
		queue:           NewQueue(),
		downloader:      NewDownloader(config, runtime),
		runtime:         runtime,
		progressCh:      make(chan tea.Msg, 64),
		urlInput:        ti,
		currentProgress: prog,
		overallProgress: prog,
		expanded:        make(map[string]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q", "q":
			return m, tea.Quit
		case "tab":
			m.screen = (m.screen + 1) % 3
			return m, nil
		case "shift+tab":
			if m.screen == 0 {
				m.screen = 2
			} else {
				m.screen--
			}
			return m, nil
		case "ctrl+d":
			return m.tryStartDownloads()
		case "ctrl+r":
			m.queue.Entries = nil // Clear all
			m.queue.nextId = 1
			m.queueCursor = 0
			m.historyCursor = 0
			return m, nil
		}

		if m.screen == ScreenInput {
			// Handle Input Screen Logic
			if m.urlInput.Focused() {
				switch msg.String() {
				case "enter":
					url := strings.TrimSpace(m.urlInput.Value())
					if url == "" {
						return m, nil
					}
					m.urlInput.SetValue("")
					m.resolvingCount++
					return m, m.downloader.ResolvePlaylist(url, EntryConfig{})
				case "down":
					// Focus list if items exist
					if len(m.queue.GetQueued()) > 0 {
						m.urlInput.Blur()
						m.queueCursor = 0
					}
					return m, nil
				default:
					m.urlInput, cmd = m.urlInput.Update(msg)
					cmds = append(cmds, cmd)
					return m, tea.Batch(cmds...)
				}
			} else {
				// Handle List Navigation
				rows := calculateListRows(m.queue.GetQueued(), m.expanded)
				maxIdx := len(rows) - 1
				viewportHeight := m.height - 15
				if viewportHeight < 5 {
					viewportHeight = 5
				}

				switch msg.String() {
				case "up":
					if m.queueCursor > 0 {
						m.queueCursor--
						if m.queueCursor < m.queueOffset {
							m.queueOffset = m.queueCursor
						}
					} else {
						m.urlInput.Focus()
					}
				case "down":
					if m.queueCursor < maxIdx {
						m.queueCursor++
						if m.queueCursor >= m.queueOffset+viewportHeight {
							m.queueOffset = m.queueCursor - viewportHeight + 1
						}
					}
				case "enter", "space":
					if len(rows) > 0 {
						row := rows[m.queueCursor]
						if row.IsHeader {
							if m.expanded[row.BatchID] {
								delete(m.expanded, row.BatchID)
							} else {
								m.expanded[row.BatchID] = true
							}
						} else {
							// For items, space/enter might mean "Start"
							if msg.String() == "space" && !m.isRunning {
								return m.tryStartDownloads()
							}
						}
					}
				case "delete", "d", "backspace":
					if len(rows) > 0 {
						row := rows[m.queueCursor]
						if row.IsHeader {
							// Remove all in batch. Needs helper in Queue potentially but raw loop works.
							var newEntries []DownloadEntry
							for _, e := range m.queue.Entries {
								if e.Playlist == nil || e.Playlist.BatchID != row.BatchID {
									newEntries = append(newEntries, e)
								}
							}
							m.queue.Entries = newEntries
						} else {
							m.queue.Remove(row.EntryID)
						}
						// Adjust cursor if out of bounds
						newLen := len(calculateListRows(m.queue.GetQueued(), m.expanded))
						if m.queueCursor >= newLen && m.queueCursor > 0 {
							m.queueCursor = newLen - 1
						}
					}
				case "s":
					// Start specifically this item?
					// For now global start unless specific logic added.
					return m.tryStartDownloads()
				default:
					// Type to focus input
					if len(msg.String()) == 1 {
						m.urlInput.Focus()
						m.urlInput, cmd = m.urlInput.Update(msg)
						cmds = append(cmds, cmd)
					}
				}
				return m, tea.Batch(cmds...)
			}
		} else if m.screen == ScreenHistory {
			// Handle History Logic
			rows := calculateListRows(m.queue.GetCompleted(), m.expanded)
			maxIdx := len(rows) - 1
			viewportHeight := m.height - 10
			if viewportHeight < 5 {
				viewportHeight = 5
			}

			switch msg.String() {
			case "up":
				if m.historyCursor > 0 {
					m.historyCursor--
					if m.historyCursor < m.historyOffset {
						m.historyOffset = m.historyCursor
					}
				}
			case "down":
				if m.historyCursor < maxIdx {
					m.historyCursor++
					if m.historyCursor >= m.historyOffset+viewportHeight {
						m.historyOffset = m.historyCursor - viewportHeight + 1
					}
				}
			case "enter", "space":
				if len(rows) > 0 {
					row := rows[m.historyCursor]
					if row.IsHeader {
						if m.expanded[row.BatchID] {
							delete(m.expanded, row.BatchID)
						} else {
							m.expanded[row.BatchID] = true
						}
					}
				}
			case "delete", "d", "backspace":
				if len(rows) > 0 {
					row := rows[m.historyCursor]
					if row.IsHeader {
						var newEntries []DownloadEntry
						for _, e := range m.queue.Entries {
							if e.Playlist == nil || e.Playlist.BatchID != row.BatchID {
								newEntries = append(newEntries, e)
							}
						}
						m.queue.Entries = newEntries
					} else {
						m.queue.Remove(row.EntryID)
					}
					newLen := len(calculateListRows(m.queue.GetCompleted(), m.expanded))
					if m.historyCursor >= newLen && m.historyCursor > 0 {
						m.historyCursor = newLen - 1
					}
				}
			}
			return m, nil
		}

	// ── Mouse clicks ─────────────────────────────────────────────────────────
	case tea.MouseReleaseMsg:
		if msg.Button != ansi.MouseLeft {
			break
		}

		// Tab clicks
		switch {
		case zone.Get(zoneTabInput).InBounds(msg):
			m.screen = ScreenInput
			return m, nil
		case zone.Get(zoneTabDownload).InBounds(msg):
			m.screen = ScreenDownload
			return m, nil
		case zone.Get(zoneTabHistory).InBounds(msg):
			m.screen = ScreenHistory
			return m, nil
		}

		// Action buttons (retained for now if mouse user clicks old areas, but UI removed)
		if zone.Get(zoneStartBtn).InBounds(msg) {
			return m.tryStartDownloads()
		}
		if zone.Get(zoneRemoveBtn).InBounds(msg) {
			return m.tryRemoveLast()
		}

		// Per-entry and Playlist buttons
		// We loop over VISIBLE items to check for clicks.
		// However, iterating all entries is safer/easier if list isn't huge.
		// Since buttons are dynamic per ID/BatchID, we check them.

		// Checking Queued items
		for _, entry := range m.queue.GetQueued() {
			if zone.Get(zoneRemoveEntry(entry.ID)).InBounds(msg) {
				m.queue.Remove(entry.ID)
				return m, nil
			}
			if zone.Get(zoneStartEntry(entry.ID)).InBounds(msg) {
				// Start implies global start unless we implement specific item starting logic.
				// For now global start.
				return m.tryStartDownloads()
			}

			if entry.Playlist != nil {
				batchID := entry.Playlist.BatchID
				if zone.Get(zoneTogglePlaylist(batchID)).InBounds(msg) {
					if m.expanded[batchID] {
						delete(m.expanded, batchID)
					} else {
						m.expanded[batchID] = true
					}
					return m, nil
				}
				if zone.Get(zoneRemovePlaylist(batchID)).InBounds(msg) {
					// Remove entire batch
					var newEntries []DownloadEntry
					for _, e := range m.queue.Entries {
						if e.Playlist == nil || e.Playlist.BatchID != batchID {
							newEntries = append(newEntries, e)
						}
					}
					m.queue.Entries = newEntries
					return m, nil
				}
				if zone.Get(zoneStartPlaylist(batchID)).InBounds(msg) {
					return m.tryStartDownloads()
				}
			}
		}

		// Handle selection clicks for Input screen
		if m.screen == ScreenInput {
			rows := calculateListRows(m.queue.GetQueued(), m.expanded)
			// Iterate through rows to find what was clicked to set cursor
			// Only check visible ones or all? All is safer but slower?
			// Rows is fast enough.
			for i, row := range rows {
				clicked := false
				if row.IsHeader {
					if zone.Get(zoneSelectPlaylist(row.BatchID)).InBounds(msg) {
						clicked = true
						// Toggle logical? User said "override cursor index". Maybe select resets toggle?
						// Let's just select.
					}
				} else {
					if zone.Get(zoneSelectEntry(row.Entry.ID)).InBounds(msg) {
						clicked = true
					}
				}

				if clicked {
					m.queueCursor = i
					// Ensure visible?
					// Wait, if I clicked it, it must be visible.
					// But queueOffset handles visibility.
					// If I clicked it, it is on screen. So offset is probably fine.
					return m, nil
				}
			}
		}

		// Check History items (Completed)
		for _, entry := range m.queue.GetCompleted() {
			if zone.Get(zoneRemoveEntry(entry.ID)).InBounds(msg) {
				m.queue.Remove(entry.ID)
				return m, nil
			}
			// History doesn't have start buttons, only remove/toggle
			if entry.Playlist != nil {
				batchID := entry.Playlist.BatchID
				if zone.Get(zoneTogglePlaylist(batchID)).InBounds(msg) {
					if m.expanded[batchID] {
						delete(m.expanded, batchID)
					} else {
						m.expanded[batchID] = true
					}
					return m, nil
				}
			}
		}

		// Handle selection clicks for History screen
		if m.screen == ScreenHistory {
			rows := calculateListRows(m.queue.GetCompleted(), m.expanded)
			for i, row := range rows {
				clicked := false
				if row.IsHeader {
					if zone.Get(zoneSelectPlaylist(row.BatchID)).InBounds(msg) {
						clicked = true
					}
				} else {
					if zone.Get(zoneSelectEntry(row.Entry.ID)).InBounds(msg) {
						clicked = true
					}
				}

				if clicked {
					m.historyCursor = i
					return m, nil
				}
			}
		}

	case tea.MouseWheelMsg:
		var rows []ListRow
		var viewportHeight int
		var currentCursor *int
		var currentOffset *int

		if m.screen == ScreenInput {
			if m.urlInput.Focused() {
				if msg.Button == tea.MouseWheelDown && len(m.queue.GetQueued()) > 0 {
					m.urlInput.Blur()
					m.queueCursor = 0
					m.queueOffset = 0
				}
				return m, nil
			}

			rows = calculateListRows(m.queue.GetQueued(), m.expanded)
			viewportHeight = m.height - 15
			currentCursor = &m.queueCursor
			currentOffset = &m.queueOffset
		} else if m.screen == ScreenHistory {
			rows = calculateListRows(m.queue.GetCompleted(), m.expanded)
			viewportHeight = m.height - 10
			currentCursor = &m.historyCursor
			currentOffset = &m.historyOffset
		}

		if currentCursor != nil {
			if viewportHeight < 5 {
				viewportHeight = 5
			}

			// Determine direction
			delta := 0
			if msg.Button == tea.MouseWheelUp {
				delta = -1
			} else if msg.Button == tea.MouseWheelDown {
				delta = 1
			}

			newCursor := *currentCursor + delta

			if m.screen == ScreenInput && newCursor < 0 {
				m.urlInput.Focus()
				return m, nil
			}

			maxCursor := len(rows) - 1
			if maxCursor < 0 {
				maxCursor = 0
			}

			if newCursor < 0 {
				newCursor = 0
			} else if newCursor > maxCursor {
				newCursor = maxCursor
			}

			*currentCursor = newCursor

			// Keep cursor in view (adjust offset)
			// If cursor is above offset, move offset up
			if *currentCursor < *currentOffset {
				*currentOffset = *currentCursor
			}
			// If cursor is below viewport, move offset down
			if *currentCursor >= *currentOffset+viewportHeight {
				*currentOffset = *currentCursor - viewportHeight + 1
			}
		}

	// ── Domain messages ───────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Update progress bar widths
		targetWidth := m.width - 20
		if targetWidth < 20 {
			targetWidth = 20
		}
		m.currentProgress.SetWidth(targetWidth)
		m.overallProgress.SetWidth(targetWidth)

		return m, nil

	case PlaylistResolvedMsg:
		m.resolvingCount--
		if msg.Error != nil {
			m.queue.Add(msg.OriginalURL, msg.Config)
			id := m.queue.Entries[len(m.queue.Entries)-1].ID
			m.queue.Update(id, func(e *DownloadEntry) {
				e.Status = StatusFailed
				e.Error = fmt.Sprintf("playlist resolve error: %v", msg.Error)
			})
			return m, nil
		}
		if msg.PlaylistTitle != "" {
			m.queue.AddPlaylistItems(msg.Items, msg.PlaylistTitle, msg.Config)
		} else if len(msg.Items) > 0 {
			item := msg.Items[0]
			m.queue.Add(item.URL, msg.Config)
			if item.Title != "" {
				id := m.queue.Entries[len(m.queue.Entries)-1].ID
				m.queue.Update(id, func(e *DownloadEntry) { e.Title = item.Title })
			}
		}
		return m, nil

	case ProgressMsg:
		m.queue.Update(msg.ID, func(e *DownloadEntry) {
			e.Progress = msg.Progress
			if msg.Title != "" {
				e.Title = msg.Title
			}
		})
		return m, listenProgress(m.progressCh)

	case DownloadCompleteMsg:
		m.queue.Update(msg.ID, func(e *DownloadEntry) {
			if msg.Error != nil {
				e.Status = StatusFailed
				e.Error = msg.Error.Error()
			} else {
				e.Status = StatusCompleted
				e.OutputPath = msg.OutputPath
			}
		})
		if m.isRunning {
			return m, m.startNextDownload()
		}
		return m, nil
	}

	if m.screen == ScreenInput {
		// On Windows, pastes often contain \r or \n which can crash the renderer
		// if textinput isn't expecting them (e.g. single-line mode).
		if p, ok := msg.(tea.PasteMsg); ok {
			clean := strings.ReplaceAll(p.Content, "\r", "")
			clean = strings.ReplaceAll(clean, "\n", "")
			msg = tea.PasteMsg{Content: clean}
		}

		m.urlInput, cmd = m.urlInput.Update(msg)

		// Fallback sanitization for non-paste inputs (e.g. weird key combos)
		v := m.urlInput.Value()
		if strings.ContainsAny(v, "\r\n") {
			v = strings.ReplaceAll(v, "\r", "")
			v = strings.ReplaceAll(v, "\n", "")
			m.urlInput.SetValue(v)
		}

		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.View{Content: "Loading..."}
	}

	var s strings.Builder
	s.WriteString(m.renderTabs())
	s.WriteString("\n\n")

	switch m.screen {
	case ScreenInput:
		s.WriteString(m.renderInputScreen())
	case ScreenDownload:
		s.WriteString(m.renderDownloadScreen())
	case ScreenHistory:
		s.WriteString(m.renderHistoryScreen())
	}

	s.WriteString("\n\n")
	s.WriteString(m.renderFooter())

	// zone.Scan must wrap the entire final output at the root model level.
	content := zone.Scan(s.String())

	return tea.View{
		Content:   content,
		AltScreen: true,
		MouseMode: tea.MouseModeCellMotion,
	}
}
