import { ref } from 'vue'

const STORAGE_KEY = 'tideflow-typing-sound'

function loadEnabled(): boolean {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return v !== 'false'
  } catch {
    return true
  }
}

export function useTypingSounds() {
  const enabled = ref(loadEnabled())

  let audioCtx: AudioContext | null = null

  const getCtx = () => {
    if (!audioCtx) {
      audioCtx = new AudioContext()
    }
    return audioCtx
  }

  const saveEnabled = () => {
    try { localStorage.setItem(STORAGE_KEY, String(enabled.value)) } catch {}
  }

  const toggle = () => {
    enabled.value = !enabled.value
    saveEnabled()
  }

  // Crisp keystroke: short high-freq click
  const playKey = () => {
    if (!enabled.value) return
    const ctx = getCtx()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'square'
    osc.frequency.setValueAtTime(4200 + Math.random() * 600, ctx.currentTime)
    gain.gain.setValueAtTime(0.08, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.05)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.05)
  }

  // Backspace: lower pitch click
  const playBackspace = () => {
    if (!enabled.value) return
    const ctx = getCtx()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'square'
    osc.frequency.setValueAtTime(2200, ctx.currentTime)
    gain.gain.setValueAtTime(0.06, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.06)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.06)
  }

  // Correct: cheerful two-note chime
  const playCorrect = () => {
    if (!enabled.value) return
    const ctx = getCtx()
    const notes = [523, 659] // C5, E5
    notes.forEach((freq, i) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.type = 'sine'
      osc.frequency.setValueAtTime(freq, ctx.currentTime + i * 0.08)
      gain.gain.setValueAtTime(0.12, ctx.currentTime + i * 0.08)
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + i * 0.08 + 0.2)
      osc.start(ctx.currentTime + i * 0.08)
      osc.stop(ctx.currentTime + i * 0.08 + 0.2)
    })
  }

  // Wrong: low buzz
  const playWrong = () => {
    if (!enabled.value) return
    const ctx = getCtx()
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.type = 'sawtooth'
    osc.frequency.setValueAtTime(180, ctx.currentTime)
    gain.gain.setValueAtTime(0.1, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.15)
    osc.start(ctx.currentTime)
    osc.stop(ctx.currentTime + 0.15)
  }

  return { enabled, toggle, playKey, playBackspace, playCorrect, playWrong }
}
