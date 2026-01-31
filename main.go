package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// yt-dlp
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

	// js runtime
	runtime, found := detectRuntime()
	if !found {
		fmt.Println("No JavaScript runtime detected (deno, bun, or node).")
		fmt.Println("This is required for reliable downloads.")

		if askYesNo("Install Deno now? (recommended)") {
			if err := installDeno(); err != nil {
				fmt.Println("Auto-install failed:", err)
				printDenoGuide()
			} else {
				runtime, found = detectRuntime()
			}
		}

		if !found {
			fmt.Println("Continuing without a JS runtime. Things may break.")
			fmt.Println("Press enter to continue anyway...")
			fmt.Scanln()
		}
	}

	// run tui
	p := tea.NewProgram(initialModel(runtime), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
