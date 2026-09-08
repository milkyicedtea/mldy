package download

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

	"mldy/internal/config"
)

// ---- events -----------------------------------------------------------------

// ProgressEvent reports download progress for a queue entry.
type ProgressEvent struct {
	ID       int
	Progress float64
	Title    string
}

// CompleteEvent reports the terminal state of a single download.
type CompleteEvent struct {
	ID         int
	OutputPath string
	Error      string
}

// PlaylistItem is one video entry returned by --flat-playlist -J.
type PlaylistItem struct {
	URL   string
	Title string
}

// PlaylistResult is returned after a playlist URL has been resolved into items.
type PlaylistResult struct {
	OriginalURL   string
	PlaylistTitle string
	Items         []PlaylistItem
	Error         string
	Config        config.EntryConfig
}

// ---- downloader -------------------------------------------------------------

type Downloader struct {
	globalConfig config.Config
	runtime      string
}

func NewDownloader(cfg config.Config, runtime string) *Downloader {
	return &Downloader{globalConfig: cfg, runtime: runtime}
}

// SetConfig updates the global config and JS runtime; the next yt-dlp
// invocation picks them up. Callers must serialize access.
func (d *Downloader) SetConfig(cfg config.Config, runtime string) {
	d.globalConfig = cfg
	d.runtime = runtime
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
func (d *Downloader) buildArgs(cfg config.Config, url string) []string {
	args := d.baseArgs()
	args = append(args,
		"--no-playlist",
		"--embed-thumbnail",
		"--embed-metadata",
		"-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", cfg.OutputFolder),
	)

	// Resolve effective kind when set to auto.
	kind := cfg.Kind
	if kind == config.KindAuto {
		switch cfg.Format {
		case "mp3", "m4a", "opus", "flac", "wav", "aac":
			kind = config.KindAudio
		default:
			kind = config.KindVideo
		}
	}

	switch kind {
	case config.KindAudio:
		args = append(args,
			"-x",
			"--audio-format", cfg.Format,
			"--audio-quality", string(cfg.AudioQuality),
		)
	case config.KindVideo:
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
// without downloading anything. It is synchronous; callers run it in a goroutine.
func (d *Downloader) ResolvePlaylist(url string, cfg config.EntryConfig) PlaylistResult {
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
		return PlaylistResult{
			OriginalURL: url,
			Error:       fmt.Sprintf("failed to resolve playlist: %v", err),
			Config:      cfg,
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
		return PlaylistResult{
			OriginalURL: url,
			Error:       fmt.Sprintf("failed to parse playlist JSON: %v", err),
			Config:      cfg,
		}
	}

	if root.Type != "playlist" {
		// It's a single video — treat it as a one-item "playlist" so the
		// caller doesn't need a special code path.
		videoURL := root.WebpageURL
		if videoURL == "" {
			videoURL = url
		}
		return PlaylistResult{
			OriginalURL:   url,
			PlaylistTitle: "",
			Items:         []PlaylistItem{{URL: videoURL, Title: root.Title}},
			Config:        cfg,
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

	return PlaylistResult{
		OriginalURL:   url,
		PlaylistTitle: root.Title,
		Items:         items,
		Config:        cfg,
	}
}

// StartDownload runs yt-dlp for a single entry, streaming progress into
// progressCh. It is synchronous; callers run it in a goroutine.
func (d *Downloader) StartDownload(entry *DownloadEntry, progressCh chan<- ProgressEvent) CompleteEvent {
	finalConfig := d.globalConfig.MergeWith(entry.Config)

	if err := os.MkdirAll(finalConfig.OutputFolder, 0755); err != nil {
		return CompleteEvent{
			ID:    entry.ID,
			Error: fmt.Sprintf("failed to create output directory: %v", err),
		}
	}

	args := d.buildArgs(finalConfig, entry.URL)
	cmd := exec.Command("yt-dlp", args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CompleteEvent{ID: entry.ID, Error: err.Error()}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return CompleteEvent{ID: entry.ID, Error: err.Error()}
	}

	if err := cmd.Start(); err != nil {
		return CompleteEvent{ID: entry.ID, Error: err.Error()}
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
				progressCh <- ProgressEvent{ID: entry.ID, Progress: progress, Title: displayTitle}
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
		return CompleteEvent{ID: entry.ID, Error: msg}
	}

	return CompleteEvent{
		ID:         entry.ID,
		OutputPath: outputPath,
	}
}
