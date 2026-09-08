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
  <ul class="list">
    {#each activeList as entry (entry.id)}
      <li class="item" animate:flip={{ duration: 200 }}>
        <p class="name">{playlistPrefix(entry)}{label(entry)}</p>
        <div class="bar"><div class="fill" style={`width:${Math.min(100, entry.progress)}%`}></div></div>
        <span class="pct">{entry.progress.toFixed(1)}%</span>
      </li>
    {/each}
  </ul>
{/if}

<section class="overall" class:running={running}>
  <p class="section">Overall Progress:</p>
  <div class="overall-row">
    <div class="bar"><div class="fill" style={`width:${Math.min(100, overall)}%`}></div></div>
    <span class="pct">{overall.toFixed(1)}%</span>
  </div>
  <p class="dim">Completed: {completedCount}/{total}</p>
  {#if running}
    <p class="dim">downloading…</p>
  {/if}
</section>

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
    display: grid;
    gap: 14px;
  }
  .name {
    margin: 0 0 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .item .bar {
    display: inline-block;
  }
  .bar {
    width: calc(100% - 5rem);
    height: 10px;
    background: var(--panel);
    border-radius: 5px;
    overflow: hidden;
    vertical-align: middle;
  }
  .fill {
    height: 100%;
    background: var(--green);
    transition: width 0.2s ease;
  }
  @keyframes pulse {
    from {
      opacity: 0.65;
    }
    to {
      opacity: 1;
    }
  }
  .item .fill,
  .overall.running .fill {
    animation: pulse 1.2s ease-in-out infinite alternate;
  }
  .pct {
    margin-left: 8px;
    color: var(--dim);
    font-variant-numeric: tabular-nums;
  }
  .overall {
    margin-top: 24px;
    border-top: 1px solid var(--border);
    padding-top: 10px;
  }
  .section {
    font-weight: 700;
    margin: 0 0 6px;
  }
  .overall-row {
    display: flex;
    align-items: center;
  }
  .overall-row .bar {
    flex: 1;
  }
</style>
