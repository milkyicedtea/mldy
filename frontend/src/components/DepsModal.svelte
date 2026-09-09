<script lang="ts">
import {
  activePrompt,
  confirmPrompt,
  declinePrompt,
  deps,
  GUIDES,
  quitApp,
  runDepsCheck,
} from '@/lib/store.svelte'

const cur = $derived(activePrompt())

let logEl = $state<HTMLElement | null>(null)

$effect(() => {
  if (deps.phase === 'running' && logEl) {
    logEl.scrollTop = logEl.scrollHeight
  }
})

function guideLines(): string[] {
  if (deps.failedOp) return Object.values(GUIDES).flat()
  return GUIDES[cur?.dep ?? ''] ?? []
}

function allGuides(): string[] {
  return Object.values(GUIDES).flat()
}

runDepsCheck()
</script>

{#if deps.phase !== 'done'}
<div class="overlay" role="dialog" aria-modal="true">
  <div class="modal w-[min(560px,90vw)] max-h-[80vh]">
    {#if deps.phase === 'checking'}
      <p class="dim">Checking dependencies…</p>
    {:else if deps.phase === 'prompt' && cur}
      <h2 class="modal-title mb-2">{cur.title}</h2>
      <p>{cur.body}</p>
      <div class="btns">
        <button class="mbtn primary" onclick={confirmPrompt}>{cur.confirmLabel}</button>
        {#if cur.optionalLabel}
          <button class="mbtn" onclick={declinePrompt}>{cur.optionalLabel}</button>
        {/if}
        {#if cur.mandatory}
          <button class="mbtn danger" onclick={quitApp}>Exit</button>
        {/if}
      </div>
    {:else if deps.phase === 'running'}
      <h2 class="modal-title mb-2">{cur?.title}</h2>
      <p class="dim">Running…</p>
      <pre class="log" bind:this={logEl}>{deps.log.join('\n')}</pre>
    {:else if deps.phase === 'guide'}
      <h2 class="modal-title mb-2">{deps.failedOp ? deps.failedOp.title : cur?.title}</h2>
      {#if deps.failedOp}
        <pre class="text-orange text-[0.85em] whitespace-pre-wrap break-words">{deps.failedOp.error}</pre>
      {/if}
      <pre class="log">{(guideLines().length ? guideLines() : allGuides()).join('\n')}</pre>
      <div class="btns">
        {#if cur?.mandatory}
          <button class="mbtn danger" onclick={quitApp}>Exit</button>
        {:else}
          <button class="mbtn" onclick={declinePrompt}>Continue</button>
        {/if}
        <button class="mbtn primary" onclick={runDepsCheck}>Re-check</button>
      </div>
    {/if}
  </div>
</div>
{/if}

