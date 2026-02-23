package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ---- messages ---------------------------------------------------------------

type ProgressMsg struct {
	ID       int
	Progress float64
	Title    string
}

type DownloadCompleteMsg struct {
	ID         int
	OutputPath string
	Error      error
}

// PlaylistItem is one video entry returned by --flat-playlist -J.
type PlaylistItem struct {
	URL   string
	Title string
}

// PlaylistResolvedMsg is sent after a playlist URL has been expanded into items.
type PlaylistResolvedMsg struct {
	OriginalURL   string
	PlaylistTitle string
	Items         []PlaylistItem
	Error         error
	Config        EntryConfig
}

// ---- downloader -------------------------------------------------------------

type Downloader struct {
	globalConfig Config
	runtime      string
}

func NewDownloader(config Config, runtime string) *Downloader {
	return &Downloader{globalConfig: config, runtime: runtime}
}

// baseArgs returns the args common to every yt-dlp invocation.
func (d *Downloader) baseArgs() []string {
	args := []string{"--newline", "--progress"}
	if d.runtime != "" {
		args = append(args, "--js-runtimes", d.runtime)
		if d.runtime == "deno" || d.runtime == "bun" {
			args = append(args, "--remote-components", "ejs:npm")
		} else {
			args = append(args, "--remote-components", "ejs:github")
		}
	}
	return args
}

// buildArgs constructs the full yt-dlp argument list for a single video download.
func (d *Downloader) buildArgs(cfg Config, url string) []string {
	args := d.baseArgs()
	args = append(args,
		"--no-playlist",
		"-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", cfg.OutputFolder),
	)

	// Resolve effective kind when set to auto.
	kind := cfg.Kind
	if kind == KindAuto {
		switch cfg.Format {
		case "mp3", "m4a", "opus", "flac", "wav", "aac":
			kind = KindAudio
		default:
			kind = KindVideo
		}
	}

	switch kind {
	case KindAudio:
		args = append(args,
			"-x",
			"--audio-format", cfg.Format,
			"--audio-quality", string(cfg.AudioQuality),
		)
	case KindVideo:
		if cfg.VideoQuality == "best" {
			args = append(args, "-f", "bestvideo+bestaudio")
		} else {
			height := strings.TrimSuffix(cfg.VideoQuality, "p")
			args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%s]+bestaudio", height))
		}
		if cfg.Format != "" && cfg.Format != "best" {
			args = append(args, "--merge-output-format", cfg.Format)
		}
	}

	args = append(args, url)
	return args
}

// ResolvePlaylist runs yt-dlp with --flat-playlist to enumerate playlist items
// without downloading anything, then sends a PlaylistResolvedMsg.
func (d *Downloader) ResolvePlaylist(url string, config EntryConfig) tea.Cmd {
	return func() tea.Msg {
		args := []string{
			"--flat-playlist",
			"--no-warnings",
			"-J", // dump JSON to stdout
			url,
		}
		// Include runtime args so auth/region handling is consistent.
		if d.runtime != "" {
			args = append([]string{"--js-runtimes", d.runtime}, args...)
		}

		out, err := exec.Command("yt-dlp", args...).Output()
		if err != nil {
			return PlaylistResolvedMsg{
				OriginalURL: url,
				Error:       fmt.Errorf("failed to resolve playlist: %w", err),
				Config:      config,
			}
		}

		// yt-dlp -J returns a single JSON object. For a playlist the top-level
		// "_type" is "playlist" and entries live in the "entries" array. For a
		// single video it's "video".
		var root struct {
			Type    string `json:"_type"`
			Title   string `json:"title"`
			Entries []struct {
				URL   string `json:"url"`
				Title string `json:"title"`
				ID    string `json:"id"`
			} `json:"entries"`
			// single-video fields
			WebpageURL string `json:"webpage_url"`
		}
		if err := json.Unmarshal(out, &root); err != nil {
			return PlaylistResolvedMsg{
				OriginalURL: url,
				Error:       fmt.Errorf("failed to parse playlist JSON: %w", err),
				Config:      config,
			}
		}

		if root.Type != "playlist" {
			// It's a single video — treat it as a one-item "playlist" so the
			// caller doesn't need a special code path.
			videoURL := root.WebpageURL
			if videoURL == "" {
				videoURL = url
			}
			return PlaylistResolvedMsg{
				OriginalURL:   url,
				PlaylistTitle: "",
				Items:         []PlaylistItem{{URL: videoURL}},
				Config:        config,
			}
		}

		items := make([]PlaylistItem, 0, len(root.Entries))
		for _, e := range root.Entries {
			u := e.URL
			// Flat-playlist entries sometimes only carry an ID, not a full URL.
			if !strings.HasPrefix(u, "http") && e.ID != "" {
				u = "https://www.youtube.com/watch?v=" + e.ID
			}
			items = append(items, PlaylistItem{URL: u, Title: e.Title})
		}

		return PlaylistResolvedMsg{
			OriginalURL:   url,
			PlaylistTitle: root.Title,
			Items:         items,
			Config:        config,
		}
	}
}

// StartDownload runs yt-dlp for a single entry, streaming progress via progressCh.
func (d *Downloader) StartDownload(entry *DownloadEntry, progressCh chan<- tea.Msg) tea.Cmd {
	return func() tea.Msg {
		finalConfig := d.globalConfig.MergeWith(entry.Config)

		if err := os.MkdirAll(finalConfig.OutputFolder, 0755); err != nil {
			return DownloadCompleteMsg{
				ID:    entry.ID,
				Error: fmt.Errorf("failed to create output directory: %w", err),
			}
		}

		args := d.buildArgs(finalConfig, entry.URL)
		cmd := exec.Command("yt-dlp", args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}

		if err := cmd.Start(); err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}

		progressRe := regexp.MustCompile(`(\d+\.?\d*)%`)
		// outputPath tracks the final file path, updated as yt-dlp prints its
		// destination lines. For audio, the post-conversion line wins.
		var outputPath string
		var displayTitle string

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// "[download] Destination: /path/to/file.webm" — initial download target.
			if strings.Contains(line, "[download] Destination:") {
				if parts := strings.SplitN(line, "Destination:", 2); len(parts) == 2 {
					outputPath = strings.TrimSpace(parts[1])
				}
			}

			// "[ExtractAudio] Destination: /path/to/file.mp3" — final converted file,
			// overwrites the webm path so we report the correct extension.
			if strings.Contains(line, "[ExtractAudio] Destination:") {
				if parts := strings.SplitN(line, "Destination:", 2); len(parts) == 2 {
					outputPath = strings.TrimSpace(parts[1])
				}
			}

			if displayTitle == "" && outputPath != "" {
				displayTitle = filepath.Base(outputPath)
			}

			if matches := progressRe.FindStringSubmatch(line); len(matches) > 1 {
				if progress, err := strconv.ParseFloat(matches[1], 64); err == nil {
					progressCh <- ProgressMsg{ID: entry.ID, Progress: progress, Title: displayTitle}
				}
			}
		}

		var stderrBuf strings.Builder
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			stderrBuf.WriteString(stderrScanner.Text())
			stderrBuf.WriteByte('\n')
		}

		if err := cmd.Wait(); err != nil {
			msg := fmt.Sprintf("yt-dlp error: %v", err)
			if s := strings.TrimSpace(stderrBuf.String()); s != "" {
				msg += "\n\n" + s
			}
			return DownloadCompleteMsg{ID: entry.ID, Error: fmt.Errorf("%s", msg)}
		}

		return DownloadCompleteMsg{
			ID:         entry.ID,
			OutputPath: outputPath,
		}
	}
}
