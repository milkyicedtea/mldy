<script lang="ts">
import { fade } from 'svelte/transition'
import { completed, label, listRows, removeEntry, toggleExpand, ui } from '@/lib/store.svelte'

const rows = $derived(listRows(completed(), ui.expanded))

function selectRow(i: number): void {
  ui.historyCursor = i
}
</script>

<h1 class="title">Download History</h1>

{#if rows.length === 0}
  <p class="dim">No completed downloads yet — downloads land here once they finish.</p>
{:else}
  <ul class="list" role="listbox" aria-label="History">
    {#each rows as row, i (row.kind === 'header' ? row.batchId : row.entry.id)}
      {#if row.kind === 'header'}
        <li class="row header" class:selected={i === ui.historyCursor} transition:fade={{ duration: 150 }}>
          <button class="row-main playlist" onclick={() => { selectRow(i); toggleExpand(row.batchId); }}>
            <span class="arrow">{ui.expanded.has(row.batchId) ? '▼' : '▶'}</span>
            {row.title}
          </button>
        </li>
      {:else}
        <li class="row" class:selected={i === ui.historyCursor} transition:fade={{ duration: 150 }}>
          <div class="entry">
            <button class="row-main" class:playlist-item={!!row.entry.playlist} onclick={() => selectRow(i)}>
              <span class={row.entry.status === 'completed' ? 'text-green' : row.entry.status === 'failed' ? 'text-red' : ''}>
                {row.entry.status === 'failed' ? '✗' : '✓'}
              </span>
              {label(row.entry)}
            </button>
            <span class="actions">
              <button class="btn remove" title="Remove" onclick={() => removeEntry(row.entry.id)}>✕</button>
            </span>
          </div>
          {#if row.entry.status === 'failed' && row.entry.error}
            <pre class="mt-0.5 mb-1.5 ml-[34px] text-orange text-[0.85em] whitespace-pre-wrap break-words">{row.entry.error}</pre>
          {:else if row.entry.outputPath}
            <p class="dim mt-0.5 mb-1.5 ml-[34px]">Saved to: {row.entry.outputPath}</p>
          {/if}
        </li>
      {/if}
    {/each}
  </ul>
{/if}

