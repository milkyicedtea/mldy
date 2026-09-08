<script lang="ts">
import {
  addURL,
  label,
  listRows,
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
</script>

<h1 class="title">Add URLs to Queue</h1>

<input
  class="url-input"
  type="text"
  placeholder="Paste a video or playlist URL, then press Enter"
  bind:value={ui.url}
  onkeydown={onKeydown}
  onfocus={() => (ui.inputFocused = true)}
/>

{#if resolving > 0}
  <p class="dim resolving">⟳ Resolving {resolving} URL(s)…</p>
{/if}

{#if rows.length === 0 && resolving === 0}
  <p class="dim">No items in queue</p>
{:else}
  <p class="section">Queued ({queued().length}):</p>
  <ul class="list" role="listbox" aria-label="Queue">
    {#each rows as row, i (row.kind === 'header' ? row.batchId : row.entry.id)}
      {#if row.kind === 'header'}
        <li class="row header" class:selected={i === ui.queueCursor && !ui.inputFocused}>
          <button class="row-main playlist" onclick={() => { selectRow(i); toggleExpand(row.batchId); }}>
            <span class="arrow">{ui.expanded.has(row.batchId) ? '▼' : '▶'}</span>
            {row.title}
            <span class="dim">({row.count})</span>
          </button>
          <span class="actions">
            <button class="btn start" title="Start downloads" onclick={start}>▶</button>
            <button class="btn remove" title="Remove playlist" onclick={() => removeBatch(row.batchId)}>✕</button>
          </span>
        </li>
      {:else}
        <li class="row" class:selected={i === ui.queueCursor && !ui.inputFocused}>
          <button class="row-main" class:playlist-item={!!row.entry.playlist}
                  onclick={() => selectRow(i)}>
            {#if row.entry.playlist}<span class="idx">{row.entry.playlist.Index}/{row.entry.playlist.Total}</span>{/if}
            <span class="label">{label(row.entry)}</span>
          </button>
          <span class="actions">
            <button class="btn start" title="Start downloads" onclick={start}>▶</button>
            <button class="btn remove" title="Remove" onclick={() => removeEntry(row.entry.id)}>✕</button>
          </span>
        </li>
      {/if}
    {/each}
  </ul>
{/if}

<p class="hint dim">
  {#if !isRunning && queued().length > 0}
    Ctrl+Enter start · Backspace removes last ·
  {/if}
  Alt+R clears the queue
  <button class="linkish" onclick={removeLast}>remove last</button>
</p>

{#if config}
  <section class="config">
    <p class="section">Current Config:</p>
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

<style>
  .title {
    font-size: 1.05rem;
    color: var(--accent);
    margin: 0 0 10px;
  }
  .url-input {
    width: 100%;
    box-sizing: border-box;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--fg);
    font: inherit;
    padding: 8px 10px;
    outline: none;
  }
  .url-input:focus {
    border-color: var(--select);
  }
  .resolving {
    margin: 8px 0;
  }
  .section {
    font-weight: 700;
    margin: 14px 0 6px;
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 1px 0;
  }
  .row.selected .row-main,
  .row.selected .idx {
    color: var(--select);
    font-weight: 700;
  }
  .row-main {
    background: none;
    border: none;
    color: var(--fg);
    font: inherit;
    text-align: left;
    padding: 3px 6px;
    cursor: pointer;
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .header .row-main {
    color: var(--blue);
    font-weight: 700;
  }
  .row.selected .row-main.playlist {
    color: var(--select);
  }
  .playlist-item {
    padding-left: 26px;
  }
  .idx {
    color: var(--dim);
    margin-right: 8px;
  }
  .arrow {
    display: inline-block;
    width: 1.2em;
  }
  .label {
    color: inherit;
  }
  .row-main:not(.playlist) .label {
    color: var(--fg);
  }
  .actions {
    display: flex;
    gap: 2px;
  }
  .btn {
    background: none;
    border: none;
    font: inherit;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
  }
  .btn.start { color: var(--green); }
  .btn.remove { color: var(--red); }
  .btn:hover { background: var(--hover); }
  .hint {
    margin-top: 10px;
  }
  .linkish {
    background: none;
    border: none;
    color: var(--dim);
    font: inherit;
    text-decoration: underline;
    cursor: pointer;
    padding: 0;
  }
  .config {
    margin-top: 16px;
    border-top: 1px solid var(--border);
    padding-top: 8px;
  }
  .config dl {
    margin: 0;
    display: grid;
    gap: 2px;
  }
  .config dl div {
    display: flex;
  }
  .config dt {
    color: var(--dim);
    width: 10rem;
    flex-shrink: 0;
  }
  .config dd {
    margin: 0;
    word-break: break-all;
  }
</style>
