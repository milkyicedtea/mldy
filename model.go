package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	runtime    string // JavaScript runtime (deno, bun, node, or empty)

	// Input screen
	urlInput textinput.Model

	// Download screen
	currentProgress progress.Model
	overallProgress progress.Model

	// UI
	width  int
	height int
	err    error
}

func initialModel(runtime string) Model {
	config, _ := loadConfig()

	ti := textinput.New()
	ti.Placeholder = "Enter YouTube URL..."
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 80

	currentProg := progress.New(progress.WithDefaultGradient())
	overallProg := progress.New(progress.WithDefaultGradient())

	return Model{
		screen:          ScreenInput,
		config:          config,
		queue:           NewQueue(),
		downloader:      NewDownloader(config, runtime),
		runtime:         runtime,
		urlInput:        ti,
		currentProgress: currentProg,
		overallProgress: overallProg,
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
			// Cycle through screens
			m.screen = (m.screen + 1) % 3
			return m, nil

		case "shift+tab":
			// Cycle backwards through screens
			if m.screen == 0 {
				m.screen = 2
			} else {
				m.screen--
			}
			return m, nil

		case "enter":
			if m.screen == ScreenInput && m.urlInput.Value() != "" {
				// Add URL to queue
				m.queue.Add(m.urlInput.Value(), EntryConfig{})
				m.urlInput.SetValue("")
				return m, nil
			}

		case "d":
			// Start downloading from queue
			if m.screen == ScreenInput || m.screen == ScreenDownload {
				queued := m.queue.GetQueued()
				if len(queued) > 0 {
					entry := queued[0]
					m.queue.Update(entry.ID, func(e *DownloadEntry) {
						e.Status = StatusDownloading
					})
					return m, m.downloader.StartDownload(m.queue.GetByID(entry.ID))
				}
			}

		case "backspace", "delete":
			if m.screen == ScreenInput {
				// Remove last queued item
				queued := m.queue.GetQueued()
				if len(queued) > 0 && m.urlInput.Value() == "" {
					m.queue.Remove(queued[len(queued)-1].ID)
					return m, nil
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case ProgressMsg:
		m.queue.Update(msg.ID, func(e *DownloadEntry) {
			e.Progress = msg.Progress
			if msg.Title != "" {
				e.Title = msg.Title
			}
		})
		return m, nil

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

		// Start next download if available
		queued := m.queue.GetQueued()
		if len(queued) > 0 {
			entry := queued[0]
			m.queue.Update(entry.ID, func(e *DownloadEntry) {
				e.Status = StatusDownloading
			})
			return m, m.downloader.StartDownload(m.queue.GetByID(entry.ID))
		}
		return m, nil
	}

	// Update input on input screen
	if m.screen == ScreenInput {
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var s strings.Builder

	// Header with tabs
	s.WriteString(m.renderTabs())
	s.WriteString("\n\n")

	// Screen content
	switch m.screen {
	case ScreenInput:
		s.WriteString(m.renderInputScreen())
	case ScreenDownload:
		s.WriteString(m.renderDownloadScreen())
	case ScreenHistory:
		s.WriteString(m.renderHistoryScreen())
	}

	// Footer
	s.WriteString("\n\n")
	s.WriteString(m.renderFooter())

	return s.String()
}

func (m Model) renderTabs() string {
	activeTab := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 2)

	inactiveTab := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	var tabs []string

	if m.screen == ScreenInput {
		tabs = append(tabs, activeTab.Render("Input/Queue"))
	} else {
		tabs = append(tabs, inactiveTab.Render("Input/Queue"))
	}

	if m.screen == ScreenDownload {
		tabs = append(tabs, activeTab.Render("Downloads"))
	} else {
		tabs = append(tabs, inactiveTab.Render("Downloads"))
	}

	if m.screen == ScreenHistory {
		tabs = append(tabs, activeTab.Render("History"))
	} else {
		tabs = append(tabs, inactiveTab.Render("History"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m Model) renderInputScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	s.WriteString(titleStyle.Render("Add URLs to Queue"))
	s.WriteString("\n\n")
	s.WriteString(m.urlInput.View())
	s.WriteString("\n\n")

	// Show queued items
	queued := m.queue.GetQueued()
	if len(queued) > 0 {
		s.WriteString(lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Queued (%d):", len(queued))))
		s.WriteString("\n")
		for i, entry := range queued {
			s.WriteString(fmt.Sprintf("  %d. %s\n", i+1, entry.URL))
		}
	} else {
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("No items in queue"))
	}

	// Show current config
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Current Config:"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  Format: %s\n", m.config.Format))
	s.WriteString(fmt.Sprintf("  Audio Quality: %s\n", m.config.AudioQuality))
	s.WriteString(fmt.Sprintf("  Video Quality: %s\n", m.config.VideoQuality))
	s.WriteString(fmt.Sprintf("  Output Folder: %s\n", m.config.OutputFolder))
	if m.runtime != "" {
		s.WriteString(fmt.Sprintf("  JS Runtime: %s\n", m.runtime))
	} else {
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("  JS Runtime: none (some videos may fail)\n"))
	}

	return s.String()
}

func (m Model) renderDownloadScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	s.WriteString(titleStyle.Render("Active Downloads"))
	s.WriteString("\n\n")

	active := m.queue.GetActive()
	if len(active) == 0 {
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("No active downloads"))
	} else {
		for _, entry := range active {
			s.WriteString(fmt.Sprintf("Downloading: %s\n", entry.URL))
			if entry.Title != "" {
				s.WriteString(fmt.Sprintf("Title: %s\n", entry.Title))
			}
			s.WriteString(m.currentProgress.ViewAs(entry.Progress / 100.0))
			s.WriteString(fmt.Sprintf(" %.1f%%\n\n", entry.Progress))
		}
	}

	// Overall progress
	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Bold(true).Render("Overall Progress:"))
	s.WriteString("\n")
	totalProg := m.queue.TotalProgress()
	s.WriteString(m.overallProgress.ViewAs(totalProg / 100.0))
	s.WriteString(fmt.Sprintf(" %.1f%%", totalProg))
	s.WriteString("\n")

	completed := len(m.queue.GetCompleted())
	total := len(m.queue.Entries)
	s.WriteString(fmt.Sprintf("\nCompleted: %d/%d", completed, total))

	return s.String()
}

func (m Model) renderHistoryScreen() string {
	var s strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170"))

	s.WriteString(titleStyle.Render("Download History"))
	s.WriteString("\n\n")

	completed := m.queue.GetCompleted()
	if len(completed) == 0 {
		s.WriteString(lipgloss.NewStyle().Faint(true).Render("No completed downloads"))
	} else {
		wrapWidth := 100 // TODO next: dynamically get terminal width

		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Width(wrapWidth - 4)

		for _, entry := range completed {
			if entry.Status == StatusCompleted {
				s.WriteString(successStyle.Render("✓ "))
			} else {
				s.WriteString(failStyle.Render("✗ "))
			}

			s.WriteString(fmt.Sprintf("%s", entry.URL))
			if entry.Title != "" {
				s.WriteString(fmt.Sprintf(" (%s)", entry.Title))
			}
			s.WriteString("\n")

			if entry.Status == StatusFailed && entry.Error != "" {
				// Wrap the error message to a reasonable width
				wrapped := errorStyle.Render(entry.Error)

				// Add indentation for each line
				indented := ""
				for _, line := range strings.Split(wrapped, "\n") {
					indented += errorStyle.Render(line) + "\n"
				}

				// Prefix first line with "Error:" for clarity
				indented = strings.Replace(indented, "  ", "  Error: ", 1)
				s.WriteString(indented)
			} else if entry.OutputPath != "" {
				s.WriteString(fmt.Sprintf("  Saved to: %s\n", entry.OutputPath))
			}
			s.WriteString("\n")
		}
	}

	return s.String()
}

func (m Model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var helps []string
	helps = append(helps, "tab: switch screen")

	if m.screen == ScreenInput {
		helps = append(helps, "enter: add URL")
		helps = append(helps, "d: start download")
		helps = append(helps, "backspace: remove last")
	}

	if m.screen == ScreenDownload || m.screen == ScreenInput {
		if len(m.queue.GetQueued()) > 0 {
			helps = append(helps, "d: start download")
		}
	}

	helps = append(helps, "q: quit")

	return helpStyle.Render(strings.Join(helps, " • "))
}
