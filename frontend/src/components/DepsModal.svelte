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

<div class="overlay" role="dialog" aria-modal="true">
  <div class="modal">
    {#if deps.phase === 'checking'}
      <p class="dim">Checking dependencies…</p>
    {:else if deps.phase === 'prompt' && cur}
      <h2>{cur.title}</h2>
      <p>{cur.body}</p>
      <div class="btns">
        <button class="primary" onclick={confirmPrompt}>{cur.confirmLabel}</button>
        {#if cur.optionalLabel}
          <button onclick={declinePrompt}>{cur.optionalLabel}</button>
        {/if}
        {#if cur.mandatory}
          <button class="danger" onclick={quitApp}>Exit</button>
        {/if}
      </div>
    {:else if deps.phase === 'running'}
      <h2>{cur?.title}</h2>
      <p class="dim">Running…</p>
      <pre class="log" bind:this={logEl}>{deps.log.join('\n')}</pre>
    {:else if deps.phase === 'guide'}
      <h2>{deps.failedOp ? deps.failedOp.title : cur?.title}</h2>
      {#if deps.failedOp}
        <pre class="error">{deps.failedOp.error}</pre>
      {/if}
      <pre class="log">{(guideLines().length ? guideLines() : allGuides()).join('\n')}</pre>
      <div class="btns">
        {#if cur?.mandatory}
          <button class="danger" onclick={quitApp}>Exit</button>
        {:else}
          <button onclick={declinePrompt}>Continue</button>
        {/if}
        <button class="primary" onclick={runDepsCheck}>Re-check</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }
  .modal {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 20px 24px;
    width: min(560px, 90vw);
    max-height: 80vh;
    overflow-y: auto;
  }
  h2 {
    margin: 0 0 8px;
    color: var(--accent);
    font-size: 1rem;
  }
  .btns {
    display: flex;
    gap: 8px;
    margin-top: 14px;
  }
  button {
    font: inherit;
    padding: 6px 14px;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: none;
    color: var(--fg);
    cursor: pointer;
  }
  button.primary {
    background: var(--tab-active);
    color: var(--tab-active-fg);
    border-color: transparent;
    font-weight: 600;
  }
  button.danger {
    color: var(--red);
    border-color: var(--red);
  }
  .log {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 10px;
    max-height: 240px;
    overflow-y: auto;
    font-size: 0.85em;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .error {
    color: var(--orange);
    white-space: pre-wrap;
    word-break: break-word;
    font-size: 0.85em;
  }
</style>
