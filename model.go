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
}

func initialModel(runtime string) Model {
	config, _ := loadConfig()

	ti := textinput.New()
	ti.Placeholder = "Enter YouTube URL or playlist..."
	ti.Focus()
	ti.CharLimit = 500
	ti.SetWidth(80)

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
		case "ctrl+c", "q":
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
		case "enter":
			if m.screen == ScreenInput {
				url := strings.TrimSpace(m.urlInput.Value())
				if url == "" {
					return m, nil
				}
				m.urlInput.SetValue("")
				m.resolvingCount++
				return m, m.downloader.ResolvePlaylist(url, EntryConfig{})
			}
		case "ctrl+d":
			return m.tryStartDownloads()
		case "backspace", "delete":
			if m.screen == ScreenInput && m.urlInput.Value() == "" {
				return m.tryRemoveLast()
			}
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

		// Action buttons
		if zone.Get(zoneStartBtn).InBounds(msg) {
			return m.tryStartDownloads()
		}
		if zone.Get(zoneRemoveBtn).InBounds(msg) {
			return m.tryRemoveLast()
		}

		// Per-entry ✕ buttons
		for _, entry := range m.queue.GetQueued() {
			if zone.Get(zoneRemoveEntry(entry.ID)).InBounds(msg) {
				m.queue.Remove(entry.ID)
				return m, nil
			}
		}

	case tea.MouseWheelMsg:
		// Optional: handle scroll if needed

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
