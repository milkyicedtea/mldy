package app

import (
	"fmt"
	"io"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"mldy/internal/config"
	"mldy/internal/deps"
	"mldy/internal/download"
)

// Entry is the JSON view of a queue entry sent to the frontend.
type Entry struct {
	ID         int                    `json:"id"`
	URL        string                 `json:"url"`
	Title      string                 `json:"title"`
	Status     string                 `json:"status"`
	Progress   float64                `json:"progress"`
	Error      string                 `json:"error,omitempty"`
	Playlist   *download.PlaylistMeta `json:"playlist,omitempty"`
	OutputPath string                 `json:"outputPath,omitempty"`
}

// State is the full snapshot emitted to the frontend on every change.
type State struct {
	Entries   []Entry       `json:"entries"`
	Resolving int           `json:"resolving"`
	IsRunning bool          `json:"isRunning"`
	Config    config.Config `json:"config"`
	Runtime   string        `json:"runtime"`
}

// Service owns the queue and drives downloads. It is the sole source of
// "state" events; the frontend never mutates state directly, it only calls
// these methods.
type Service struct {
	mu        sync.Mutex
	queue     *download.Queue
	dl        *download.Downloader
	cfg       config.Config
	runtime   string
	isRunning bool
	resolving int
}

func NewService() *Service {
	cfg, _ := config.LoadConfig()
	// The GUI has no console; keep status chatter out of the way unless a
	// deps operation temporarily installs a streaming writer.
	deps.SetOutput(io.Discard)
	runtime, _, found := deps.DetectRuntime()
	if !found {
		runtime = ""
	}
	return &Service{
		queue:   download.NewQueue(),
		dl:      download.NewDownloader(cfg, runtime),
		cfg:     cfg,
		runtime: runtime,
	}
}

func (s *Service) statusString(st download.DownloadStatus) string {
	switch st {
	case download.StatusQueued:
		return "queued"
	case download.StatusDownloading:
		return "downloading"
	case download.StatusCompleted:
		return "completed"
	case download.StatusFailed:
		return "failed"
	}
	return "unknown"
}

func (s *Service) snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := make([]Entry, len(s.queue.Entries))
	for i, e := range s.queue.Entries {
		entries[i] = Entry{
			ID:         e.ID,
			URL:        e.URL,
			Title:      e.Title,
			Status:     s.statusString(e.Status),
			Progress:   e.Progress,
			Error:      e.Error,
			Playlist:   e.Playlist,
			OutputPath: e.OutputPath,
		}
	}
	return State{
		Entries:   entries,
		Resolving: s.resolving,
		IsRunning: s.isRunning,
		Config:    s.cfg,
		Runtime:   s.runtime,
	}
}

// emit publishes the current state to the frontend. Call without holding mu.
func (s *Service) emit() {
	if a := application.Get(); a != nil {
		a.Event.Emit("state", s.snapshot())
	}
}

// GetState returns the current snapshot without emitting an event.
func (s *Service) GetState() State {
	return s.snapshot()
}

// AddURL resolves a URL (single video or playlist) and adds the results to
// the queue. Mirrors the TUI flow: increment "resolving", resolve in the
// background, then either add playlist items, add the single video, or add a
// failed entry carrying the resolve error.
func (s *Service) AddURL(rawURL string) {
	s.mu.Lock()
	s.resolving++
	s.mu.Unlock()
	s.emit()

	go func() {
		res := s.dl.ResolvePlaylist(rawURL, config.EntryConfig{})

		s.mu.Lock()
		s.resolving--
		if res.Error != "" {
			s.queue.Add(rawURL, res.Config)
			id := s.queue.Entries[len(s.queue.Entries)-1].ID
			s.queue.Update(id, func(e *download.DownloadEntry) {
				e.Status = download.StatusFailed
				e.Error = "playlist resolve error: " + res.Error
			})
		} else if res.PlaylistTitle != "" {
			s.queue.AddPlaylistItems(res.Items, res.PlaylistTitle, res.Config)
		} else if len(res.Items) > 0 {
			item := res.Items[0]
			s.queue.Add(item.URL, res.Config)
			if item.Title != "" {
				id := s.queue.Entries[len(s.queue.Entries)-1].ID
				s.queue.Update(id, func(e *download.DownloadEntry) { e.Title = item.Title })
			}
		}
		s.mu.Unlock()
		s.emit()
	}()
}

// Start begins downloading the queue sequentially, one entry at a time.
// Returns false if a run is already in progress, URLs are still resolving,
// or the queue is empty — mirroring the TUI's tryStartDownloads.
func (s *Service) Start() bool {
	s.mu.Lock()
	if s.isRunning || s.resolving > 0 || len(s.queue.GetQueued()) == 0 {
		s.mu.Unlock()
		return false
	}
	s.isRunning = true
	s.mu.Unlock()
	s.emit()

	go s.runQueue()
	return true
}

// runQueue drains the queue front-to-back. When the last entry completes it
// clears isRunning so a later Start can begin a fresh run.
func (s *Service) runQueue() {
	for {
		s.mu.Lock()
		queued := s.queue.GetQueued()
		if len(queued) == 0 {
			s.isRunning = false
			s.mu.Unlock()
			s.emit()
			return
		}
		id := queued[0].ID
		s.queue.Update(id, func(e *download.DownloadEntry) {
			e.Status = download.StatusDownloading
		})
		entry := s.queue.GetByID(id)
		s.mu.Unlock()
		s.emit()

		if entry == nil { // removed between snapshot and start
			continue
		}

		ch := make(chan download.ProgressEvent, 64)
		pumpDone := make(chan struct{})
		go func() {
			defer close(pumpDone)
			for ev := range ch {
				s.mu.Lock()
				s.queue.Update(ev.ID, func(e *download.DownloadEntry) {
					e.Progress = ev.Progress
					if ev.Title != "" {
						e.Title = ev.Title
					}
				})
				s.mu.Unlock()
				s.emit()
			}
		}()

		complete := s.dl.StartDownload(entry, ch)
		close(ch)
		<-pumpDone

		s.mu.Lock()
		s.queue.Update(complete.ID, func(e *download.DownloadEntry) {
			if complete.Error != "" {
				e.Status = download.StatusFailed
				e.Error = complete.Error
			} else {
				e.Status = download.StatusCompleted
				e.OutputPath = complete.OutputPath
			}
		})
		s.mu.Unlock()
		s.emit()
	}
}

// RemoveEntry removes a single entry by ID.
func (s *Service) RemoveEntry(id int) {
	s.mu.Lock()
	s.queue.Remove(id)
	s.mu.Unlock()
	s.emit()
}

// RemoveBatch removes every entry of a playlist batch.
func (s *Service) RemoveBatch(batchID string) {
	s.mu.Lock()
	s.queue.RemoveBatch(batchID)
	s.mu.Unlock()
	s.emit()
}

// RemoveLast removes the most recently queued entry (TUI's backspace action).
func (s *Service) RemoveLast() {
	s.mu.Lock()
	if queued := s.queue.GetQueued(); len(queued) > 0 {
		s.queue.Remove(queued[len(queued)-1].ID)
	}
	s.mu.Unlock()
	s.emit()
}

// ClearAll empties the queue, including history (TUI's ctrl+r).
func (s *Service) ClearAll() {
	s.mu.Lock()
	s.queue.Clear()
	s.mu.Unlock()
	s.emit()
}

// UpdateConfig validates and persists a new global configuration, applies it
// to the downloader, and emits the new state. A pinned JS runtime must be
// installed; "auto" re-runs detection (deno > bun > node).
func (s *Service) UpdateConfig(cfg config.Config) error {
	switch {
	case !cfg.Kind.IsValid():
		return fmt.Errorf("invalid kind: %q", cfg.Kind)
	case !cfg.AudioQuality.IsValid():
		return fmt.Errorf("invalid audio quality: %q", cfg.AudioQuality)
	case !config.ValidJSRuntime(cfg.JSRuntime):
		return fmt.Errorf("invalid JS runtime: %q", cfg.JSRuntime)
	}
	if cfg.JSRuntime == "" {
		cfg.JSRuntime = "auto"
	}

	s.mu.Lock()
	if cfg.JSRuntime != "auto" {
		if !deps.RuntimeAvailable(cfg.JSRuntime) {
			s.mu.Unlock()
			return fmt.Errorf("js runtime %q is not installed", cfg.JSRuntime)
		}
		s.runtime = cfg.JSRuntime
	} else if runtime, _, found := deps.DetectRuntime(); found {
		s.runtime = runtime
	}
	s.cfg = cfg
	s.dl.SetConfig(cfg, s.runtime)
	s.mu.Unlock()

	if err := config.Save(cfg); err != nil {
		return err
	}
	s.emit()
	return nil
}

// PickOutputFolder opens a native directory chooser (GTK on Linux, WebView2
// on Windows, NSOpenPanel on macOS) and returns the selected path. An empty
// string means the dialog was cancelled.
func (s *Service) PickOutputFolder() (string, error) {
	a := application.Get()
	if a == nil {
		return "", fmt.Errorf("application not ready")
	}
	builder := a.Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		SetTitle("Choose output folder")
	if win := a.Window.Current(); win != nil {
		builder = builder.AttachToWindow(win)
	}
	return builder.PromptForSingleSelection()
}
