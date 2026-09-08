<script lang="ts">
import type { OutputKind } from '@bindings/mldy/internal/config'
import { fade } from 'svelte/transition'
import { closeSettings, pickOutputFolder, saveSettings, ui } from '@/lib/store.svelte'

interface SettingsForm {
  kind: string
  format: string
  audioQuality: string
  customAudio: string
  videoQuality: string
  jsRuntime: string
  outputFolder: string
}

const AUDIO_FORMATS = ['mp3', 'm4a', 'opus', 'flac', 'wav', 'aac']
const VIDEO_FORMATS = ['mp4', 'mkv', 'webm']
const VBR_QUALITIES = ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '10']
const HEIGHTS = ['2160', '1440', '1080', '720', '480', '360']
const CUSTOM = '__custom__'

let form = $state<SettingsForm | null>(null)
let error = $state<string | null>(null)
let saving = $state(false)

const show = $derived(ui.settingsOpen)

// Initialize the form from the current config each time the modal opens,
// and discard it when it closes.
$effect(() => {
  if (!show) {
    form = null
    error = null
    return
  }
  const c = ui.state?.config
  if (!c || form) return
  const vbr = /^\d+$/.test(c.AudioQuality) && Number(c.AudioQuality) <= 10
  form = {
    kind: c.Kind,
    format: c.Format,
    audioQuality: vbr ? c.AudioQuality : CUSTOM,
    customAudio: vbr ? '' : c.AudioQuality,
    videoQuality: c.VideoQuality.replace(/p$/i, ''),
    jsRuntime: c.JSRuntime || 'auto',
    outputFolder: c.OutputFolder,
  }
})

// withCurrent prepends the stored value when it is not one of the presets,
// so an existing custom config survives a round-trip through the form.
function withCurrent(list: string[], current: string): string[] {
  if (current && !list.includes(current)) return [current, ...list]
  return list
}

const audioFormats = $derived(
  form ? withCurrent(AUDIO_FORMATS, form.kind === 'video' ? '' : form.format) : []
)
const videoFormats = $derived(
  form ? withCurrent(VIDEO_FORMATS, form.kind === 'audio' ? '' : form.format) : []
)

let browsing = $state(false)

async function browse(): Promise<void> {
  if (!form || browsing) return
  browsing = true
  try {
    const dir = await pickOutputFolder()
    // Replace the whole form object — deep mutation on a proxied $state can
    // miss the bound input on some webview builds.
    if (dir) form = { ...form, outputFolder: dir }
  } catch (e) {
    error = e instanceof Error ? e.message : String(e)
  } finally {
    browsing = false
  }
}

async function save(): Promise<void> {
  if (!form || saving) return
  const audioQuality =
    form.audioQuality === CUSTOM ? form.customAudio.trim() || '5' : form.audioQuality
  saving = true
  try {
    await saveSettings({
      Kind: form.kind as OutputKind,
      Format: form.format,
      AudioQuality: audioQuality,
      VideoQuality: form.videoQuality,
      OutputFolder: form.outputFolder.trim(),
      JSRuntime: form.jsRuntime,
    })
    closeSettings()
  } catch (e) {
    error = e instanceof Error ? e.message : String(e)
  } finally {
    saving = false
  }
}
</script>

{#if ui.settingsOpen}
  <div class="overlay" role="dialog" aria-modal="true" transition:fade={{ duration: 120 }}>
    <div class="modal">
      <h2>Settings</h2>
      {#if form}
        <form
          onsubmit={(e) => {
            e.preventDefault()
            void save()
          }}
        >
          <div class="field">
            <label for="cfg-kind">Output kind</label>
            <select id="cfg-kind" bind:value={form.kind}>
              <option value="auto">Auto (decide by format)</option>
              <option value="audio">Audio only</option>
              <option value="video">Video</option>
            </select>
          </div>

          <div class="field">
            <label for="cfg-format">Format</label>
            <select id="cfg-format" bind:value={form.format}>
              {#if form.kind !== 'video'}
                <optgroup label="Audio">
                  {#each audioFormats as f (f)}<option value={f}>{f}</option>{/each}
                </optgroup>
              {/if}
              {#if form.kind !== 'audio'}
                <optgroup label="Video">
                  {#each videoFormats as f (f)}<option value={f}>{f}</option>{/each}
                </optgroup>
              {/if}
            </select>
          </div>

          {#if form.kind !== 'video'}
            <div class="field">
              <label for="cfg-aq">Audio quality</label>
              <div class="inline">
                <select id="cfg-aq" bind:value={form.audioQuality}>
                  {#each VBR_QUALITIES as q (q)}
                    <option value={q}>
                      {q === '0' ? '0 — best' : q === '10' ? '10 — worst' : q === '5' ? '5 (default)' : q}
                    </option>
                  {/each}
                  <option value={CUSTOM}>Custom bitrate (CBR)</option>
                </select>
                {#if form.audioQuality === CUSTOM}
                  <input class="grow" bind:value={form.customAudio} placeholder="e.g. 192K" spellcheck="false" />
                {/if}
              </div>
            </div>
          {/if}

          {#if form.kind !== 'audio'}
            <div class="field">
              <label for="cfg-vq">Video quality</label>
              <select id="cfg-vq" bind:value={form.videoQuality}>
                <option value="best">Best available</option>
                {#each HEIGHTS as h (h)}<option value={h}>{h}p or lower</option>{/each}
                {#if !HEIGHTS.includes(form.videoQuality) && form.videoQuality !== 'best'}
                  <option value={form.videoQuality}>{form.videoQuality} (current)</option>
                {/if}
              </select>
            </div>
          {/if}

          <div class="field">
            <label for="cfg-runtime">JavaScript runtime</label>
            <select id="cfg-runtime" bind:value={form.jsRuntime}>
              <option value="auto">Auto (deno → bun → node)</option>
              <option value="deno">Deno</option>
              <option value="bun">Bun</option>
              <option value="node">Node.js</option>
            </select>
          </div>

          <div class="field">
            <label for="cfg-folder">Output folder</label>
            <div class="inline">
              <input id="cfg-folder" class="grow" bind:value={form.outputFolder} spellcheck="false" />
              <button type="button" disabled={browsing} onclick={() => void browse()}>Browse…</button>
            </div>
          </div>

          {#if error}<p class="error">{error}</p>{/if}
          <p class="dim note">Applies to new downloads; saved to ~/.config/mldy/config.yaml.</p>

          <div class="btns">
            <button type="button" onclick={closeSettings}>Cancel</button>
            <button type="submit" class="primary" disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

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
    width: min(480px, 90vw);
    max-height: 85vh;
    overflow-y: auto;
  }
  h2 {
    margin: 0 0 14px;
    color: var(--accent);
    font-size: 1rem;
  }
  .field {
    display: grid;
    gap: 4px;
    margin-bottom: 12px;
  }
  .field label {
    color: var(--dim);
    font-size: 0.85em;
  }
  select,
  input {
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--fg);
    font: inherit;
    padding: 6px 8px;
    outline: none;
    width: 100%;
    box-sizing: border-box;
  }
  /* Popup list on WebKitGTK uses UA defaults unless option carries its own
     colors. */
  select option,
  select optgroup {
    background-color: var(--panel);
    color: var(--fg);
  }
  select:focus,
  input:focus {
    border-color: var(--select);
  }
  .inline {
    display: flex;
    gap: 8px;
  }
  .inline select {
    flex: 0 0 55%;
  }
  .inline .grow {
    flex: 1;
  }
  .note {
    margin: 4px 0 0;
    font-size: 0.85em;
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
  button:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .error {
    color: var(--orange);
    font-size: 0.85em;
    margin: 4px 0 0;
    word-break: break-word;
  }
</style>
