package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	rt "runtime"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"mldy/internal/deps"
	"mldy/internal/ui"
)

func main() {
	// ── yt-dlp ───────────────────────────────────────────────────────────────
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Println("yt-dlp not found.")
		if deps.AskYesNo("Install yt-dlp now?") {
			if err := deps.InstallYtDlp(); err != nil {
				fmt.Println("Auto-install failed:", err)
				deps.PrintYtDlpGuide()
				os.Exit(1)
			}
		} else {
			deps.PrintYtDlpGuide()
			os.Exit(1)
		}
	}

	// ── ffmpeg ────────────────────────────────────────────────────────────────
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		fmt.Println("ffmpeg not found.")
		if deps.AskYesNo("Install ffmpeg now?") {
			if err := deps.InstallFfmpeg(); err != nil {
				fmt.Println("Auto-install failed:", err)
				deps.PrintFfmpegGuide()
				os.Exit(1)
			}
		} else {
			deps.PrintFfmpegGuide()
			os.Exit(1)
		}
	}

	// ── update check (yt-dlp daily; ffmpeg piggybacks) ──────────────────────
	deps.CheckUpdates()

	// ── JS runtime ────────────────────────────────────────────────────────────
	runtime, found, meetsRecommended := deps.DetectRuntime()

	if !found {
		fmt.Println("No suitable JavaScript runtime found (deno ≥2, bun ≥1.0.31, node ≥20).")

		if deps.AskYesNo("Install Deno now? (recommended)") {
			if err := deps.InstallDeno(); err != nil {
				fmt.Println("Auto-install failed:", err)
				deps.PrintDenoGuide()
			} else {
				// On Windows the PATH isn't refreshed in the current process
				// after an installation, so we need to relaunch.
				if rt.GOOS == "windows" {
					restartSelf()
				}
				runtime, found, meetsRecommended = deps.DetectRuntime()
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
		if deps.AskYesNoDefaultNo(fmt.Sprintf("Upgrade %s to the recommended version?", runtime)) {
			if err := deps.UpdateRuntime(runtime); err != nil {
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
		ui.InitialModel(runtime),
		// tea.WithAltScreen(),
		// tea.WithMouseCellMotion(), // enables click events
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
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}
