<script lang="ts">
import { fade } from 'svelte/transition'
import {
  addURL,
  label,
  listRows,
  openSettings,
  queued,
  removeBatch,
  removeEntry,
  removeLast,
  start,
  toggleExpand,
  ui,
} from '@/lib/store.svelte'

const rows = $derived(listRows(queued(), ui.expanded))
const resolving = $derived(ui.state?.resolving ?? 0)
const isRunning = $derived(ui.state?.isRunning ?? false)
const config = $derived(ui.state?.config)

function selectRow(i: number): void {
  ui.inputFocused = false
  ui.queueCursor = i
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter') {
    e.preventDefault()
    addURL()
  }
}

// A paste of multiple URLs queues them all directly; a single URL is left
// in the input for the user to confirm with Enter.
function onPaste(e: ClipboardEvent): void {
  const text = e.clipboardData?.getData('text') ?? ''
  if (text.split(/\s+/).filter(Boolean).length > 1) {
    e.preventDefault()
    addURL(text)
  }
}
</script>

<h1 class="title">Add URLs to Queue</h1>

<input
  class="w-full box-border rounded-md border border-border bg-panel px-2.5 py-2 font-[inherit] text-fg outline-none focus:border-select"
  type="text"
  placeholder="Paste a video or playlist URL, then press Enter"
  bind:value={ui.url}
  onkeydown={onKeydown}
  onpaste={onPaste}
  onfocus={() => (ui.inputFocused = true)}
/>

{#if resolving > 0}
  <p class="dim my-2">⟳ Resolving {resolving} URL(s)…</p>
{/if}

{#if rows.length === 0 && resolving === 0}
  <p class="dim">Queue is empty — paste a video or playlist URL above, then press Enter.</p>
{:else}
  <p class="section mt-3.5">Queued ({queued().length}):</p>
  <ul class="list" role="listbox" aria-label="Queue">
    {#each rows as row, i (row.kind === 'header' ? row.batchId : row.entry.id)}
      {#if row.kind === 'header'}
        <li class="row header" class:selected={i === ui.queueCursor && !ui.inputFocused} transition:fade={{ duration: 150 }}>
          <button class="row-main playlist" onclick={() => { selectRow(i); toggleExpand(row.batchId); }}>
            <span class="arrow">{ui.expanded.has(row.batchId) ? '▼' : '▶'}</span>
            {row.title}
            <span class="dim">({row.count})</span>
          </button>
          <span class="actions">
            <button class="btn start" title="Start downloads" disabled={isRunning} onclick={start}>▶</button>
            <button class="btn remove" title="Remove playlist" onclick={() => removeBatch(row.batchId)}>✕</button>
          </span>
        </li>
      {:else}
        <li class="row" class:selected={i === ui.queueCursor && !ui.inputFocused} transition:fade={{ duration: 150 }}>
          <button class="row-main" class:playlist-item={!!row.entry.playlist}
                  onclick={() => selectRow(i)}>
            {#if row.entry.playlist}<span class="idx">{row.entry.playlist.Index}/{row.entry.playlist.Total}</span>{/if}
            <span class="label">{label(row.entry)}</span>
          </button>
          <span class="actions">
            <button class="btn start" title="Start downloads" disabled={isRunning} onclick={start}>▶</button>
            <button class="btn remove" title="Remove" onclick={() => removeEntry(row.entry.id)}>✕</button>
          </span>
        </li>
      {/if}
    {/each}
  </ul>
{/if}

<p class="mt-2.5 dim">
  {#if !isRunning && queued().length > 0}
    Ctrl+Enter start · Backspace removes last ·
  {/if}
  Alt+R clears the queue
  <button class="linkish" onclick={removeLast}>remove last</button>
</p>

{#if config}
  <section class="config">
    <p class="section mt-3.5">Current Config: <button class="linkish" onclick={openSettings}>edit</button></p>
    <dl>
      <div><dt>Kind</dt><dd>{config.Kind}</dd></div>
      <div><dt>Format</dt><dd>{config.Format}</dd></div>
      <div><dt>Audio Quality</dt><dd>{config.AudioQuality}</dd></div>
      <div><dt>Video Quality</dt><dd>{config.VideoQuality}</dd></div>
      <div><dt>Output Folder</dt><dd>{config.OutputFolder}</dd></div>
      <div>
        <dt>JS Runtime</dt>
        <dd>{ui.state?.runtime || 'none (some videos may fail)'}</dd>
      </div>
    </dl>
  </section>
{/if}

