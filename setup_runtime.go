package main

import (
	"errors"
	"fmt"
	"os/exec"
	rt "runtime"
	"strconv"
	"strings"
)

// runtimeVersionInfo holds version requirements for a JS runtime.
type runtimeVersionInfo struct {
	minimum [3]int
	// recommended is the soft floor above which the installation is considered ideal.
	// A zero value means "always nudge" (used for fast-moving runtimes like
	// deno and bun where we can't pin a specific recommended version).
	recommended      [3]int
	recommendedLabel string
}

var runtimeVersions = map[string]runtimeVersionInfo{
	"deno": {
		minimum:          [3]int{2, 0, 0},
		recommended:      [3]int{}, // zero → always nudge
		recommendedLabel: "latest",
	},
	"node": {
		// 24 LTS is recommended; 20 is the absolute floor.
		minimum:          [3]int{20, 0, 0},
		recommended:      [3]int{24, 0, 0},
		recommendedLabel: "24 LTS",
	},
	"bun": {
		minimum:          [3]int{1, 0, 31},
		recommended:      [3]int{}, // zero → always nudge
		recommendedLabel: "latest",
	},
}

// parseVersion parses "v20.11.0", "2.0.0", "deno 2.0.0 (...)" etc. into [major, minor, patch].
func parseVersion(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	v = strings.Fields(v)[0]
	parts := strings.SplitN(v, ".", 3)

	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Strip non-numeric suffix (e.g. "-rc1")
		for j, c := range p {
			if c < '0' || c > '9' {
				p = p[:j]
				break
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return result, fmt.Errorf("invalid version segment %q in %q", p, v)
		}
		result[i] = n
	}
	return result, nil
}

// versionAtLeast returns true if actual >= minimum.
func versionAtLeast(actual, minimum [3]int) bool {
	for i := range minimum {
		if actual[i] > minimum[i] {
			return true
		}
		if actual[i] < minimum[i] {
			return false
		}
	}
	return true
}

// getRuntimeVersion runs the binary and extracts its version string.
func getRuntimeVersion(runtime string) (string, error) {
	var (
		out []byte
		err error
	)

	switch runtime {
	case "node":
		out, err = exec.Command("node", "--version").Output()
	case "deno":
		out, err = exec.Command("deno", "--version").Output()
	case "bun":
		out, err = exec.Command("bun", "--version").Output()
	default:
		return "", fmt.Errorf("unknown runtime: %s", runtime)
	}
	if err != nil {
		return "", fmt.Errorf("could not get %s version: \"%w\".\n"+
			"If you installed deno through yt-dlp, this may be normal", runtime, err)
	}

	// node  → "v20.11.0"
	// deno  → "deno 2.0.0 (release, ...)" — version is second field
	// bun   → "1.0.31"
	raw := strings.TrimSpace(string(out))
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "", fmt.Errorf("empty version output from %s", runtime)
	}

	if runtime == "deno" && len(fields) >= 2 {
		return fields[1], nil
	}
	return fields[0], nil
}

// checkRuntimeVersion verifies the installed version meets the minimum and
// recommended thresholds.
// Returns (meetsMinimum, meetsRecommended, error).
func checkRuntimeVersion(runtime string) (bool, bool, error) {
	info, ok := runtimeVersions[runtime]
	if !ok {
		return false, false, fmt.Errorf("unknown runtime: %s", runtime)
	}

	versionStr, err := getRuntimeVersion(runtime)
	if err != nil {
		return false, false, err
	}

	parsed, err := parseVersion(versionStr)
	if err != nil {
		return false, false, fmt.Errorf("could not parse %s version %q: %w", runtime, versionStr, err)
	}

	minStr := fmt.Sprintf("%d.%d.%d", info.minimum[0], info.minimum[1], info.minimum[2])

	if !versionAtLeast(parsed, info.minimum) {
		fmt.Printf("⚠  %s %s is below the minimum required %s (recommended: %s).\n",
			runtime, versionStr, minStr, info.recommendedLabel)
		return false, false, nil
	}

	// Zero recommended means "always nudge" (deno/bun track latest).
	zeroVersion := [3]int{}
	meetsRecommended := info.recommended != zeroVersion && versionAtLeast(parsed, info.recommended)

	if meetsRecommended {
		fmt.Printf("✓  %s %s detected.\n", runtime, versionStr)
	} else {
		fmt.Printf("✓  %s %s detected (upgrade to %s recommended).\n",
			runtime, versionStr, info.recommendedLabel)
	}

	return true, meetsRecommended, nil
}

// updateRuntime attempts to upgrade the given runtime to its latest version.
// Returns true if the update succeeded and the program should restart on Windows.
func updateRuntime(runtime string) error {
	fmt.Printf("Updating %s...\n", runtime)
	switch runtime {
	case "deno":
		switch rt.GOOS {
		case "darwin", "linux":
			return run("deno", "upgrade")
		case "windows":
			return run("deno", "upgrade")
		}
	case "bun":
		switch rt.GOOS {
		case "darwin", "linux":
			return run("bun", "upgrade")
		case "windows":
			return run("bun", "upgrade")
		}
	case "node":
		// Node doesn't self-update; use the system package manager or fnm/nvm.
		switch rt.GOOS {
		case "darwin":
			return run("brew", "upgrade", "node")
		case "windows":
			return run("winget", "upgrade", "--id", "OpenJS.NodeJS.LTS", "--source", "winget")
		case "linux":
			id, idLike, err := detectLinuxDistro()
			if err != nil {
				return err
			}
			switch {
			case id == "debian" || id == "ubuntu" || strings.Contains(idLike, "debian"):
				return run("sudo", "apt", "upgrade", "-y", "nodejs")
			case id == "fedora" || strings.Contains(idLike, "rhel") ||
				strings.Contains(idLike, "fedora") || strings.Contains(idLike, "centos"):
				return run("sudo", "dnf", "upgrade", "-y", "nodejs")
			case id == "arch" || strings.Contains(idLike, "arch"):
				return run("sudo", "pacman", "-Syu", "nodejs")
			case id == "opensuse" || strings.Contains(idLike, "suse") ||
				strings.Contains(idLike, "opensuse"):
				return run("sudo", "zypper", "update", "-y", "nodejs")
			case id == "alpine":
				return run("sudo", "apk", "upgrade", "nodejs")
			default:
				return fmt.Errorf("unsupported distro for node upgrade: %s", id)
			}
		}
	}
	return errors.New("unsupported OS")
}

// installDeno installs Deno from scratch.
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

// detectRuntime finds the first available JS runtime that meets the minimum
// version requirement. Preference order: deno > bun > node.
// Returns (runtimeName, meetsMinimum, meetsRecommended).
func detectRuntime() (string, bool, bool) {
	for _, runtime := range []string{"deno", "bun", "node"} {
		if _, err := exec.LookPath(runtime); err != nil {
			continue
		}

		ok, recommended, err := checkRuntimeVersion(runtime)
		if err != nil {
			fmt.Printf("⚠  Could not verify %s version: %v\n", runtime, err)
			// Binary exists but version unreadable — accept with a warning.
			return runtime, true, false
		}
		if !ok {
			// Too old — keep looking.
			continue
		}

		return runtime, true, recommended
	}

	return "", false, false
}
