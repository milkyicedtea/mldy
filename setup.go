package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	rt "runtime"
	"strings"
)

func askYesNo(prompt string) bool {
	var input string
	fmt.Printf("%s [Y/n]: ", prompt)
	fmt.Scanln(&input)

	input = strings.ToLower(strings.TrimSpace(input))
	return input == "" || input == "y" || input == "yes"
}

func run(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func installYtDlp() error {
	var prefix []string
	if os.Geteuid() != 0 { // check if not root
		prefix = []string{"sudo"}
	}

	switch rt.GOOS {
	case "darwin":
		return run("brew", "install", "yt-dlp")
	case "linux":
		args := append(prefix, "apt", "install", "-y", "yt-dlp")
		return run(args[0], args[1:]...)
	case "windows":
		return run("winget", "install", "yt-dlp")
	default:
		return errors.New("unsupported OS")
	}
}

func printYtDlpGuide() {
	fmt.Println("\nManual yt-dlp installation:")
	fmt.Println("macOS:   brew install yt-dlp")
	fmt.Println("Linux:   check your distro's package manager, or use github yt-dlp releases")
	fmt.Println("Windows: winget install yt-dlp")
	fmt.Println("Or: https://github.com/yt-dlp/yt-dlp")
}

func installDeno() error {
	switch rt.GOOS {
	case "darwin", "linux":
		return run("sh", "-c", "curl -fsSL https://deno.land/install.sh | sh")
	case "windows":
		return run("winget", "install", "DenoLand.Deno")
	default:
		return errors.New("unsupported OS")
	}
}

func printDenoGuide() {
	fmt.Println("\nManual Deno installation:")
	fmt.Println("macOS/Linux: curl -fsSL https://deno.land/install.sh | sh")
	fmt.Println("Windows:     winget install DenoLand.Deno")
	fmt.Println("Or: https://deno.land/")
}

func detectRuntime() (string, bool) {
	runtimes := []string{"deno", "bun", "node"}

	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			return runtime, true
		}
	}

	return "", false
}
