package download

import (
	"fmt"
	"time"

	"mldy/internal/config"
)

type DownloadStatus int

const (
	StatusQueued DownloadStatus = iota
	StatusDownloading
	StatusCompleted
	StatusFailed
)

func (s DownloadStatus) String() string {
	switch s {
	case StatusQueued:
		return "Queued"
	case StatusDownloading:
		return "Downloading"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// PlaylistMeta is set on entries that were expanded from a playlist.
type PlaylistMeta struct {
	BatchID       string // Unique ID for this specific addition
	PlaylistTitle string
	Index         int // 1-based position within the playlist
	Total         int // total number of items in the playlist
}

type DownloadEntry struct {
	ID       int
	URL      string
	Title    string
	Status   DownloadStatus
	Progress float64
	Error    string
	Config   config.EntryConfig

	// Non-nil when this entry was expanded from a playlist.
	Playlist *PlaylistMeta

	StartTime  time.Time
	EndTime    time.Time
	OutputPath string
}

// DisplayTitle returns the best available label for UI display.
func (e *DownloadEntry) DisplayTitle() string {
	if e.Title != "" {
		return e.Title
	}
	return e.URL
}

// PlaylistLabel returns a short prefix like "[My Playlist 3/12]" or "".
func (e *DownloadEntry) PlaylistLabel() string {
	if e.Playlist == nil {
		return ""
	}
	return fmt.Sprintf("[%s %d/%d] ", e.Playlist.PlaylistTitle, e.Playlist.Index, e.Playlist.Total)
}
