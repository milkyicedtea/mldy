<script lang="ts">
import { completed, label, listRows, removeEntry, toggleExpand, ui } from '@/lib/store.svelte'

const rows = $derived(listRows(completed(), ui.expanded))

function selectRow(i: number): void {
  ui.historyCursor = i
}
</script>

<h1 class="title">Download History</h1>

{#if rows.length === 0}
  <p class="dim">No completed downloads</p>
{:else}
  <ul class="list" role="listbox" aria-label="History">
    {#each rows as row, i (row.kind === 'header' ? row.batchId : row.entry.id)}
      {#if row.kind === 'header'}
        <li class="row header" class:selected={i === ui.historyCursor}>
          <button class="row-main playlist" onclick={() => { selectRow(i); toggleExpand(row.batchId); }}>
            <span class="arrow">{ui.expanded.has(row.batchId) ? '▼' : '▶'}</span>
            {row.title}
            {#if i === ui.historyCursor}<span class="dim">[Enter: Expand/Collapse]</span>{/if}
          </button>
        </li>
      {:else}
        <li class="row" class:selected={i === ui.historyCursor}>
          <div class="entry">
            <button class="row-main" class:playlist-item={!!row.entry.playlist} onclick={() => selectRow(i)}>
              <span class="icon" class:ok={row.entry.status === 'completed'} class:fail={row.entry.status === 'failed'}>
                {row.entry.status === 'failed' ? '✗' : '✓'}
              </span>
              {label(row.entry)}
            </button>
            <span class="actions">
              <button class="btn remove" title="Remove" onclick={() => removeEntry(row.entry.id)}>✕</button>
            </span>
          </div>
          {#if row.entry.status === 'failed' && row.entry.error}
            <pre class="error">{row.entry.error}</pre>
          {:else if row.entry.outputPath}
            <p class="dim saved">Saved to: {row.entry.outputPath}</p>
          {/if}
        </li>
      {/if}
    {/each}
  </ul>
{/if}

<style>
  .title {
    font-size: 1.05rem;
    color: var(--accent);
    margin: 0 0 10px;
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .row {
    padding: 1px 0;
  }
  .row.selected .row-main {
    color: var(--select);
    font-weight: 700;
  }
  .entry {
    display: flex;
    align-items: center;
    gap: 4px;
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
  .arrow {
    display: inline-block;
    width: 1.2em;
  }
  .icon.ok { color: var(--green); }
  .icon.fail { color: var(--red); }
  .btn {
    background: none;
    border: none;
    color: var(--red);
    font: inherit;
    cursor: pointer;
    padding: 2px 6px;
    border-radius: 4px;
  }
  .btn:hover { background: var(--hover); }
  .error {
    margin: 2px 0 6px 34px;
    color: var(--orange);
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 0.85em;
  }
  .saved {
    margin: 2px 0 6px 34px;
  }
</style>
