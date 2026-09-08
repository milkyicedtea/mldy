package deps

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
	// recommended is the soft floor above which the installation is considered
	// ideal. Pinned to concrete milestones (deno 2.9, bun 1.4, node 24 LTS) so
	// the upgrade nudge only appears when the user actually falls behind.
	recommended      [3]int
	recommendedLabel string
}

var runtimeVersions = map[string]runtimeVersionInfo{
	"deno": {
		minimum:          [3]int{2, 0, 0},
		recommended:      [3]int{2, 9, 0},
		recommendedLabel: "2.9 or newer",
	},
	"node": {
		// 24 LTS is recommended; 20 is the absolute floor.
		minimum:          [3]int{20, 0, 0},
		recommended:      [3]int{24, 0, 0},
		recommendedLabel: "24 LTS",
	},
	"bun": {
		minimum:          [3]int{1, 0, 31},
		recommended:      [3]int{1, 4, 0},
		recommendedLabel: "1.4 or newer",
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

// RuntimeStatus describes a detected runtime: the version found on disk and
// what the app considers ideal for it.
type RuntimeStatus struct {
	Version            string
	RecommendedVersion string
	MeetsMinimum       bool
	MeetsRecommended   bool
}

// checkRuntimeVersion verifies the installed version meets the minimum and
// recommended thresholds.
// Returns (status, error); error means the version could not be determined.
func checkRuntimeVersion(runtime string) (RuntimeStatus, error) {
	info, ok := runtimeVersions[runtime]
	if !ok {
		return RuntimeStatus{}, fmt.Errorf("unknown runtime: %s", runtime)
	}

	versionStr, err := getRuntimeVersion(runtime)
	if err != nil {
		return RuntimeStatus{}, err
	}

	parsed, err := parseVersion(versionStr)
	if err != nil {
		return RuntimeStatus{}, fmt.Errorf("could not parse %s version %q: %w", runtime, versionStr, err)
	}

	minStr := fmt.Sprintf("%d.%d.%d", info.minimum[0], info.minimum[1], info.minimum[2])

	if !versionAtLeast(parsed, info.minimum) {
		printf("⚠  %s %s is below the minimum required %s (recommended: %s).\n",
			runtime, versionStr, minStr, info.recommendedLabel)
		return RuntimeStatus{
			Version:            versionStr,
			RecommendedVersion: info.recommendedLabel,
			MeetsMinimum:       false,
			MeetsRecommended:   false,
		}, nil
	}

	meetsRecommended := versionAtLeast(parsed, info.recommended)
	if meetsRecommended {
		printf("✓  %s %s detected.\n", runtime, versionStr)
	} else {
		printf("✓  %s %s detected (upgrade to %s recommended).\n",
			runtime, versionStr, info.recommendedLabel)
	}

	return RuntimeStatus{
		Version:            versionStr,
		RecommendedVersion: info.recommendedLabel,
		MeetsMinimum:       true,
		MeetsRecommended:   meetsRecommended,
	}, nil
}

// UpdateRuntime updateRuntime attempts to upgrade the given runtime to its latest version.
// Returns true if the update succeeded and the program should restart on Windows.
func UpdateRuntime(runtime string) error {
	printf("Updating %s...\n", runtime)
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

// InstallDeno installDeno installs Deno from scratch.
func InstallDeno() error {
	switch rt.GOOS {
	case "darwin", "linux":
		return run("sh", "-c", "curl -fsSL https://deno.land/install.sh | sh")
	case "windows":
		return run("winget", "install", "DenoLand.Deno")
	default:
		return errors.New("unsupported OS")
	}
}

func PrintDenoGuide() {
	printf("\nManual Deno installation:")
	printf("macOS/Linux: curl -fsSL https://deno.land/install.sh | sh")
	printf("Windows:     winget install DenoLand.Deno")
	printf("Or: https://deno.land/")
}

// DetectRuntime finds the first available JS runtime that meets the minimum
// version requirement. Preference order: deno > bun > node.
// Returns (runtimeName, status, found).
func DetectRuntime() (string, RuntimeStatus, bool) {
	for _, runtime := range []string{"deno", "bun", "node"} {
		if _, err := exec.LookPath(runtime); err != nil {
			continue
		}

		st, err := checkRuntimeVersion(runtime)
		if err != nil {
			printf("⚠  Could not verify %s version: %v\n", runtime, err)
			// Binary exists but version unreadable — accept with a warning.
			return runtime, RuntimeStatus{}, true
		}
		if !st.MeetsMinimum {
			// Too old — keep looking.
			continue
		}

		return runtime, st, true
	}

	return "", RuntimeStatus{}, false
}

// RuntimeAvailable reports whether the named JS runtime is installed and
// meets the minimum version requirement.
func RuntimeAvailable(name string) bool {
	st, err := checkRuntimeVersion(name)
	return err == nil && st.MeetsMinimum
}
