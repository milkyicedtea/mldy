package ui

import (
	"fmt"
	"time"

	"mldy/internal/config"
	"mldy/internal/download"
)

type Queue struct {
	Entries []download.DownloadEntry
	nextId  int
}

func NewQueue() *Queue {
	return &Queue{
		Entries: make([]download.DownloadEntry, 0),
		nextId:  1,
	}
}

func (q *Queue) add(url, title string, playlist *download.PlaylistMeta, cfg config.EntryConfig) {
	q.Entries = append(q.Entries, download.DownloadEntry{
		ID:       q.nextId,
		URL:      url,
		Title:    title,
		Status:   download.StatusQueued,
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
func (q *Queue) AddPlaylistItems(items []download.PlaylistItem, playlistTitle string, cfg config.EntryConfig) {
	total := len(items)
	batchID := fmt.Sprintf("pl-%d", time.Now().UnixNano())
	for i, item := range items {
		q.add(item.URL, item.Title, &download.PlaylistMeta{
			BatchID:       batchID,
			PlaylistTitle: playlistTitle,
			Index:         i + 1,
			Total:         total,
		}, cfg)
	}
}

func (q *Queue) GetQueued() []download.DownloadEntry {
	var out []download.DownloadEntry
	for _, e := range q.Entries {
		if e.Status == download.StatusQueued {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) GetActive() []download.DownloadEntry {
	var out []download.DownloadEntry
	for _, e := range q.Entries {
		if e.Status == download.StatusDownloading {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) GetCompleted() []download.DownloadEntry {
	var out []download.DownloadEntry
	for _, e := range q.Entries {
		if e.Status == download.StatusCompleted || e.Status == download.StatusFailed {
			out = append(out, e)
		}
	}
	return out
}

func (q *Queue) Update(id int, fn func(*download.DownloadEntry)) {
	for i := range q.Entries {
		if q.Entries[i].ID == id {
			fn(&q.Entries[i])
			return
		}
	}
}

func (q *Queue) GetByID(id int) *download.DownloadEntry {
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
		case download.StatusCompleted:
			total += 100
		case download.StatusDownloading:
			total += e.Progress
		default: // download.StatusFailed, download.StatusQueued
			// count as 0% progress
		}
	}
	return total / float64(len(q.Entries))
}
