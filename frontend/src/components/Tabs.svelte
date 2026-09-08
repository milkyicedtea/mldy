<script lang="ts">
import { active, completed, openSettings, queued, setScreen, ui } from '@/lib/store.svelte'

const tabs = [
  { id: 'input', label: 'Input/Queue' },
  { id: 'downloads', label: 'Downloads' },
  { id: 'history', label: 'History' },
] as const

const counts = $derived.by(() => ({
  input: queued().length,
  downloads: active().length,
  history: completed().length,
}))
</script>

<nav class="tabs">
  <div class="tabs-left">
    {#each tabs as t (t.id)}
      <button
        class="tab"
        class:active={ui.screen === t.id}
        onclick={() => setScreen(t.id)}
      >
        {t.label}
        {#if counts[t.id] > 0}<span class="badge">{counts[t.id]}</span>{/if}
      </button>
    {/each}
  </div>
  <button class="gear" title="Settings" aria-label="Open settings" onclick={openSettings}>⚙</button>
</nav>

<style>
  .tabs {
    display: flex;
    align-items: center;
    padding: 8px 12px 0;
    border-bottom: 1px solid var(--border);
  }
  .tabs-left {
    display: flex;
    gap: 4px;
  }
  .gear {
    margin-left: auto;
    background: none;
    border: none;
    color: var(--dim);
    font: inherit;
    font-size: 1.05em;
    padding: 4px 8px;
    cursor: pointer;
    border-radius: 6px;
  }
  .gear:hover {
    color: var(--fg);
    background: var(--hover);
  }
  .tab {
    background: none;
    border: none;
    color: var(--dim);
    font: inherit;
    padding: 6px 14px;
    cursor: pointer;
    border-radius: 6px 6px 0 0;
  }
  .tab:hover {
    color: var(--fg);
  }
  .tab.active {
    background: var(--tab-active);
    color: var(--tab-active-fg);
    font-weight: 600;
  }
  .badge {
    margin-left: 7px;
    padding: 0 6px;
    border-radius: 8px;
    background: var(--tab-active);
    color: var(--tab-active-fg);
    font-size: 0.8em;
    font-weight: 600;
  }
  .tab.active .badge {
    background: var(--bg);
  }
</style>
