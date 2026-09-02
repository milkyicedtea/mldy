package deps

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	rt "runtime"
	"strings"
	"time"
)

const (
	ytdlpRepo   = "yt-dlp/yt-dlp"
	checkEvery  = 24 * time.Hour
	stampSubdir = "mldy"
	stampFile   = "last_update_check"
)

// latestTag fetches the latest release tag of a GitHub repo ("v" prefix stripped).
func latestTag(repo string) (string, error) {
	req, err := http.NewRequest(http.MethodGet,
		"https://api.github.com/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API: HTTP %d", resp.StatusCode)
	}

	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	return strings.TrimPrefix(body.TagName, "v"), nil
}

// installedYtDlpVersion returns what yt-dlp itself reports. Its release tags
// and `--version` output are identical date strings, so equality is exact.
func installedYtDlpVersion() (string, error) {
	out, err := exec.Command("yt-dlp", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func sudoPrefix() []string {
	if rt.GOOS != "windows" && os.Geteuid() != 0 {
		return []string{"sudo"}
	}
	return nil
}

// upgradeDistroPackage upgrades packages via the distro package manager.
func upgradeDistroPackage(pkgs ...string) error {
	prefix := sudoPrefix()
	id, idLike, err := detectLinuxDistro()
	if err != nil {
		return err
	}
	switch {
	case id == "arch" || strings.Contains(idLike, "arch"):
		return runPackageManager(prefix, "pacman", append([]string{"-Syu", "--noconfirm"}, pkgs...)...)
	case id == "debian" || id == "ubuntu" || strings.Contains(idLike, "debian"):
		return runPackageManager(prefix, "apt", append([]string{"install", "-y"}, pkgs...)...)
	case id == "fedora" || strings.Contains(idLike, "rhel") ||
		strings.Contains(idLike, "fedora") || strings.Contains(idLike, "centos"):
		return runPackageManager(prefix, "dnf", append([]string{"upgrade", "-y"}, pkgs...)...)
	case id == "opensuse" || strings.Contains(idLike, "suse") || strings.Contains(idLike, "opensuse"):
		return runPackageManager(prefix, "zypper", append([]string{"install", "-y"}, pkgs...)...)
	case id == "alpine":
		return runPackageManager(prefix, "apk", append([]string{"add"}, pkgs...)...)
	default:
		return fmt.Errorf("unsupported distro: %s (%s)", id, idLike)
	}
}

func updateYtDlp() error {
	switch rt.GOOS {
	case "windows":
		// Installed as a standalone exe build -> built-in self-updater works.
		return run("yt-dlp", "-U")
	case "darwin":
		return run("brew", "upgrade", "yt-dlp")
	case "linux":
		// Standalone GitHub-binary installs self-update; package-manager
		// installs exit non-zero with a hint, so fall back to the distro.
		if err := run("yt-dlp", "-U"); err == nil {
			return nil
		}
		return upgradeDistroPackage("yt-dlp")
	default:
		return errors.New("unsupported OS")
	}
}

func updateFfmpeg() error {
	switch rt.GOOS {
	case "windows":
		return run("winget", "upgrade", "-e", "--id", "Gyan.FFmpeg", "--source", "winget")
	case "darwin":
		return run("brew", "upgrade", "ffmpeg")
	case "linux":
		return upgradeDistroPackage("ffmpeg")
	default:
		return errors.New("unsupported OS")
	}
}

// stampPath returns the throttle-stamp file path in the user config dir.
func stampPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stampSubdir, stampFile), nil
}

// checkUpdates compares installed yt-dlp against the latest GitHub release and
// offers to update (ffmpeg piggybacks on the same update pass). Throttled to
// once per day via a stamp file; every failure path is non-fatal so startup is
// never blocked.
func CheckUpdates() {
	stamp, err := stampPath()
	if err != nil {
		return
	}
	if info, err := os.Stat(stamp); err == nil && time.Since(info.ModTime()) < checkEvery {
		return
	}

	installed, err := installedYtDlpVersion()
	if err != nil {
		return // yt-dlp absent/broken — the install path in main() handles that
	}
	latest, err := latestTag(ytdlpRepo)
	if err != nil {
		return // offline / rate-limited
	}
	_ = os.MkdirAll(filepath.Dir(stamp), 0o755)
	_ = os.WriteFile(stamp, nil, 0o644) // declining == snooze for 24h

	if latest == installed {
		return
	}

	fmt.Printf("yt-dlp %s available (installed: %s).\n", latest, installed)
	if !AskYesNo("Update yt-dlp now?") {
		return
	}
	if err := updateYtDlp(); err != nil {
		fmt.Println("yt-dlp update failed:", err)
		return
	}
	fmt.Println("yt-dlp updated.")

	// ffmpeg has no version probe worth trusting; refresh it opportunistically
	// while we are already shelling out. Failure is harmless.
	if AskYesNoDefaultNo("Also update ffmpeg?") {
		if err := updateFfmpeg(); err != nil {
			fmt.Println("ffmpeg update failed:", err)
		} else {
			fmt.Println("ffmpeg updated.")
		}
	}
}
