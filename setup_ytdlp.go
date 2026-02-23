package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"
)

func installYtDlp() error {
	var prefix []string
	if rt.GOOS != "windows" && os.Geteuid() != 0 {
		prefix = []string{"sudo"}
	}

	switch rt.GOOS {
	case "darwin":
		return run("brew", "install", "yt-dlp")
	case "windows":
		installDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WindowsApps")
		destPath := filepath.Join(installDir, "yt-dlp.exe")

		resp, err := http.Get("https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe")
		if err != nil {
			return fmt.Errorf("failed to download yt-dlp: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download yt-dlp: HTTP %d", resp.StatusCode)
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to create yt-dlp.exe: %w", err)
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			return fmt.Errorf("failed to write yt-dlp.exe: %w", err)
		}

		return nil
	case "linux":
		id, idLike, err := detectLinuxDistro()
		if err != nil {
			return err
		}

		switch {
		case id == "debian" ||
			id == "ubuntu" ||
			strings.Contains(idLike, "debian"):
			return runPackageManager(prefix, "apt", "install", "-y", "yt-dlp")

		case id == "fedora" ||
			strings.Contains(idLike, "rhel") ||
			strings.Contains(idLike, "fedora") ||
			strings.Contains(idLike, "centos"):
			return runPackageManager(prefix, "dnf", "install", "-y", "yt-dlp")

		case id == "arch" ||
			strings.Contains(idLike, "arch"):
			return runPackageManager(prefix, "pacman", "-S", "yt-dlp")

		case id == "opensuse" ||
			strings.Contains(idLike, "suse") ||
			strings.Contains(idLike, "opensuse"):
			return runPackageManager(prefix, "zypper", "install", "-y", "yt-dlp")

		case id == "alpine":
			return runPackageManager(prefix, "apk", "add", "yt-dlp")

		default:
			return fmt.Errorf("unsupported distro: %s (%s)", id, idLike)
		}
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
