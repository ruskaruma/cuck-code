// Cuck Code inside Claude Code. When you send a prompt she walks through the
// door; while Claude works you hear them through it; when Claude finishes he
// walks out, turns to you and winks. It all plays in the band above the
// prompt and vanishes after. When Claude needs your permission he leans out
// and asks you for a yes. The spinner and the "Baked for 3s" line get the
// same treatment. /cuck sets it up (quiet, safe for work, costumes) and keeps
// your stats: a card, a README badge, achievements.

import { atom, read, update } from 'claude-code'
import type { Register, Timer } from 'claude-code'

import type { Clip, EventKind, Kind } from '../types'
import type { Lang, Pack } from './lines'
import { LANGS, TEST_COMMAND, linesFor, readPack, season } from './lines'
import type { Cast, Look, Shot, Skin } from './scene'
import { DURATION, MAX_COLUMNS, MIN_COLUMNS, ROWS, WINK_AT, encode, eventDuration, frame, pick } from './scene'
import type { Stats } from './stats'
import { EMPTY, award, badge, card, restoreStats, summary } from './stats'

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
  trailer: boolean // sign Claude's commits "Cucked-By: Claude"
  lang: Lang // the language of her lines
  cast: Cast // who walks in, and who Claude is
  pack: string | null // a voice pack, by its folder's name
}
const DEFAULTS: Settings = { quiet: false, sfw: false, skin: 'auto', meetings: true, trailer: false, lang: 'mix', cast: 'she-he', pack: null }
const SKINS: readonly Skin[] = ['none', 'halloween', 'christmas', 'valentine']
const CASTS: readonly Cast[] = ['she-he', 'he-she', 'he-he', 'she-she']
const LANG_NAMES: Record<Lang, string> = {
  mix: 'English and Spanish', en: 'English', es: 'Spanish', pt: 'Portuguese', fr: 'French', it: 'Italian', hi: 'Hindi',
}
const CAST_NAMES: Record<Cast, string> = {
  'she-he': 'she walks in, and Claude is him',
  'he-she': 'he walks in, and Claude is her',
  'he-he': 'he walks in, and Claude is him',
  'she-she': 'she walks in, and Claude is her',
}
const oneOf = <T extends string>(xs: readonly T[], v: unknown): v is T => typeof v === 'string' && (xs as readonly string[]).includes(v)

// Settings as saved, checked field by field: anything unknown keeps its default.
function restore(saved: unknown, legacyQuiet: unknown): Settings {
  const s = { ...DEFAULTS, quiet: legacyQuiet === true }
  if (typeof saved !== 'object' || saved === null) return s
  const o = saved as Record<string, unknown>
  if (typeof o.quiet === 'boolean') s.quiet = o.quiet
  if (typeof o.sfw === 'boolean') s.sfw = o.sfw
  if (typeof o.meetings === 'boolean') s.meetings = o.meetings
  if (typeof o.trailer === 'boolean') s.trailer = o.trailer
  if (o.skin === 'auto' || oneOf(SKINS, o.skin)) s.skin = o.skin
  if (oneOf(LANGS, o.lang)) s.lang = o.lang
  if (oneOf(CASTS, o.cast)) s.cast = o.cast
  if (typeof o.pack === 'string' && /^[\w.-]+$/.test(o.pack)) s.pack = o.pack
  return s
}

const onOff = (b: boolean) => (b ? 'on' : 'off')

function help(s: Settings): string {
  return [
    'Cuck Code',
    `  /cuck quiet               mute or unmute the sound (now ${s.quiet ? 'muted' : 'playing'})`,
    `  /cuck sfw                 safe for work: the same scene, clean lines (now ${onOff(s.sfw)})`,
    `  /cuck skin auto|halloween|christmas|valentine|off   costumes; auto puts them on for the season (now ${s.skin})`,
    `  /cuck lang mix|en|es|pt|fr|it|hi   the language of her lines (now ${s.lang})`,
    `  /cuck cast she-he|he-she|he-he|she-she   who walks in, then who Claude is (now ${s.cast})`,
    `  /cuck pack [name|off]     a voice pack: \`cuck pack add owner/repo\` installs one (now ${s.pack ?? 'none'})`,
    `  /cuck meetings on|off     go quiet and safe for work while you're on a call (now ${onOff(s.meetings)})`,
    '  /cuck stats               what you\'ve sat through, and your achievements',
    '  /cuck badge               a README badge of your time in the chair',
    `  /cuck trailer on|off      sign Claude's commits "Cucked-By: Claude" (now ${onOff(s.trailer)})`,
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
    case 'trailer': {
      const trailer = flag(s.trailer)
      return [{ ...s, trailer }, trailer ? 'Claude signs its commits "Cucked-By: Claude" now.' : "Claude's commits go unsigned again."]
    }
    case 'skin':
      if (value === 'auto') return [{ ...s, skin: 'auto' }, "Costumes follow the calendar: Halloween from Oct 24, Christmas from Dec 18, Valentine's from Feb 10."]
      if (value === 'halloween') return [{ ...s, skin: 'halloween' }, 'Halloween: a witch and a vampire.']
      if (value === 'christmas') return [{ ...s, skin: 'christmas' }, 'Christmas: an elf and Santa.']
      if (value === 'valentine') return [{ ...s, skin: 'valentine' }, "Valentine's: roses on the door."]
      if (value === 'off' || value === 'none') return [{ ...s, skin: 'none' }, 'No costumes.']
      return [s, `Skins: auto, halloween, christmas, valentine, off (now ${s.skin}).`]
    case 'lang':
      if (oneOf(LANGS, value)) return [{ ...s, lang: value }, `Her lines are in ${LANG_NAMES[value]} now.`]
      return [s, `Languages: ${LANGS.map(l => `${l} (${LANG_NAMES[l]})`).join(', ')}. Now ${s.lang}.`]
    case 'cast':
      if (oneOf(CASTS, value)) return [{ ...s, cast: value }, `The cast: ${CAST_NAMES[value]}.`]
      return [s, `Casts: ${CASTS.map(c => `${c} (${CAST_NAMES[c]})`).join('; ')}. Now ${s.cast}.`]
  }
  return [s, help(s)]
}

// Where `cuck pack add` puts voice packs: next to cuck's own config file.
function packsFolder(os: Os, env: { config?: string; xdg?: string; home?: string; appData?: string }): string {
  if (env.config) return `${env.config.replace(/[\\/][^\\/]*$/, '')}/packs`
  const base = os === 'windows' ? env.appData : os === 'mac' ? `${env.home}/Library/Application Support` : env.xdg || `${env.home}/.config`
  return `${base}/cuck-code/packs`
}

// The words Claude Code's dialog uses when you turn a tool call down.
const REFUSED = /doesn't want to proceed|was rejected|denied/i

// The longer Claude takes, the more the building notices: minutes into a turn.
const ESCALATIONS: readonly (readonly [number, EventKind])[] = [[5, 'pizza'], [15, 'neighbour'], [30, 'landlord']]

// Something interrupting the busy loop: started `at` (0 until its turn
// comes), its sounds played so far, how many friends walked in, and for a
// knock, the call he's asking about.
type Ev = { kind: EventKind; at: number; cue: number; count: number; id?: string }

export const register: Register = on => {
  let settings = DEFAULTS
  let os: Os = 'linux'
  let meeting = false
  let checkedAt = -Infinity
  let band: { requestId: string; columns: number } | null = null
  let spinnerWord = pick(linesFor({ skin: 'none', sfw: false, cast: 'she-he' }).spinner)
  let current: Clip | null = null
  let cue = 0 // the clip's sounds played so far
  let working = false
  // The turn: when it started, which model did it, the friends he brought,
  // and which visitors have come by.
  let turnStart = 0
  let model = ''
  let friends = 0
  let visited = new Set<EventKind>()
  // While busy: what the band says and since when, when the next sounds are
  // due, and what's interrupting it, first in line first.
  let speech = { text: '', since: 0 }
  let nextBeat = 0
  let nextVoice = 0
  let beats = 0
  let lastVoice = ''
  let queue: Ev[] = []
  let stats: Stats = { ...EMPTY, earned: {} }
  let pack: Pack | null = null
  let packs = ''
  // Set up in session.start, as they reach the engine: keep the clock running
  // while a clip plays, play a sound, keep the stats, show a toast.
  let kick = () => {}
  let play = (_name: string) => {}
  let save = () => {}
  let toast = (_text: string) => {}

  const lookAt = (now: number): Look => ({
    skin: settings.skin === 'auto' ? season(new Date(now)) : settings.skin,
    sfw: settings.sfw || meeting,
    cast: settings.cast,
  })
  const linesAt = (now: number) => linesFor(lookAt(now), model, settings.lang, pack)
  const muted = () => settings.quiet || meeting

  // Counts something you sat through, and hands out what it earned.
  const tally = (change: (s: Stats) => void, now: number, hour = -1) => {
    change(stats)
    for (const a of award(stats, now, hour)) {
      toast(`Achievement unlocked: ${a.name}. ${a.how[0]?.toUpperCase()}${a.how.slice(1)}.`)
      if (!muted()) play('fanfare')
    }
    save()
  }

  const startClip = (kind: Kind, now: number): Clip => {
    const lines = linesAt(now)
    const say: Record<Kind, readonly string[]> = {
      enter: lines.enter,
      busy: lines.busy,
      leave: lines.leave,
      breather: lines.happenings.breather.say,
      refused: lines.refused,
      morning: lines.morning,
      card: ['/cuck badge puts it in your README.'],
    }
    current = { kind, caption: pick(say[kind]), startedAt: now }
    cue = 0
    if (kind === 'busy') {
      speech = { text: current.caption, since: now }
      nextBeat = now
      nextVoice = now + 2000
      beats = 0
    }
    return current
  }

  // Lines something up to interrupt the busy loop. A knock goes first, in
  // place of whatever was on; friends arriving together walk in together.
  const happen = (kind: EventKind, extra: Partial<Ev> = {}) => {
    const ev: Ev = { kind, at: 0, cue: 0, count: 1, ...extra }
    if (kind === 'knock') queue = [ev, ...queue.slice(1)]
    else if (kind === 'friend') {
      const waiting = queue.find(q => q.kind === 'friend' && q.at === 0)
      if (waiting) waiting.count += 1
      else queue.push(ev)
    } else if (queue.length < 4 && !queue.some(q => q.kind === kind && q.at === 0)) queue.push(ev)
  }

  // Its turn has come: what the band says while it's on.
  const begin = (ev: Ev, now: number) => {
    const lines = linesAt(now)
    ev.at = now
    ev.cue = 0
    if (ev.kind === 'knock') speech = { text: lines.knock[1], since: now + 400 }
    else if (ev.kind === 'friend') speech = { text: (ev.count === 1 ? lines.friends[0] : lines.friends[1]).replace('{n}', String(ev.count)), since: now }
    else if (ev.kind !== 'yes' && ev.kind !== 'slam') {
      const say = lines.happenings[ev.kind].say
      if (say.length > 0) speech = { text: pick(say), since: now }
    }
  }

  // You answered him: back in he goes, or the door slams.
  const answer = (refused: boolean, now: number) => {
    const lines = linesAt(now)
    queue[0] = { kind: refused ? 'slam' : 'yes', at: now, cue: 0, count: 1 }
    speech = { text: pick(refused ? lines.slam : lines.yes), since: now }
    if (!refused) tally(s => (s.yeses += 1), now)
  }

  // The sounds of a clip or an event, at their moments.
  const clipCues = (kind: Kind, now: number): readonly (readonly [number, string])[] => {
    const lines = linesAt(now)
    if (kind === 'leave') return [[WINK_AT, lines.wink[0]]] // to you, as he winks
    if (kind === 'refused') return [[300, 'lock']]
    if (kind === 'breather') return lines.happenings.breather.cues
    return []
  }
  const eventCues = (ev: Ev, now: number): readonly (readonly [number, string])[] => {
    const lines = linesAt(now)
    if (ev.kind === 'knock') return [[400, 'knock'], [1100, lines.knock[0]]]
    if (ev.kind === 'slam') return [[0, 'slam']]
    if (ev.kind === 'yes') return []
    return lines.happenings[ev.kind].cues
  }

  const shotFor = (c: Clip, now: number, columns: number): Shot => {
    const look = lookAt(now)
    const t = Math.max(0, now - c.startedAt)
    const busy = c.kind === 'busy'
    const ev = queue[0]
    const shot: Shot = {
      kind: c.kind,
      t,
      columns,
      look,
      caption: busy ? speech.text || c.caption : c.caption,
      captionAge: busy ? now - speech.since : t,
      wink: linesFor(look, model, settings.lang, pack).wink[1],
      friends,
    }
    if (busy && ev && ev.at > 0) shot.event = { kind: ev.kind, t: now - ev.at, count: ev.count }
    if (c.kind === 'card') shot.card = card(stats, look.sfw, Math.min(48, columns - 2))
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
    const lines = linesAt(now)
    const sounds: string[] = []
    const due = (cues: readonly (readonly [number, string])[], at: number, played: number) => {
      let n = played
      for (let next = cues[n]; next && now - at >= next[0]; next = cues[++n]) sounds.push(next[1])
      return n
    }
    if (c.kind === 'busy') {
      for (const [minutes, kind] of ESCALATIONS) {
        if (working && now - turnStart >= minutes * 60000 && !visited.has(kind)) {
          visited.add(kind)
          happen(kind)
          if (kind === 'landlord') tally(s => (s.landlords += 1), now)
        }
      }
      let ev = queue[0]
      if (ev && ev.at > 0 && now - ev.at > eventDuration(ev.kind, ev.count)) {
        queue.shift()
        ev = queue[0]
        if (!working) return { done: true, sounds: [] } // the turn ended on a slam
        if (!ev) {
          speech = { text: pick(lines.busy), since: now }
          nextBeat = now
          nextVoice = Math.max(nextVoice, now + 2500)
        }
      }
      if (ev && ev.at === 0) begin(ev, now)
      if (ev) {
        ev.cue = due(eventCues(ev, now), ev.at, ev.cue)
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
    } else {
      cue = due(clipCues(c.kind, now), c.startedAt, cue)
    }
    const cells = band ? encode(frame(shotFor(c, now, band.columns))) : undefined
    return { cells, requestId: band?.requestId, sounds: muted() ? [] : sounds }
  }

  // Each step on its own: one that fails mustn't take the show down with it.
  on('session.start', async ($, e, next) => {
    // A clip by name from sounds/, or a voice pack's file by its path.
    play = name => {
      const file = /[\\/]/.test(name) ? name : `${$.plugin.root}/sounds/${name}.wav`
      void $.process.run(playerArgv(os, file), { timeoutMs: 10000 }).catch(() => {})
    }
    save = () => {
      void $.store.set('stats', stats).catch(() => {})
    }
    toast = text => $.ui.toast(text, { timeoutMs: 6000 })
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
            queue = []
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
    stats = restoreStats(await $.store.get('stats').catch(() => null))
    const windows = await $.env.get('OS').then(v => v === 'Windows_NT', () => false)
    const uname = windows ? null : await $.process.run(['uname'], { timeoutMs: 5000 }).catch(() => null)
    os = windows ? 'windows' : uname?.stdout.trim() === 'Darwin' ? 'mac' : 'linux'
    const env = async (read: Promise<string | undefined>) => read.catch(() => undefined)
    packs = packsFolder(os, {
      config: await env($.env.get('CUCK_CONFIG')),
      xdg: await env($.env.get('XDG_CONFIG_HOME')),
      home: await env($.env.get('HOME')),
      appData: await env($.env.get('APPDATA')),
    })
    if (settings.pack !== null) {
      const name = settings.pack
      pack = await $.fs.read(`${packs}/${name}/pack.json`).then(json => readPack(`${packs}/${name}`, name, json), () => null)
    }
    await $.command
      .register({
        name: 'cuck',
        description: 'Cuck Code: quiet, safe for work, costumes',
        argumentHint: 'stats | badge | quiet | sfw | skin | lang | cast | pack | meetings | trailer',
        immediate: true,
      })
      .catch(() => {})
    await $.command.register({ name: 'cuck-quiet', description: 'Mute or unmute the Cuck Code sounds', immediate: true }).catch(() => {})

    // The last session had her over: she lets herself out as this one starts.
    const session = await $.session.id().catch(() => '')
    const last = await $.store.get('morning').catch(() => null)
    if (typeof last === 'string' && last !== session && current === null) {
      await $.store.delete('morning').catch(() => {})
      const c = startClip('morning', await $.clock.now())
      await update($, clip, () => c)
      kick()
    }
    return next(e)
  })

  on('command.run', { command: 'cuck' }, async ($, e) => {
    const what = e.args.trim().toLowerCase()
    const sfw = settings.sfw || meeting
    if (what === 'stats') {
      // The card shows in the band too, unless Claude's busy in it.
      if (!working && current === null) {
        const c = startClip('card', await $.clock.now())
        await update($, clip, () => c)
        kick()
      }
      return { text: ['Cuck Code', ...summary(stats, sfw).map(l => `  ${l}`)].join('\n') }
    }
    if (what === 'pack' || what.startsWith('pack ')) {
      const name = what.slice(4).trim()
      const installed = await $.fs.list(packs).then(
        entries => entries.filter(f => f.kind === 'dir').map(f => f.name),
        () => [] as string[],
      )
      if (name === '') {
        const list = installed.length > 0 ? installed.join(', ') : 'none yet'
        return { text: `Voice packs: ${list}. Now: ${settings.pack ?? 'none'}.\nAdd one with \`cuck pack add owner/repo\` in a terminal, then /cuck pack <name>.` }
      }
      if (name === 'off' || name === 'none') {
        settings = { ...settings, pack: null }
        pack = null
        await $.store.set('settings', settings)
        return { text: 'No voice pack: the usual lines.' }
      }
      const loaded = /^[\w.-]+$/.test(name)
        ? await $.fs.read(`${packs}/${name}/pack.json`).then(json => readPack(`${packs}/${name}`, name, json), () => null)
        : null
      if (loaded === null) return { text: `No voice pack called "${name}" in ${packs}. Installed: ${installed.join(', ') || 'none'}.` }
      settings = { ...settings, pack: name }
      pack = loaded
      await $.store.set('settings', settings)
      return { text: `Voice pack: ${loaded.name}.` }
    }
    if (what === 'badge') {
      const md = badge(stats, sfw)
      const copied = await $.ui.copy({ text: md }).then(r => r.isCopied, () => false)
      return { text: `${md}\n\n${copied ? 'Copied: paste it into your README.' : 'Paste it into your README.'}` }
    }
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
      turnStart = now
      friends = 0
      visited = new Set()
      queue = []
      spinnerWord = pick(linesAt(now).spinner)
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
        const line = linesAt(now).knock[1]
        happen('knock', { id })
        if (current?.kind === 'busy') begin(queue[0] ?? { kind: 'knock', at: 0, cue: 0, count: 1, id }, now)
        kick()
        // The dialog opens once this returns.
        for (const ms of [150, 700]) $.clock.after(ms, () => $.ui.notice(id, line))
      } catch {
        // the show is never worth a failed permission check
      }
    }
    return verdict
  }).catch(($, e, next) => next(e))

  // A call has run: the one you were asked about, or a test run.
  on('tool.call', async ($, e, next) => {
    const result = await next(e)
    try {
      const said = 'text' in result && typeof result.text === 'string' ? result.text : String(result.result ?? '')
      const refused = result.isError === true && REFUSED.test(said)
      const knock = queue[0]
      if (knock?.kind === 'knock' && knock.id === e.tool_use_id) {
        answer(refused, await $.clock.now())
      } else if (working && e.tool === 'Bash' && !e.run_in_background && TEST_COMMAND.test(e.command) && !refused && !('deny' in result && result.deny)) {
        const failed = result.isError === true
        happen(failed ? 'flop' : 'passed')
        tally(s => (failed ? (s.flops += 1) : (s.passes += 1)), await $.clock.now())
      }
    } catch {
      // the show is never worth a failed tool call
    }
    return result
  }).catch(($, e, next) => next(e))

  // He brought a friend: each agent Claude starts walks in through the door.
  on('agent.spawn', async ($, e, next) => {
    if (working) {
      friends += 1
      happen('friend')
      const thisTurn = friends
      tally(s => {
        s.friends += 1
        s.mostFriends = Math.max(s.mostFriends, thisTurn)
      }, await $.clock.now())
    }
    return next(e)
  }).catch(($, e, next) => next(e))

  // Context compaction: they come out for a breather.
  on('session.compact', async ($, e, next) => {
    if (!e.agentId) {
      tally(s => (s.breathers += 1), await $.clock.now())
      if (working) happen('breather')
      else if (current === null) {
        const c = startClip('breather', await $.clock.now())
        await update($, clip, () => c)
        kick()
      }
    }
    return next(e)
  }).catch(($, e, next) => next(e))

  on('turn.complete', async ($, e, next) => {
    const result = await next(e)
    if (e.agentId) {
      return result
    }
    working = false
    model = e.usage?.model ?? model
    const now = await $.clock.now()
    if (e.isAborted) {
      // Turned down at the door: it slams, and that's the end of it.
      if (queue[0]?.kind === 'knock') answer(true, now)
      if (queue[0]?.kind === 'slam' && current?.kind === 'busy') {
        queue = queue.slice(0, 1)
        kick()
        return result
      }
      current = null
      queue = []
      await update($, clip, () => null)
      return result
    }
    queue = []
    tally(s => {
      s.turns += 1
      s.ms += e.durationMs
      s.longest = Math.max(s.longest, e.durationMs)
      if (e.reason === 'refusal') s.refused += 1
    }, now, new Date(now).getHours())
    // Claude said no: the door stays shut. Otherwise out he comes.
    const c = startClip(e.reason === 'refusal' ? 'refused' : 'leave', now)
    await update($, clip, () => c)
    kick()
    if (!muted() && c.kind === 'leave') play('ding')
    // She stays the night; the next session sees her out.
    void $.session.id().then(id => $.store.set('morning', id)).catch(() => {})
    return result
  })

  // With /cuck trailer on, Claude signs what it commits, and its pull requests.
  on('attribution.text', async ($, e, next) => {
    const { text } = await next(e)
    if (!settings.trailer || (e.kind !== 'commit' && e.kind !== 'pr')) return { text }
    const sign = e.kind === 'commit' ? 'Cucked-By: Claude' : 'Cucked with [Cuck Code](https://github.com/ruskaruma/cuck-code)'
    return { text: text.trim() === '' ? sign : `${text.trimEnd()}${e.kind === 'commit' ? '\n' : '\n\n'}${sign}` }
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
    next({ ...e, props: { ...e.props, word: linesFor({ skin: 'none', sfw: settings.sfw || meeting, cast: settings.cast }).done } }),
  )
}
