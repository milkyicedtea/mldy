<script lang="ts">
import { completed, queued, ui } from '@/lib/store.svelte'

const resolving = $derived(ui.state?.resolving ?? 0)
const running = $derived(ui.state?.isRunning ?? false)
const queuedCount = $derived(queued().length)
const historyCount = $derived(completed().length)

const hints = $derived.by(() => {
  const h = ['1/2/3 or tab: switch screen']
  if (ui.screen === 'input') {
    if (ui.inputFocused) {
      h.push('enter: add URL', 'down: focus list')
    } else {
      h.push('up: focus input', 'enter: expand', 'space: start', 'd: delete')
    }
    if (resolving > 0) h.push('resolving…')
    else if (running) h.push('ctrl+enter: downloading…')
    else if (queuedCount > 0) h.push('ctrl+enter: start')
  } else if (ui.screen === 'downloads') {
    if (running) h.push('downloading…')
    else if (queuedCount > 0) h.push('ctrl+enter: start downloads')
  } else if (ui.screen === 'history') {
    if (historyCount > 0) h.push('up/down: move', 'enter: expand', 'd: remove')
    else h.push('no history yet')
  }
  return h
})
</script>

<footer class="footer">
  {hints.join('  •  ')}
</footer>

<style>
  .footer {
    border-top: 1px solid var(--border);
    padding: 6px 16px;
    color: var(--dim);
    font-size: 0.85rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
