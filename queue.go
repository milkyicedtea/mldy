package main

import (
	"fmt"
	"time"
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
	Config   EntryConfig

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

type Queue struct {
	Entries []DownloadEntry
	nextId  int
}

func NewQueue() *Queue {
	return &Queue{
		Entries: make([]DownloadEntry, 0),
		nextId:  1,
	}
}

func (q *Queue) add(url, title string, playlist *PlaylistMeta, config EntryConfig) {
	q.Entries = append(q.Entries, DownloadEntry{
		ID:       q.nextId,
		URL:      url,
		Title:    title,
		Status:   StatusQueued,
		Config:   config,
		Playlist: playlist,
	})
	q.nextId++
}

// Add queues a single video URL.
func (q *Queue) Add(url string, config EntryConfig) {
	q.add(url, "", nil, config)
}

// AddPlaylistItems expands a resolved playlist into individual queue entries.
func (q *Queue) AddPlaylistItems(items []PlaylistItem, playlistTitle string, config EntryConfig) {
	total := len(items)
	for i, item := range items {
		q.add(item.URL, item.Title, &PlaylistMeta{
			PlaylistTitle: playlistTitle,
			Index:         i + 1,
			Total:         total,
		}, config)
	}
}

func (q *Queue) GetQueued() []DownloadEntry {
	var out []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusQueued {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) GetActive() []DownloadEntry {
	var out []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusDownloading {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) GetCompleted() []DownloadEntry {
	var out []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusCompleted || e.Status == StatusFailed {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) Update(id int, fn func(*DownloadEntry)) {
	for i := range q.Entries {
		if q.Entries[i].ID == id {
			fn(&q.Entries[i])
			return
		}
	}
}

func (q *Queue) GetByID(id int) *DownloadEntry {
	for i := range q.Entries {
		if q.Entries[i].ID == id {
			return &q.Entries[i]
		}
	}
	return nil
}

func (q *Queue) Remove(id int) {
	for i, e := range q.Entries {
		if e.ID == id {
			q.Entries = append(q.Entries[:i], q.Entries[i+1:]...)
			return
		}
	}
}

func (q *Queue) TotalProgress() float64 {
	if len(q.Entries) == 0 {
		return 0
	}
	var total float64
	for _, e := range q.Entries {
		switch e.Status {
		case StatusCompleted:
			total += 100
		case StatusDownloading:
			total += e.Progress
		default: // StatusFailed, StatusQueued
			// count as 0% progress
		}
	}
	return total / float64(len(q.Entries))
}
