import type { DepsStatus, Entry, State } from '@bindings/mldy/internal/app'
import * as depsSvc from '@bindings/mldy/internal/app/depsservice'
import * as svc from '@bindings/mldy/internal/app/service'
import { Events } from '@wailsio/runtime'

export type Screen = 'input' | 'downloads' | 'history'

export type Row =
  | { kind: 'header'; batchId: string; title: string; count: number }
  | { kind: 'item'; entry: Entry }

export const ui = $state({
  screen: 'input' as Screen,
  state: null as State | null,
  queueCursor: 0,
  historyCursor: 0,
  expanded: new Set<string>(),
  url: '',
  inputFocused: true,
})

export const deps = $state({
  status: null as DepsStatus | null,
  phase: 'checking' as 'checking' | 'prompt' | 'running' | 'guide' | 'done',
  log: [] as string[],
  failedOp: null as { title: string; error: string } | null,
})

let started = false

export function initStore(): void {
  if (started) return
  started = true

  svc.GetState().then((s) => {
    ui.state = s
  })
  Events.On('state', (ev: { data: unknown }) => {
    ui.state = ev.data as State
    clampCursors()
  })
  Events.On('deps:log', (ev: { data: unknown }) => {
    deps.log.push(String(ev.data))
  })
}

function clampCursors(): void {
  if (!ui.state) return
  const q = listRows(queued(), ui.expanded).length
  if (ui.queueCursor >= q) ui.queueCursor = Math.max(0, q - 1)
  const h = listRows(completed(), ui.expanded).length
  if (ui.historyCursor >= h) ui.historyCursor = Math.max(0, h - 1)
}

export function entries(): Entry[] {
  return ui.state?.entries ?? []
}

export function queued(): Entry[] {
  return entries().filter((e) => e.status === 'queued')
}

export function active(): Entry[] {
  return entries().filter((e) => e.status === 'downloading')
}

export function completed(): Entry[] {
  return entries().filter((e) => e.status === 'completed' || e.status === 'failed')
}

export function totalProgress(): number {
  const all = entries()
  if (all.length === 0) return 0
  let total = 0
  for (const e of all) {
    if (e.status === 'completed') total += 100
    else if (e.status === 'downloading') total += e.progress
  }
  return total / all.length
}

export function label(e: Entry): string {
  return e.title || e.url
}

export function playlistPrefix(e: Entry): string {
  if (!e.playlist) return ''
  return `[${e.playlist.PlaylistTitle} ${e.playlist.Index}/${e.playlist.Total}] `
}

// Group entries into playlist headers + items, matching the TUI's list layout.
export function listRows(list: Entry[], expanded: Set<string>): Row[] {
  const rows: Row[] = []
  let lastBatch = ''
  for (const e of list) {
    const bid = e.playlist?.BatchID ?? ''
    if (bid) {
      if (bid !== lastBatch) {
        rows.push({
          kind: 'header',
          batchId: bid,
          title: e.playlist!.PlaylistTitle,
          count: e.playlist!.Total,
        })
        lastBatch = bid
      }
      if (!expanded.has(bid)) continue
    } else {
      lastBatch = ''
    }
    rows.push({ kind: 'item', entry: e })
  }
  return rows
}

// ---- actions ------------------------------------------------------------------

export function setScreen(screen: Screen): void {
  ui.screen = screen
}

export function toggleExpand(batchId: string): void {
  if (ui.expanded.has(batchId)) ui.expanded.delete(batchId)
  else ui.expanded.add(batchId)
}

export function addURL(): void {
  const url = ui.url.trim()
  if (!url) return
  ui.url = ''
  svc.AddURL(url)
}

export function start(): void {
  svc.Start()
}

export function removeEntry(id: number): void {
  svc.RemoveEntry(id)
}

export function removeBatch(batchId: string): void {
  svc.RemoveBatch(batchId)
}

export function removeLast(): void {
  svc.RemoveLast()
}

export function clearAll(): void {
  svc.ClearAll()
}

export function moveCursor(delta: number, screen: 'input' | 'history'): void {
  const list = screen === 'input' ? queued() : completed()
  const rows = listRows(list, ui.expanded)
  const max = Math.max(0, rows.length - 1)
  if (screen === 'input') {
    ui.queueCursor = Math.min(max, Math.max(0, ui.queueCursor + delta))
  } else {
    ui.historyCursor = Math.min(max, Math.max(0, ui.historyCursor + delta))
  }
}

// activateSelected handles Enter/Space on the cursor row, mirroring the TUI:
// Enter toggles a playlist header; Space on an item starts the queue.
export function activateSelected(screen: 'input' | 'history', starting: boolean): void {
  const cursor = screen === 'input' ? ui.queueCursor : ui.historyCursor
  const rows = listRows(screen === 'input' ? queued() : completed(), ui.expanded)
  const row = rows[cursor]
  if (!row) return
  if (row.kind === 'header') {
    if (!starting) toggleExpand(row.batchId)
  } else if (starting) {
    start()
  }
}

// deleteSelected handles d/Delete/Backspace on the cursor row: a header
// removes the whole batch, an item removes just that entry.
export function deleteSelected(screen: 'input' | 'history'): void {
  const cursor = screen === 'input' ? ui.queueCursor : ui.historyCursor
  const rows = listRows(screen === 'input' ? queued() : completed(), ui.expanded)
  const row = rows[cursor]
  if (!row) return
  if (row.kind === 'header') removeBatch(row.batchId)
  else removeEntry(row.entry.id)
}

// ---- deps modal flow ------------------------------------------------------------

interface Prompt {
  kind: 'install' | 'update'
  dep: string
  title: string
  body: string
  confirmLabel: string
  optionalLabel: string | null
  mandatory: boolean
}

function buildPrompt(status: DepsStatus): Prompt | null {
  if (!status.ytDlpInstalled) {
    return {
      kind: 'install',
      dep: 'yt-dlp',
      title: 'yt-dlp not found',
      body: 'mldy needs yt-dlp to download videos. Install it now?',
      confirmLabel: 'Install yt-dlp',
      optionalLabel: null,
      mandatory: true,
    }
  }
  if (!status.ffmpegInstalled) {
    return {
      kind: 'install',
      dep: 'ffmpeg',
      title: 'ffmpeg not found',
      body: 'ffmpeg is required for format conversion and merging. Install it now?',
      confirmLabel: 'Install ffmpeg',
      optionalLabel: null,
      mandatory: true,
    }
  }
  if (!status.runtimeFound) {
    return {
      kind: 'install',
      dep: 'deno',
      title: 'No JavaScript runtime found',
      body: 'Some videos need a JS runtime (Deno ≥2, Bun ≥1.0.31, or Node ≥20). Install Deno now (recommended)?',
      confirmLabel: 'Install Deno',
      optionalLabel: 'Continue without',
      mandatory: false,
    }
  }
  if (!status.runtimeRecommended) {
    return {
      kind: 'update',
      dep: status.runtime,
      title: `${status.runtime} upgrade recommended`,
      body: `Your ${status.runtime} version is below the recommended threshold. Upgrade it?`,
      confirmLabel: `Upgrade ${status.runtime}`,
      optionalLabel: 'Skip',
      mandatory: false,
    }
  }
  if (status.updateAvailable && status.update) {
    return {
      kind: 'update',
      dep: 'yt-dlp',
      title: `yt-dlp ${status.update.Latest} available`,
      body: `Installed: ${status.update.Installed}. Update yt-dlp now?`,
      confirmLabel: 'Update yt-dlp',
      optionalLabel: 'Skip',
      mandatory: false,
    }
  }
  return null
}

export async function runDepsCheck(): Promise<void> {
  deps.phase = 'checking'
  deps.log = []
  deps.failedOp = null
  deps.status = await depsSvc.Check()
  nextPrompt()
}

function nextPrompt(): void {
  const p = buildPrompt(deps.status!)
  if (p) {
    deps.phase = 'prompt'
    currentPrompt = p
  } else {
    deps.phase = 'done'
  }
}

let currentPrompt: Prompt | null = null

export function activePrompt(): Prompt | null {
  return currentPrompt
}

export async function confirmPrompt(): Promise<void> {
  const p = currentPrompt
  if (!p) return
  deps.phase = 'running'
  deps.log = []
  try {
    if (p.kind === 'install') await depsSvc.Install(p.dep)
    else await depsSvc.Update(p.dep)
    // After a successful yt-dlp update, offer the ffmpeg piggyback update
    // exactly like the TUI did.
    if (p.kind === 'update' && p.dep === 'yt-dlp' && deps.status?.ffmpegInstalled) {
      currentPrompt = {
        kind: 'update',
        dep: 'ffmpeg',
        title: 'Also update ffmpeg?',
        body: 'ffmpeg has no reliable version probe; it can be refreshed opportunistically.',
        confirmLabel: 'Update ffmpeg',
        optionalLabel: 'Skip',
        mandatory: false,
      }
      deps.phase = 'prompt'
      return
    }
    await runDepsCheck()
  } catch (err) {
    deps.failedOp = { title: `${p.title} — failed`, error: String(err) }
    deps.phase = 'guide'
  }
}

export function declinePrompt(): void {
  const p = currentPrompt
  if (!p) return
  if (p.mandatory) {
    deps.failedOp = null
    deps.phase = 'guide'
    return
  }
  // Non-mandatory: yt-dlp updates are internally throttled per day; runtime
  // nudges just get skipped for this session.
  nextPrompt()
}

export function quitApp(): void {
  depsSvc.Quit()
}

export const GUIDES: Record<string, string[]> = {
  'yt-dlp': [
    'Manual yt-dlp installation:',
    'macOS:   brew install yt-dlp',
    "Linux:   check your distro's package manager, or use github yt-dlp releases",
    'Windows: winget install yt-dlp',
    'Or: https://github.com/yt-dlp/yt-dlp',
  ],
  ffmpeg: [
    'Manual ffmpeg installation:',
    'macOS:   brew install ffmpeg',
    "Linux:   check your distro's package manager",
    'Windows: winget install -e --id Gyan.FFmpeg --source winget',
    'Or: https://ffmpeg.org/download.html',
  ],
  deno: [
    'Manual Deno installation:',
    'macOS/Linux: curl -fsSL https://deno.land/install.sh | sh',
    'Windows:     winget install DenoLand.Deno',
    'Or: https://deno.land/',
  ],
}
