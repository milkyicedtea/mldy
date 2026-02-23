package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	rt "runtime"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func main() {
	// ── yt-dlp ───────────────────────────────────────────────────────────────
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Println("yt-dlp not found.")
		if askYesNo("Install yt-dlp now?") {
			if err := installYtDlp(); err != nil {
				fmt.Println("Auto-install failed:", err)
				printYtDlpGuide()
				os.Exit(1)
			}
		} else {
			printYtDlpGuide()
			os.Exit(1)
		}
	}

	// ── ffmpeg ────────────────────────────────────────────────────────────────
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("ffmpeg not found.")
		if askYesNo("Install ffmpeg now?") {
			if err := installFfmpeg(); err != nil {
				fmt.Println("Auto-install failed:", err)
				printFfmpegGuide()
				os.Exit(1)
			}
		} else {
			printFfmpegGuide()
			os.Exit(1)
		}
	}

	// ── JS runtime ────────────────────────────────────────────────────────────
	runtime, found, meetsRecommended := detectRuntime()

	if !found {
		fmt.Println("No suitable JavaScript runtime found (deno ≥2, bun ≥1.0.31, node ≥20).")

		if askYesNo("Install Deno now? (recommended)") {
			if err := installDeno(); err != nil {
				fmt.Println("Auto-install failed:", err)
				printDenoGuide()
			} else {
				// On Windows the PATH isn't refreshed in the current process
				// after an installation, so we need to relaunch.
				if rt.GOOS == "windows" {
					restartSelf()
				}
				runtime, found, meetsRecommended = detectRuntime()
			}
		}

		if !found {
			fmt.Println("Continuing without a JS runtime. Things may break.")
			fmt.Println("Press enter to continue anyway...")
			fmt.Scanln()
		}
	}

	// Offer an upgrade only when the installed version is below the recommended threshold.
	if found && !meetsRecommended {
		if askYesNoDefaultNo(fmt.Sprintf("Upgrade %s to the recommended version?", runtime)) {
			if err := updateRuntime(runtime); err != nil {
				fmt.Printf("Upgrade failed: %v\n", err)
			} else {
				fmt.Printf("%s upgraded successfully.\n", runtime)
				if rt.GOOS == "windows" {
					restartSelf()
				}
			}
		}
	}

	// ── TUI ───────────────────────────────────────────────────────────────────
	zone.NewGlobal()
	defer zone.Close()

	p := tea.NewProgram(
		initialModel(runtime),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // enables click events
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// restartSelf re-executes the current binary with the same arguments and exits.
// Used on Windows after installs/updates so the new PATH is picked up.
func restartSelf() {
	self, err := os.Executable()
	if err != nil {
		fmt.Println("Could not determine executable path; please restart manually.")
		os.Exit(0)
	}

	fmt.Println("Restarting to apply PATH changes...")
	cmd := exec.Command(self, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}
