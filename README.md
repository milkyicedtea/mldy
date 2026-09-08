# mldy

A desktop GUI for downloading videos using yt-dlp, built with Wails v3 and Svelte 5.

## Features

- Native desktop app (Wails v3 + Svelte 5 frontend)
- Add single videos or entire playlists; playlists are grouped and expandable
- Download queue with per-item and overall progress tracking
- Download history with success/failure status, error details, and output paths
- Automatic dependency management: yt-dlp, ffmpeg, and a JavaScript runtime
  (Deno, Bun, or Node.js) are detected, installed, and updated from the app
- Cross-platform support (Windows, Linux, macOS)

## Screenshots

<table>
  <tr>
    <td><img src="screenshots/input-queue-empty.png" alt="Input/Queue Tab Empty"/>
    <td><img src="screenshots/input-queue-full.png" alt="Input/Queue Tab Full"/>
  </tr>
  <tr>
    <td><img src="screenshots/downloads.png" alt="Downloads Tab"/>
    <td><img src="screenshots/history.png" alt="History Tab"/>
  </tr>
</table>

Note: the screenshots above were taken with the old terminal UI; the layout of
the GUI version differs.

## Requirements

- Go 1.26+
- Node/npm or Bun (frontend build)
- Platform webview runtime:
  - Linux: GTK4 + WebKitGTK 6.0 (`libgtk-4-dev`, `libwebkitgtk-6.0-dev` to build)
  - Windows: WebView2 (preinstalled on Windows 10/11)
  - macOS: preinstalled
- At runtime: yt-dlp, ffmpeg, and a JavaScript runtime (Deno ≥2, Bun ≥1.0.31,
  or Node ≥20) — the app detects and offers to install/update these itself

## Development

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.16
wails3 dev
```

## Building

```bash
wails3 build        # native build -> bin/mldy
./bin/mldy
```

Cross-compile for Windows:

```bash
./crossbuild-windows.sh   # -> bin/mldy.exe
```

## Usage

Add URLs in the Input/Queue tab (Enter to add). Start downloads with
Ctrl+Enter, the ▶ button, or `s` on a focused list row. Alt+R clears the
queue. Tab switching: click the tabs or press 1/2/3. Configuration lives in
`~/.config/mldy/config.yaml` and is shown at the bottom of the Input/Queue tab.
