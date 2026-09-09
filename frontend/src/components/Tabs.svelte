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

<nav class="flex items-center border-b border-border px-3 pt-2">
  <div class="flex gap-1">
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
  <button class="ml-auto cursor-pointer rounded-md border-0 bg-transparent px-2 py-1 font-[inherit] text-[1.05em] text-dim hover:bg-hover hover:text-fg" title="Settings" aria-label="Open settings" onclick={openSettings}>⚙</button>
</nav>

