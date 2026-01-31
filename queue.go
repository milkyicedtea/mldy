package main

import (
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

type DownloadEntry struct {
	ID         int
	URL        string
	Title      string
	Status     DownloadStatus
	Progress   float64
	Error      string
	Config     EntryConfig
	StartTime  time.Time
	EndTime    time.Time
	OutputPath string
}

type Queue struct {
	Entries      []DownloadEntry
	CurrentIndex int
	nextId       int
}

func NewQueue() *Queue {
	return &Queue{
		Entries:      make([]DownloadEntry, 0),
		CurrentIndex: -1,
		nextId:       1,
	}
}

func (q *Queue) Add(url string, config EntryConfig) {
	entry := DownloadEntry{
		ID:     q.nextId,
		URL:    url,
		Status: StatusQueued,
		Config: config,
	}
	q.nextId++
	q.Entries = append(q.Entries, entry)
}

func (q *Queue) GetQueued() []DownloadEntry {
	var queued []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusQueued {
			queued = append(queued, e)
		}
	}
	return queued
}

func (q *Queue) GetActive() []DownloadEntry {
	var active []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusDownloading {
			active = append(active, e)
		}
	}
	return active
}

func (q *Queue) GetCompleted() []DownloadEntry {
	var completed []DownloadEntry
	for _, e := range q.Entries {
		if e.Status == StatusCompleted || e.Status == StatusFailed {
			completed = append(completed, e)
		}
	}
	return completed
}

func (q *Queue) Update(id int, updates func(entry *DownloadEntry)) {
	for i := range q.Entries {
		if q.Entries[i].ID == id {
			updates(&q.Entries[i])
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
	for i, entry := range q.Entries {
		if entry.ID == id {
			q.Entries = append(q.Entries[:i], q.Entries[i+1:]...)
			break
		}
	}
}

func (q *Queue) TotalProgress() float64 {
	if len(q.Entries) == 0 {
		return 0
	}

	var total float64
	for _, e := range q.Entries {
		if e.Status == StatusCompleted {
			total += 100
		} else if e.Status == StatusDownloading {
			total += e.Progress
		}
	}
	return total / float64(len(q.Entries))
}
