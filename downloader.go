package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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

type Downloader struct {
	globalConfig Config
	runtime      string // JavaScript runtime (deno, bun, node, or empty)
}

func NewDownloader(config Config, runtime string) *Downloader {
	return &Downloader{
		globalConfig: config,
		runtime:      runtime,
	}
}

func (d *Downloader) Download(entry *DownloadEntry) tea.Cmd {
	return func() tea.Msg {
		// Merge configs
		finalConfig := d.globalConfig.MergeWith(entry.Config)

		// Ensure output directory exists
		_ = exec.Command("mkdir", "-p", finalConfig.OutputFolder).Run()

		// Build yt-dlp command based on format
		args := []string{
			"--newline",
			"--progress",
			"--remote-components", "ejs:github",
			"--extractor-args", "youtube:player-client=web_embedded,tv",
			"-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", finalConfig.OutputFolder),
		}

		// Format-specific options
		if finalConfig.Format == "mp3" {
			args = append(args,
				"-x",
				"--audio-format", "mp3",
				"--audio-quality", finalConfig.AudioQuality,
			)
		} else if finalConfig.Format == "m4a" {
			args = append(args,
				"-x",
				"--audio-format", "m4a",
				"--audio-quality", finalConfig.AudioQuality,
			)
		} else {
			// Video format
			if finalConfig.VideoQuality == "best" {
				args = append(args, "-f", "bestvideo+bestaudio")
			} else {
				args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%s]+bestaudio",
					strings.TrimSuffix(finalConfig.VideoQuality, "p")))
			}
			args = append(args, "--merge-output-format", finalConfig.Format)
		}

		args = append(args, entry.URL)

		cmd := exec.Command("yt-dlp", args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}

		if err := cmd.Start(); err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}

		// Parse progress
		scanner := bufio.NewScanner(stdout)
		progressRe := regexp.MustCompile(`(\d+\.\d+)%`)
		titleFound := false
		var title string
		var _ float64

		for scanner.Scan() {
			line := scanner.Text()

			// Try to extract title from initial output
			if !titleFound && strings.Contains(line, "[download] Destination:") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					title = strings.TrimSpace(parts[1])
					titleFound = true
				}
			}

			// Extract progress
			if matches := progressRe.FindStringSubmatch(line); len(matches) > 1 {
				if progress, err := strconv.ParseFloat(matches[1], 64); err == nil {
					_ = progress
					// Send progress update (we'll handle this via channels in the model)
				}
			}
		}

		err = cmd.Wait()

		outputPath := fmt.Sprintf("%s/%s.%s", finalConfig.OutputFolder, title, finalConfig.Format)
		if err != nil {
			return DownloadCompleteMsg{ID: entry.ID, Error: err}
		}

		return DownloadCompleteMsg{
			ID:         entry.ID,
			OutputPath: outputPath,
			Error:      nil,
		}
	}
}

// StartDownload initiates a download and returns a command that sends progress updates
func (d *Downloader) StartDownload(entry *DownloadEntry) tea.Cmd {
	return func() tea.Msg {
		finalConfig := d.globalConfig.MergeWith(entry.Config)

		// Create output directory
		if err := exec.Command("mkdir", "-p", finalConfig.OutputFolder).Run(); err != nil {
			return DownloadCompleteMsg{
				ID:    entry.ID,
				Error: fmt.Errorf("failed to create output directory: %v", err),
			}
		}

		args := []string{
			"--newline",
			"--no-playlist", // Don't download playlists by default
			"--remote-components", "ejs:github",
			"--extractor-args", "youtube:player-client=web_embedded,tv",
			"-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", finalConfig.OutputFolder),
		}

		// Add JavaScript runtime if available
		if d.runtime != "" {
			args = append(args, "--js-runtimes", d.runtime)
		}

		if finalConfig.Format == "mp3" {
			args = append(args, "-x", "--audio-format", "mp3", "--audio-quality", finalConfig.AudioQuality)
		} else if finalConfig.Format == "m4a" {
			args = append(args, "-x", "--audio-format", "m4a", "--audio-quality", finalConfig.AudioQuality)
		} else {
			if finalConfig.VideoQuality == "best" {
				args = append(args, "-f", "bestvideo+bestaudio")
			} else {
				args = append(args, "-f", fmt.Sprintf("bestvideo[height<=%s]+bestaudio",
					strings.TrimSuffix(finalConfig.VideoQuality, "p")))
			}
			args = append(args, "--merge-output-format", finalConfig.Format)
		}

		args = append(args, entry.URL)

		cmd := exec.Command("yt-dlp", args...)
		output, err := cmd.CombinedOutput()

		// Parse output for title and final status
		outputStr := string(output)
		lines := strings.Split(outputStr, "\n")
		var title string
		for _, line := range lines {
			if strings.Contains(line, "[download] Destination:") {
				parts := strings.Split(line, "Destination:")
				if len(parts) > 1 {
					title = strings.TrimSpace(parts[1])
					break
				}
			}
			// Also try to extract from the initial info line
			if strings.Contains(line, "[download]") && strings.Contains(line, "Downloading") {
				// Try to get title from this line
			}
		}

		if err != nil {
			// Include the actual output in the error message
			errorMsg := fmt.Sprintf("yt-dlp error: %v\n\nOutput:\n%s", err, outputStr)
			return DownloadCompleteMsg{
				ID:    entry.ID,
				Error: fmt.Errorf("%s", errorMsg),
			}
		}

		outputPath := fmt.Sprintf("%s/%s", finalConfig.OutputFolder, title)
		return DownloadCompleteMsg{
			ID:         entry.ID,
			OutputPath: outputPath,
			Error:      nil,
		}
	}
}
