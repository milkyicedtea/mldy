<script lang="ts">
import { onMount } from 'svelte'
import DepsModal from '@/components/DepsModal.svelte'
import Downloads from '@/components/Downloads.svelte'
import Footer from '@/components/Footer.svelte'
import History from '@/components/History.svelte'
import InputQueue from '@/components/InputQueue.svelte'
import SettingsModal from '@/components/SettingsModal.svelte'
import Tabs from '@/components/Tabs.svelte'
import {
  activateSelected,
  clearAll,
  closeSettings,
  deleteSelected,
  deps,
  initStore,
  moveCursor,
  queued,
  setScreen,
  start,
  ui,
} from '@/lib/store.svelte'

onMount(() => initStore())

function keydown(e: KeyboardEvent) {
  if (ui.settingsOpen) {
    if (e.key === 'Escape') closeSettings()
    return // modal is up: it handles its own input
  }
  if (deps.phase !== 'done') return // modal is up: it handles its own input

  // Global shortcuts. Ctrl+D/Ctrl+R are reserved by the webview
  // (bookmark/reload), so start/clear use Ctrl+Enter/Alt+R instead.
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    e.preventDefault()
    start()
    return
  }
  if (e.altKey && !e.ctrlKey && !e.metaKey && e.key.toLowerCase() === 'r') {
    e.preventDefault()
    clearAll()
    return
  }

  const target = e.target as HTMLElement | null
  const inField = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA'
  if (inField) {
    if (e.key === 'ArrowDown' && ui.screen === 'input' && queued().length > 0) {
      e.preventDefault()
      ui.inputFocused = false
    }
    return
  }

  // Tab cycling across screens (1/2/3 jump directly).
  if (e.key === '1') {
    setScreen('input')
    return
  }
  if (e.key === '2') {
    setScreen('downloads')
    return
  }
  if (e.key === '3') {
    setScreen('history')
    return
  }
  if (e.key === 'Tab') {
    e.preventDefault()
    const order = ['input', 'downloads', 'history'] as const
    const idx = order.indexOf(ui.screen)
    setScreen(order[e.shiftKey ? (idx + 2) % 3 : (idx + 1) % 3])
    return
  }

  if (ui.screen === 'input') {
    if (!ui.inputFocused) {
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        moveCursor(-1, 'input')
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        moveCursor(1, 'input')
      } else if (e.key === 'Enter') {
        activateSelected('input', false)
      } else if (e.key === ' ') {
        e.preventDefault()
        activateSelected('input', true)
      } else if (e.key === 'd' || e.key === 'Delete' || e.key === 'Backspace') {
        e.preventDefault()
        deleteSelected('input')
      } else if (e.key === 's') {
        start()
      } else if (e.key.length === 1) {
        // Type to focus the URL input, like the TUI.
        ui.inputFocused = true
        ui.url += e.key
      }
    }
  } else if (ui.screen === 'history') {
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      moveCursor(-1, 'history')
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      moveCursor(1, 'history')
    } else if (e.key === 'Enter') {
      activateSelected('history', false)
    } else if (e.key === 'd' || e.key === 'Delete' || e.key === 'Backspace') {
      e.preventDefault()
      deleteSelected('history')
    }
  }
}
</script>

<svelte:window on:keydown={keydown} />

<div class="app">
  <Tabs />
  <main class="content">
    {#if ui.state === null}
      <p class="dim">Loading…</p>
    {:else if ui.screen === 'input'}
      <InputQueue />
    {:else if ui.screen === 'downloads'}
      <Downloads />
    {:else}
      <History />
    {/if}
  </main>
  <Footer />
</div>
<DepsModal />
<SettingsModal />

<style>
  .app {
    flex-direction: column;
    height: 100vh;
  }
  .content {
    flex: 1;
    overflow-y: auto;
    padding: 12px 16px;
  }
</style>
