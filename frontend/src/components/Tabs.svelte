<script lang="ts">
import { setScreen, ui } from '@/lib/store.svelte'

const tabs = [
  { id: 'input', label: 'Input/Queue' },
  { id: 'downloads', label: 'Downloads' },
  { id: 'history', label: 'History' },
] as const
</script>

<nav class="tabs">
  {#each tabs as t (t.id)}
    <button
      class="tab"
      class:active={ui.screen === t.id}
      onclick={() => setScreen(t.id)}
    >{t.label}</button>
  {/each}
</nav>

<style>
  .tabs {
    display: flex;
    gap: 4px;
    padding: 8px 12px 0;
    border-bottom: 1px solid var(--border);
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
</style>
