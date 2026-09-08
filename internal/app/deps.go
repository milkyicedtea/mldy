package app

import (
	"errors"
	"io"
	"os"
	"os/exec"
	rt "runtime"
	"strings"
	"sync/atomic"

	"github.com/wailsapp/wails/v3/pkg/application"

	"mldy/internal/deps"
)

// DepsStatus is the startup dependency report the frontend acts on: missing
// installs, runtime upgrades, and pending yt-dlp updates.
type DepsStatus struct {
	YtDlpInstalled            bool             `json:"ytDlpInstalled"`
	FfmpegInstalled           bool             `json:"ffmpegInstalled"`
	Runtime                   string           `json:"runtime"`
	RuntimeFound              bool             `json:"runtimeFound"`
	RuntimeVersion            string           `json:"runtimeVersion,omitempty"`
	RuntimeRecommendedVersion string           `json:"runtimeRecommendedVersion,omitempty"`
	RuntimeRecommended        bool             `json:"runtimeRecommended"`
	Update                    *deps.UpdateInfo `json:"update,omitempty"`
	UpdateAvailable           bool             `json:"updateAvailable"`
}

// DepsService exposes dependency detection, installation and updates.
// Install/update output is streamed to the frontend as "deps:log" events.
type DepsService struct {
	busy atomic.Bool
}

func NewDepsService() *DepsService {
	return &DepsService{}
}

// Check reports the current dependency state. Update checking is throttled
// internally to once per day.
func (d *DepsService) Check() DepsStatus {
	deps.SetOutput(io.Discard)
	defer deps.SetOutput(io.Discard)

	status := DepsStatus{}
	_, err := exec.LookPath("yt-dlp")
	status.YtDlpInstalled = err == nil
	_, err = exec.LookPath("ffmpeg")
	status.FfmpegInstalled = err == nil

	runtime, st, found := deps.DetectRuntime()
	status.Runtime = runtime
	status.RuntimeFound = found
	status.RuntimeVersion = st.Version
	status.RuntimeRecommendedVersion = st.RecommendedVersion
	status.RuntimeRecommended = st.MeetsRecommended

	if info, ok := deps.CheckUpdateAvailable(); ok {
		status.Update = &info
		status.UpdateAvailable = true
	}
	return status
}

// eventWriter forwards each written line to the frontend as a "deps:log" event.
type eventWriter struct {
	app *application.App
}

func (w eventWriter) Write(p []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		if line != "" {
			w.app.Event.Emit("deps:log", line)
		}
	}
	return len(p), nil
}

// streamOutput installs the event-streaming writer and returns a restore func.
func (d *DepsService) streamOutput() (restore func()) {
	deps.SetOutput(eventWriter{app: application.Get()})
	return func() { deps.SetOutput(io.Discard) }
}

// Install installs a dependency: "yt-dlp", "ffmpeg" or "deno".
func (d *DepsService) Install(name string) error {
	if !d.busy.CompareAndSwap(false, true) {
		return errors.New("another dependency operation is already running")
	}
	defer d.busy.Store(false)
	defer d.streamOutput()()

	var err error
	switch name {
	case "yt-dlp":
		err = deps.InstallYtDlp()
	case "ffmpeg":
		err = deps.InstallFfmpeg()
	case "deno":
		err = deps.InstallDeno()
	default:
		err = errors.New("unknown dependency: " + name)
	}
	if err != nil {
		return err
	}

	// On Windows the PATH isn't refreshed in the current process after an
	// installation, so we relaunch (same behaviour as the TUI).
	if name == "deno" && rt.GOOS == "windows" {
		restartSelf()
	}
	return nil
}

// Update updates a dependency: "yt-dlp", "ffmpeg" or a JS runtime name
// ("deno", "bun", "node").
func (d *DepsService) Update(name string) error {
	if !d.busy.CompareAndSwap(false, true) {
		return errors.New("another dependency operation is already running")
	}
	defer d.busy.Store(false)
	defer d.streamOutput()()

	var err error
	switch name {
	case "yt-dlp":
		err = deps.UpdateYtDlp()
	case "ffmpeg":
		err = deps.UpdateFfmpeg()
	case "deno", "bun", "node":
		err = deps.UpdateRuntime(name)
	default:
		err = errors.New("unknown dependency: " + name)
	}
	if err != nil {
		return err
	}

	if name == "deno" || name == "bun" || name == "node" {
		if rt.GOOS == "windows" {
			restartSelf()
		}
	}
	return nil
}

// restartSelf re-executes the current binary with the same arguments and
// exits. Used on Windows after installs/updates so the new PATH is picked up.
func restartSelf() {
	self, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(self, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	os.Exit(0)
}

// Quit exits the application. Used when the user declines a mandatory
// dependency install — the app cannot function without yt-dlp.
func (d *DepsService) Quit() {
	if a := application.Get(); a != nil {
		a.Quit()
	}
}
