<script lang="ts">
import { flip } from 'svelte/animate'
import {
  active,
  completed,
  entries,
  label,
  playlistPrefix,
  totalProgress,
  ui,
} from '@/lib/store.svelte'

const activeList = $derived(active())
const overall = $derived(totalProgress())
const completedCount = $derived(completed().length)
const total = $derived(entries().length)
const running = $derived(ui.state?.isRunning ?? false)
const queuedCount = $derived(entries().filter((e) => e.status === 'queued').length)
</script>

<h1 class="title">Active Downloads</h1>

{#if activeList.length === 0}
  {#if queuedCount > 0}
    <p class="dim">{queuedCount} waiting in queue — Ctrl+Enter to start</p>
  {:else}
    <p class="dim">Nothing downloading. Add URLs in the Input/Queue tab.</p>
  {/if}
{:else}
  <ul class="list grid gap-3.5">
    {#each activeList as entry (entry.id)}
      <li class="item" animate:flip={{ duration: 200 }}>
        <p class="m-0 mb-1 overflow-hidden text-ellipsis whitespace-nowrap">{playlistPrefix(entry)}{label(entry)}</p>
        <div class="bar inline-block"><div class="fill" style={`width:${Math.min(100, entry.progress)}%`}></div></div>
        <span class="pct">{entry.progress.toFixed(1)}%</span>
      </li>
    {/each}
  </ul>
{/if}

<section class="mt-6 border-t border-border pt-2.5" class:running={running}>
  <p class="section">Overall Progress:</p>
  <div class="flex items-center">
    <div class="bar flex-1"><div class="fill" style={`width:${Math.min(100, overall)}%`}></div></div>
    <span class="pct">{overall.toFixed(1)}%</span>
  </div>
  <p class="dim">Completed: {completedCount}/{total}</p>
  {#if running}
    <p class="dim">downloading…</p>
  {/if}
</section>

