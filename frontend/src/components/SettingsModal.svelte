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
    <div class="modal w-[min(480px,90vw)] max-h-[85vh]">
      <h2 class="modal-title mb-3.5">Settings</h2>
      {#if form}
        <form
          onsubmit={(e) => {
            e.preventDefault()
            void save()
          }}
        >
          <div class="field">
            <label for="cfg-kind">Output kind</label>
            <select id="cfg-kind" class="field-input" bind:value={form.kind}>
              <option value="auto">Auto (decide by format)</option>
              <option value="audio">Audio only</option>
              <option value="video">Video</option>
            </select>
          </div>

          <div class="field">
            <label for="cfg-format">Format</label>
            <select id="cfg-format" class="field-input" bind:value={form.format}>
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
                <select id="cfg-aq" class="field-input" bind:value={form.audioQuality}>
                  {#each VBR_QUALITIES as q (q)}
                    <option value={q}>
                      {q === '0' ? '0 — best' : q === '10' ? '10 — worst' : q === '5' ? '5 (default)' : q}
                    </option>
                  {/each}
                  <option value={CUSTOM}>Custom bitrate (CBR)</option>
                </select>
                {#if form.audioQuality === CUSTOM}
                  <input class="field-input flex-1" bind:value={form.customAudio} placeholder="e.g. 192K" spellcheck="false" />
                {/if}
              </div>
            </div>
          {/if}

          {#if form.kind !== 'audio'}
            <div class="field">
              <label for="cfg-vq">Video quality</label>
              <select id="cfg-vq" class="field-input" bind:value={form.videoQuality}>
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
            <select id="cfg-runtime" class="field-input" bind:value={form.jsRuntime}>
              <option value="auto">Auto (deno → bun → node)</option>
              <option value="deno">Deno</option>
              <option value="bun">Bun</option>
              <option value="node">Node.js</option>
            </select>
          </div>

          <div class="field">
            <label for="cfg-folder">Output folder</label>
            <div class="inline">
              <input id="cfg-folder" class="field-input flex-1" bind:value={form.outputFolder} spellcheck="false" />
              <button type="button" class="mbtn" disabled={browsing} onclick={() => void browse()}>Browse…</button>
            </div>
          </div>

          {#if error}<p class="mt-1 text-[0.85em] break-words text-orange">{error}</p>{/if}
          <p class="dim mt-1 text-[0.85em]">Applies to new downloads; saved to ~/.config/mldy/config.yaml.</p>

          <div class="btns">
            <button type="button" class="mbtn" onclick={closeSettings}>Cancel</button>
            <button type="submit" class="mbtn primary" disabled={saving}>{saving ? 'Saving…' : 'Save'}</button>
          </div>
        </form>
      {/if}
    </div>
  </div>
{/if}

