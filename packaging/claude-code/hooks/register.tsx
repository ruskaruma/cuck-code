// Cuck Code inside Claude Code. When you send a prompt she walks through the
// door; while Claude works you hear them through it; when Claude finishes he
// walks out, turns to you and winks. It all plays in the band above the
// prompt and vanishes after. The spinner and the "Baked for 3s"
// line get the same treatment. /cuck-quiet mutes the sound.

import { atom, read, update } from 'claude-code'
import type { Register, Timer } from 'claude-code'

import type { Clip, Kind } from '../types'
import {
  BUSY_LINES, DURATION, ENTER_LINES, HER_LINES, HIS_LINES, LEAVE_LINES, MAX_COLUMNS, MIN_COLUMNS, ROWS,
  SPINNER_WORDS, WINK_AT, encode, frame, pick,
} from './scene'

const clip = atom({ plugin: 'cuck-code', key: 'clip' } as const, null)

// Plays a clip with whatever is on the machine: afplay on macOS, PipeWire,
// PulseAudio or ALSA on Linux.
const PLAYER =
  'if command -v afplay >/dev/null; then afplay "$1"; elif command -v pw-play >/dev/null; then pw-play "$1"; ' +
  'elif command -v paplay >/dev/null; then paplay "$1"; elif command -v aplay >/dev/null; then aplay -q "$1"; fi'

export const register: Register = on => {
  let ticker: Timer | null = null
  let band: { requestId: string; columns: number } | null = null
  let spinnerWord = pick(SPINNER_WORDS)
  let current: Clip | null = null
  let working = false
  let quiet = false
  // While busy: what the band says and since when, and when the next sounds are due.
  let speech = { text: '', since: 0 }
  let nextBeat = 0
  let nextVoice = 0
  let beats = 0
  let lastVoice = ''
  let winked = false

  const newClip = (kind: Kind, now: number): Clip => {
    current = { kind, caption: pick(kind === 'enter' ? ENTER_LINES : kind === 'busy' ? BUSY_LINES : LEAVE_LINES), startedAt: now }
    if (kind === 'busy') {
      speech = { text: current.caption, since: now }
      nextBeat = now
      nextVoice = now + 2000
      beats = 0
    }
    winked = false
    return current
  }

  // What to do at time now for clip c: the frame to draw, the sound to play,
  // and whether the clip has ended. `next` is the clip to hand over to.
  type Step = { cells?: string; requestId?: string; sound?: string; done?: boolean; next?: Kind }
  const step = (c: Clip, now: number): Step => {
    if (current !== c) return { done: true }
    const t = now - c.startedAt
    if (t > DURATION[c.kind]) {
      return c.kind === 'enter' && working ? { next: 'busy' } : { done: true }
    }
    let sound: string | undefined
    let caption = c.caption
    let age = t
    if (c.kind === 'busy') {
      if (now >= nextVoice) {
        // Hers most of the time, his now and then; never the same twice running.
        const lines = Math.random() < 0.8 ? HER_LINES : HIS_LINES
        const voice = pick(Object.keys(lines).filter(v => v !== lastVoice))
        lastVoice = voice
        speech = { text: lines[voice] ?? '', since: now }
        sound = voice
        nextVoice = now + 6000 + Math.random() * 5000
        nextBeat = Math.max(nextBeat, now + 1200)
      } else if (now >= nextBeat) {
        // The bed: creak one way, creak back, and every sixth beat the headboard.
        beats += 1
        sound = beats % 6 === 0 ? 'thump' : beats % 2 ? 'creak1' : 'creak2'
        nextBeat = now + 800
        if (now - speech.since > 2600) speech = { text: pick(BUSY_LINES), since: now }
      }
      caption = speech.text
      age = now - speech.since
    } else if (c.kind === 'leave' && t >= WINK_AT && !winked) {
      winked = true
      sound = 'him_gooddee' // to you, as he winks
    }
    const cells = band ? encode(frame(c.kind, t, band.columns, caption, age)) : undefined
    return { cells, requestId: band?.requestId, sound: quiet ? undefined : sound }
  }

  on('session.start', async ($, e, next) => {
    quiet = (await $.store.get('quiet')) === true
    await $.command.register({ name: 'cuck-quiet', description: 'Mute or unmute the Cuck Code sounds' })
    return next(e)
  })

  on('command.run', { command: 'cuck-quiet' }, async $ => {
    quiet = !quiet
    await $.store.set('quiet', quiet)
    return { text: quiet ? 'Cuck Code is muted. They keep it down now.' : 'Cuck Code sound is back on.' }
  })

  on('prompt.submit', async ($, e, next) => {
    if (!e.command && !e.isMeta && e.text.trim() !== '') {
      working = true
      spinnerWord = pick(SPINNER_WORDS)
      ticker?.cancel()
      let c = newClip('enter', await $.clock.now())
      await update($, clip, () => c)
      ticker = $.clock.every(33, () => {
        void $.clock.now().then(now => {
          const s = step(c, now)
          if (s.next) {
            c = newClip(s.next, now)
            void update($, clip, () => c)
          } else if (s.done) {
            if (current === c) {
              ticker?.cancel()
              ticker = null
              current = null
              void update($, clip, () => null)
            }
          } else {
            if (s.cells && s.requestId) void $.ui.blit({ requestId: s.requestId, key: 'strip', cells: s.cells }).catch(() => {})
            if (s.sound) void $.process.run(['sh', '-c', PLAYER, 'sh', `${$.plugin.root}/sounds/${s.sound}.wav`], { timeoutMs: 10000 }).catch(() => {})
          }
        })
      })
    }
    return next(e)
  }).catch(($, e, next) => next(e)) // a broken show must never hold up the prompt

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    if (e.agentId) {
      return result
    }
    working = false
    ticker?.cancel()
    ticker = null
    if (e.isAborted) {
      current = null
      await update($, clip, () => null)
      return result
    }
    const c = newClip('leave', await $.clock.now())
    await update($, clip, () => c)
    ticker = $.clock.every(33, () => {
      void $.clock.now().then(now => {
        const s = step(c, now)
        if (s.done) {
          if (current === c) {
            ticker?.cancel()
            ticker = null
            current = null
            void update($, clip, () => null)
          }
        } else {
          if (s.cells && s.requestId) void $.ui.blit({ requestId: s.requestId, key: 'strip', cells: s.cells }).catch(() => {})
          if (s.sound) void $.process.run(['sh', '-c', PLAYER, 'sh', `${$.plugin.root}/sounds/${s.sound}.wav`], { timeoutMs: 10000 }).catch(() => {})
        }
      })
    })
    if (!quiet) {
      void $.process
        .run(['sh', '-c', PLAYER, 'sh', `${$.plugin.root}/sounds/ding.wav`], { timeoutMs: 10000 })
        .catch(() => $.audio.play({ asset: 'sounds/ding.wav' }))
        .catch(() => {})
    }
    return result
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    const c = await read($, clip)
    if (c === null || e.props.hasSurvey) {
      return next(e)
    }
    const columns = Math.min(e.props.bodyColumns, MAX_COLUMNS)
    const caption = c.kind === 'busy' ? speech.text || c.caption : c.caption
    if (e.surface !== 'terminal' || columns < MIN_COLUMNS) {
      const { Text } = $.ui.resolve(e)
      return <Text dimColor>{caption}</Text>
    }
    const { Raster } = $.ui.resolve(e)
    band = { requestId: e.requestId, columns }
    const now = await $.clock.now()
    const t = Math.max(0, now - c.startedAt)
    const age = c.kind === 'busy' ? now - speech.since : t
    return <Raster key="strip" columns={columns} rows={ROWS} cells={encode(frame(c.kind, t, columns, caption, age))} />
  })

  on('ui.render', { component: 'Spinner' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, word: spinnerWord } }),
  )

  on('ui.render', { component: 'TurnDuration' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, word: 'Cucked' } }),
  )
}
