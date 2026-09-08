package download

import (
	"fmt"
	"time"

	"mldy/internal/config"
)

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

func (q *Queue) add(url, title string, playlist *PlaylistMeta, cfg config.EntryConfig) {
	q.Entries = append(q.Entries, DownloadEntry{
		ID:       q.nextId,
		URL:      url,
		Title:    title,
		Status:   StatusQueued,
		Config:   cfg,
		Playlist: playlist,
	})
	q.nextId++
}

// Add queues a single video URL.
func (q *Queue) Add(url string, cfg config.EntryConfig) {
	q.add(url, "", nil, cfg)
}

// AddPlaylistItems expands a resolved playlist into individual queue entries.
func (q *Queue) AddPlaylistItems(items []PlaylistItem, playlistTitle string, cfg config.EntryConfig) {
	total := len(items)
	batchID := fmt.Sprintf("pl-%d", time.Now().UnixNano())
	for i, item := range items {
		q.add(item.URL, item.Title, &PlaylistMeta{
			BatchID:       batchID,
			PlaylistTitle: playlistTitle,
			Index:         i + 1,
			Total:         total,
		}, cfg)
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

// RemoveBatch removes every entry belonging to the given playlist batch.
func (q *Queue) RemoveBatch(batchID string) {
	var kept []DownloadEntry
	for _, e := range q.Entries {
		if e.Playlist == nil || e.Playlist.BatchID != batchID {
			kept = append(kept, e)
		}
	}
	q.Entries = kept
}

func (q *Queue) Clear() {
	q.Entries = nil
	q.nextId = 1
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
