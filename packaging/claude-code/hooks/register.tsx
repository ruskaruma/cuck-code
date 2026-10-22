// Cuck Code inside Claude Code. When you send a prompt she walks through the
// door; while Claude works you hear them through it; when Claude finishes he
// walks out, turns to you and winks. It all plays in the band above the
// prompt and vanishes after. When Claude needs your permission he leans out
// and asks you for a yes. The spinner and the "Baked for 3s" line get the
// same treatment. /cuck sets it up: quiet, safe for work, costumes.

import { atom, read, update } from 'claude-code'
import type { Register, Timer } from 'claude-code'

import type { Clip, EventKind, Kind } from '../types'
import { linesFor, season } from './lines'
import type { Look, Shot, Skin } from './scene'
import { DURATION, EVENT_DURATION, MAX_COLUMNS, MIN_COLUMNS, ROWS, WINK_AT, encode, frame, pick } from './scene'

const clip = atom({ plugin: 'cuck-code', key: 'clip' } as const, null)

type Os = 'mac' | 'linux' | 'windows'

// Plays a clip with whatever is on the machine: afplay on macOS, PipeWire,
// PulseAudio or ALSA on Linux, PowerShell's sound player on Windows.
const PLAYER =
  'if command -v afplay >/dev/null; then afplay "$1"; elif command -v pw-play >/dev/null; then pw-play "$1"; ' +
  'elif command -v paplay >/dev/null; then paplay "$1"; elif command -v aplay >/dev/null; then aplay -q "$1"; fi'

function playerArgv(os: Os, file: string): string[] {
  if (os === 'windows') {
    const path = file.replace(/'/g, "''")
    return ['powershell', '-NoProfile', '-NonInteractive', '-Command', `(New-Object Media.SoundPlayer '${path}').PlaySync()`]
  }
  return ['sh', '-c', PLAYER, 'sh', file]
}

// Exits 0 while you're on a call: the microphone is in use (Linux, Windows),
// or a Zoom or FaceTime call is up (macOS).
function meetingArgv(os: Os): string[] {
  if (os === 'windows') {
    const store = 'HKCU:\\Software\\Microsoft\\Windows\\CurrentVersion\\CapabilityAccessManager\\ConsentStore\\microphone'
    return ['powershell', '-NoProfile', '-NonInteractive', '-Command',
      `if (Get-ChildItem '${store}' -Recurse | Get-ItemProperty | Where-Object { $_.LastUsedTimeStop -eq 0 }) { exit 0 } else { exit 1 }`]
  }
  if (os === 'mac') return ['sh', '-c', "ps -A -o comm= | grep -qiE 'CptHost|aomhost|FaceTime'"]
  return ['sh', '-c', 'command -v pactl >/dev/null && [ -n "$(pactl list source-outputs short 2>/dev/null)" ]']
}

type Settings = {
  quiet: boolean // no sound
  sfw: boolean // clean lines and sounds, for screen-shares and the office
  skin: 'auto' | Skin // costumes: auto follows the calendar
  meetings: boolean // go quiet and safe for work while you're on a call
}
const DEFAULTS: Settings = { quiet: false, sfw: false, skin: 'auto', meetings: true }

// Settings as saved, checked field by field: anything unknown keeps its default.
function restore(saved: unknown, legacyQuiet: unknown): Settings {
  const s = { ...DEFAULTS, quiet: legacyQuiet === true }
  if (typeof saved !== 'object' || saved === null) return s
  const o = saved as Record<string, unknown>
  if (typeof o.quiet === 'boolean') s.quiet = o.quiet
  if (typeof o.sfw === 'boolean') s.sfw = o.sfw
  if (typeof o.meetings === 'boolean') s.meetings = o.meetings
  if (o.skin === 'auto' || o.skin === 'none' || o.skin === 'halloween') s.skin = o.skin
  return s
}

const onOff = (b: boolean) => (b ? 'on' : 'off')

function help(s: Settings): string {
  return [
    'Cuck Code',
    `  /cuck quiet               mute or unmute the sound (now ${s.quiet ? 'muted' : 'playing'})`,
    `  /cuck sfw                 safe for work: the same scene, clean lines (now ${onOff(s.sfw)})`,
    `  /cuck skin auto|halloween|off   costumes; auto puts them on for the season (now ${s.skin})`,
    `  /cuck meetings on|off     go quiet and safe for work while you're on a call (now ${onOff(s.meetings)})`,
  ].join('\n')
}

// Applies `/cuck <args>`: the new settings and what to tell you.
function configure(s: Settings, args: string): [Settings, string] {
  const [what = '', value = ''] = args.trim().toLowerCase().split(/\s+/)
  const flag = (now: boolean) => (value === 'on' ? true : value === 'off' ? false : !now)
  switch (what) {
    case 'quiet': {
      const quiet = flag(s.quiet)
      return [{ ...s, quiet }, quiet ? 'Cuck Code is muted. They keep it down now.' : 'Cuck Code sound is back on.']
    }
    case 'sfw': {
      const sfw = flag(s.sfw)
      return [{ ...s, sfw }, sfw ? 'Safe for work: the same scene, clean lines.' : 'Safe for work is off. Back to the usual.']
    }
    case 'meetings': {
      const meetings = flag(s.meetings)
      return [{ ...s, meetings }, meetings ? "On calls it goes quiet and safe for work." : 'It plays on calls too now.']
    }
    case 'skin':
      if (value === 'auto') return [{ ...s, skin: 'auto' }, 'Costumes follow the calendar: Halloween from Oct 24 to Nov 1.']
      if (value === 'halloween') return [{ ...s, skin: 'halloween' }, 'Halloween: she came as a witch, he came as a vampire.']
      if (value === 'off' || value === 'none') return [{ ...s, skin: 'none' }, 'No costumes.']
      return [s, `Skins: auto, halloween, off (now ${s.skin}).`]
  }
  return [s, help(s)]
}

// The words Claude Code's dialog uses when you turn a tool call down.
const REFUSED = /doesn't want to proceed|was rejected|denied/i

export const register: Register = on => {
  let settings = DEFAULTS
  let os: Os = 'linux'
  let meeting = false
  let checkedAt = -Infinity
  let band: { requestId: string; columns: number } | null = null
  let spinnerWord = pick(linesFor({ skin: 'none', sfw: false }).spinner)
  let current: Clip | null = null
  let working = false
  // While busy: what the band says and since when, when the next sounds are
  // due, and what's interrupting (he's at the door asking you for a yes).
  let speech = { text: '', since: 0 }
  let nextBeat = 0
  let nextVoice = 0
  let beats = 0
  let lastVoice = ''
  let event: { kind: EventKind; at: number; id?: string; cue: number } | null = null
  let winked = false
  // Set up in session.start, as they reach the engine: keep the clock running
  // while a clip plays, and play a sound.
  let kick = () => {}
  let play = (_name: string) => {}

  const lookAt = (now: number): Look => ({
    skin: settings.skin === 'auto' ? season(new Date(now)) : settings.skin,
    sfw: settings.sfw || meeting,
  })
  const muted = () => settings.quiet || meeting

  const startClip = (kind: Kind, now: number): Clip => {
    const lines = linesFor(lookAt(now))
    current = { kind, caption: pick(kind === 'enter' ? lines.enter : kind === 'busy' ? lines.busy : lines.leave), startedAt: now }
    if (kind === 'busy') {
      speech = { text: current.caption, since: now }
      nextBeat = now
      nextVoice = now + 2000
      beats = 0
    }
    winked = false
    return current
  }

  // You answered him: back in he goes, or the door slams.
  const answer = (refused: boolean, now: number) => {
    const lines = linesFor(lookAt(now))
    event = { kind: refused ? 'slam' : 'yes', at: now, cue: 0 }
    speech = { text: pick(refused ? lines.slam : lines.yes), since: now }
  }

  const shotFor = (c: Clip, now: number, columns: number): Shot => {
    const look = lookAt(now)
    const t = Math.max(0, now - c.startedAt)
    const busy = c.kind === 'busy'
    const shot: Shot = {
      kind: c.kind,
      t,
      columns,
      look,
      caption: busy ? speech.text || c.caption : c.caption,
      captionAge: busy ? now - speech.since : t,
      wink: linesFor(look).wink[1],
    }
    if (busy && event) shot.event = { kind: event.kind, t: now - event.at }
    return shot
  }

  // What to do at time `now` for clip c: the frame to draw and the sounds to
  // play; or that it's over (`done`), or hands over to another (`next`).
  type Step = { cells?: string; requestId?: string; sounds: string[]; done?: boolean; next?: Kind }
  const advance = (c: Clip, now: number): Step => {
    const t = now - c.startedAt
    if (t > DURATION[c.kind]) {
      return c.kind === 'enter' && working ? { next: 'busy', sounds: [] } : { done: true, sounds: [] }
    }
    const lines = linesFor(lookAt(now))
    const sounds: string[] = []
    if (c.kind === 'busy') {
      if (event && now - event.at > EVENT_DURATION[event.kind]) {
        event = null
        if (!working) return { done: true, sounds: [] } // the turn ended on a slam
        speech = { text: pick(lines.busy), since: now }
        nextBeat = now
        nextVoice = Math.max(nextVoice, now + 2500)
      }
      if (event) {
        // Each interruption's sounds, at their moments.
        const cues: [number, string][] =
          event.kind === 'knock' ? [[400, 'knock'], [1100, lines.knock[0]]] : event.kind === 'slam' ? [[0, 'slam']] : []
        for (let cue = cues[event.cue]; cue && now - event.at >= cue[0]; cue = cues[++event.cue]) sounds.push(cue[1])
      } else if (now >= nextVoice) {
        // Hers most of the time, his now and then; never the same twice running.
        const pool = Math.random() < 0.8 ? lines.her : lines.his
        const all = Object.keys(pool)
        const fresh = all.filter(v => v !== lastVoice)
        const voice = pick(fresh.length > 0 ? fresh : all)
        lastVoice = voice
        speech = { text: pool[voice] ?? '', since: now }
        sounds.push(voice)
        nextVoice = now + 6000 + Math.random() * 5000
        nextBeat = Math.max(nextBeat, now + 1200)
      } else if (now >= nextBeat) {
        // The beat: one way, back the other, and every sixth the headboard.
        beats += 1
        const { creaks, thump } = lines.bed
        sounds.push(thump !== undefined && beats % 6 === 0 ? thump : creaks[beats % 2] ?? creaks[0])
        nextBeat = now + 800
        if (now - speech.since > 2600) speech = { text: pick(lines.busy), since: now }
      }
    } else if (c.kind === 'leave' && t >= WINK_AT && !winked) {
      winked = true
      sounds.push(lines.wink[0]) // to you, as he winks
    }
    const cells = band ? encode(frame(shotFor(c, now, band.columns))) : undefined
    return { cells, requestId: band?.requestId, sounds: muted() ? [] : sounds }
  }

  // Each step on its own: one that fails mustn't take the show down with it.
  on('session.start', async ($, e, next) => {
    play = name => {
      void $.process.run(playerArgv(os, `${$.plugin.root}/sounds/${name}.wav`), { timeoutMs: 10000 }).catch(() => {})
    }
    let ticker: Timer | null = null
    const stop = () => {
      ticker?.cancel()
      ticker = null
    }
    const tick = () => {
      void $.clock.now().then(now => {
        if (settings.meetings && now - checkedAt > 15000) {
          checkedAt = now
          void $.process.run(meetingArgv(os), { timeoutMs: 5000 }).then(r => {
            meeting = settings.meetings && r.exitCode === 0
          }, () => {})
        }
        const c = current
        if (c === null) return stop()
        const s = advance(c, now)
        if (s.next) {
          const n = startClip(s.next, now)
          void update($, clip, () => n)
        } else if (s.done) {
          if (current === c) {
            current = null
            stop()
            void update($, clip, () => null)
          }
        } else {
          if (s.cells && s.requestId) void $.ui.blit({ requestId: s.requestId, key: 'strip', cells: s.cells }).catch(() => {})
          for (const name of s.sounds) play(name)
        }
      })
    }
    kick = () => {
      ticker ??= $.clock.every(33, tick)
    }

    settings = restore(await $.store.get('settings').catch(() => null), await $.store.get('quiet').catch(() => null))
    const windows = await $.env.get('OS').then(v => v === 'Windows_NT', () => false)
    const uname = windows ? null : await $.process.run(['uname'], { timeoutMs: 5000 }).catch(() => null)
    os = windows ? 'windows' : uname?.stdout.trim() === 'Darwin' ? 'mac' : 'linux'
    await $.command
      .register({
        name: 'cuck',
        description: 'Cuck Code: quiet, safe for work, costumes',
        argumentHint: 'quiet | sfw | skin auto|halloween|off | meetings on|off',
        immediate: true,
      })
      .catch(() => {})
    await $.command.register({ name: 'cuck-quiet', description: 'Mute or unmute the Cuck Code sounds', immediate: true }).catch(() => {})
    return next(e)
  })

  on('command.run', { command: 'cuck' }, async ($, e) => {
    const [s, text] = configure(settings, e.args)
    settings = s
    if (!s.meetings) meeting = false
    await $.store.set('settings', s)
    return { text }
  })

  on('command.run', { command: 'cuck-quiet' }, async $ => {
    const [s, text] = configure(settings, 'quiet')
    settings = s
    await $.store.set('settings', s)
    return { text }
  })

  on('prompt.submit', async ($, e, next) => {
    if (!e.command && !e.isMeta && e.text.trim() !== '') {
      const now = await $.clock.now()
      working = true
      event = null
      spinnerWord = pick(linesFor(lookAt(now)).spinner)
      const c = startClip('enter', now)
      await update($, clip, () => c)
      kick()
    }
    return next(e)
  }).catch(($, e, next) => next(e)) // a broken show must never hold up the prompt

  // Claude needs your permission: he leans out of the door and asks you,
  // with a line under the dialog too.
  on('tool.check', async ($, e, next) => {
    const verdict = await next(e)
    const id = e.tool_use_id
    if (verdict.decision === 'ask' && e.approver === 'person' && id !== undefined && working) {
      try {
        const now = await $.clock.now()
        const line = linesFor(lookAt(now)).knock[1]
        event = { kind: 'knock', at: now, id, cue: 0 }
        speech = { text: line, since: now + 400 }
        kick()
        // The dialog opens once this returns.
        for (const ms of [150, 700]) $.clock.after(ms, () => $.ui.notice(id, line))
      } catch {
        // the show is never worth a failed permission check
      }
    }
    return verdict
  })

  // The call you were asked about has run, or been turned down.
  on('tool.call', async ($, e, next) => {
    const result = await next(e)
    if (event?.kind === 'knock' && event.id === e.tool_use_id) {
      const said = 'text' in result && typeof result.text === 'string' ? result.text : String(result.result ?? '')
      answer(result.isError === true && REFUSED.test(said), await $.clock.now().catch(() => event?.at ?? 0))
    }
    return result
  })

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    if (e.agentId) {
      return result
    }
    working = false
    const now = await $.clock.now()
    if (e.isAborted) {
      // Turned down at the door: it slams, and that's the end of it.
      if (event?.kind === 'knock') answer(true, now)
      if (event?.kind === 'slam' && current?.kind === 'busy') {
        kick()
        return result
      }
      current = null
      event = null
      await update($, clip, () => null)
      return result
    }
    event = null
    const c = startClip('leave', now)
    await update($, clip, () => c)
    kick()
    if (!muted()) play('ding')
    return result
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    const c = await read($, clip)
    if (c === null || e.props.hasSurvey) {
      return next(e)
    }
    const columns = Math.min(e.props.bodyColumns, MAX_COLUMNS)
    const shot = shotFor(c, await $.clock.now(), columns)
    if (e.surface !== 'terminal' || columns < MIN_COLUMNS) {
      const { Text } = $.ui.resolve(e)
      return <Text dimColor>{c.kind === 'leave' && shot.t > WINK_AT ? shot.wink : shot.caption}</Text>
    }
    const { Raster } = $.ui.resolve(e)
    band = { requestId: e.requestId, columns }
    return <Raster key="strip" columns={columns} rows={ROWS} cells={encode(frame(shot))} />
  })

  on('ui.render', { component: 'Spinner' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, word: spinnerWord } }),
  )

  on('ui.render', { component: 'TurnDuration' }, ($, e, next) =>
    next({ ...e, props: { ...e.props, word: linesFor({ skin: 'none', sfw: settings.sfw || meeting }).done } }),
  )
}
