package main

import (
	"errors"
	"fmt"
	"os"
	rt "runtime"
	"strings"
)

func installFfmpeg() error {
	var prefix []string
	if rt.GOOS != "windows" && os.Geteuid() != 0 {
		prefix = []string{"sudo"}
	}

	switch rt.GOOS {
	case "darwin":
		return run("brew", "install", "ffmpeg")
	case "windows":
		return run("winget", "install", "-e", "--id", "Gyan.FFmpeg", "--source", "winget")
	case "linux":
		id, idLike, err := detectLinuxDistro()
		if err != nil {
			return err
		}

		switch {
		case id == "debian" ||
			id == "ubuntu" ||
			strings.Contains(idLike, "debian"):
			return runPackageManager(prefix, "apt", "install", "-y", "ffmpeg")

		case id == "fedora" ||
			strings.Contains(idLike, "rhel") ||
			strings.Contains(idLike, "fedora") ||
			strings.Contains(idLike, "centos"):
			return runPackageManager(prefix, "dnf", "install", "-y", "ffmpeg")

		case id == "arch" ||
			strings.Contains(idLike, "arch"):
			return runPackageManager(prefix, "pacman", "-S", "ffmpeg")

		case id == "opensuse" ||
			strings.Contains(idLike, "suse") ||
			strings.Contains(idLike, "opensuse"):
			return runPackageManager(prefix, "zypper", "install", "-y", "ffmpeg")

		case id == "alpine":
			return runPackageManager(prefix, "apk", "add", "ffmpeg")

		default:
			return fmt.Errorf("unsupported distro: %s (%s)", id, idLike)
		}
	default:
		return errors.New("unsupported OS")
	}
}

func printFfmpegGuide() {
	fmt.Println("\nManual ffmpeg installation:")
	fmt.Println("macOS:   brew install ffmpeg")
	fmt.Println("Linux:   check your distro's package manager")
	fmt.Println("Windows: winget install -e --id Gyan.FFmpeg --source winget")
	fmt.Println("Or: https://ffmpeg.org/download.html")
}
