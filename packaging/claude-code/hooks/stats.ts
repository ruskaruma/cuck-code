// What you've sat through, kept across sessions: for /cuck stats, the badge
// and the achievements.

export type Stats = {
  turns: number // turns you watched
  ms: number // time in the chair, all told
  longest: number // the longest turn
  friends: number // helper agents he brought
  mostFriends: number // the most in one turn
  refused: number // turns Claude refused
  yeses: number // permissions you gave
  flops: number // test runs that failed
  passes: number // and passed
  breathers: number // compactions
  landlords: number // turns long enough for the landlord to come by
  earned: Record<string, number> // achievement id: when
}

export const EMPTY: Stats = {
  turns: 0, ms: 0, longest: 0, friends: 0, mostFriends: 0, refused: 0,
  yeses: 0, flops: 0, passes: 0, breathers: 0, landlords: 0, earned: {},
}

// Stats as saved, checked field by field.
export function restoreStats(saved: unknown): Stats {
  const s: Stats = { ...EMPTY, earned: {} }
  if (typeof saved !== 'object' || saved === null) return s
  const o = saved as Record<string, unknown>
  for (const key of Object.keys(EMPTY) as (keyof Stats)[]) {
    const v = o[key]
    if (key === 'earned') {
      if (typeof v === 'object' && v !== null) {
        for (const [id, at] of Object.entries(v)) if (typeof at === 'number') s.earned[id] = at
      }
    } else if (typeof v === 'number' && Number.isFinite(v) && v >= 0) s[key] = v
  }
  return s
}

type Achievement = { id: string; name: string; how: string; got: (s: Stats, hour: number) => boolean }

export const ACHIEVEMENTS: readonly Achievement[] = [
  { id: 'first', name: 'First Time', how: 'you watched Claude do it', got: s => s.turns >= 1 },
  { id: 'century', name: 'Century', how: 'you watched 100 turns', got: s => s.turns >= 100 },
  { id: 'marathon', name: 'Marathon', how: 'you sat through a 30-minute turn', got: s => s.longest >= 30 * 60000 },
  { id: 'committed', name: 'Committed', how: '24 hours in the chair, all told', got: s => s.ms >= 24 * 3600000 },
  { id: 'fullhouse', name: 'Full House', how: 'he brought 5 friends at once', got: s => s.mostFriends >= 5 },
  { id: 'nightshift', name: 'Night Shift', how: 'a turn finished between 2 and 5 in the morning', got: (_, hour) => hour >= 2 && hour < 5 },
  { id: 'respect', name: 'Respect', how: 'Claude said no', got: s => s.refused >= 1 },
  { id: 'flop', name: 'Performance Issues', how: 'the tests failed while you listened', got: s => s.flops >= 1 },
  { id: 'screamer', name: 'Screamer', how: 'the tests passed 10 times', got: s => s.passes >= 10 },
  { id: 'consent', name: 'Enthusiastic Consent', how: 'you said yes 25 times', got: s => s.yeses >= 25 },
  { id: 'breather', name: 'Second Wind', how: 'they stopped for a breather', got: s => s.breathers >= 1 },
  { id: 'landlord', name: 'Family Building', how: 'the landlord came by', got: s => s.landlords >= 1 },
]

// The achievements s has just earned, marked earned at `now`. `hour` is the
// hour a turn finished, or -1.
export function award(s: Stats, now: number, hour = -1): Achievement[] {
  const fresh = ACHIEVEMENTS.filter(a => s.earned[a.id] === undefined && a.got(s, hour))
  for (const a of fresh) s.earned[a.id] = now
  return fresh
}

// 45s, 14m 03s, 3h 12m.
export function span(ms: number): string {
  const secs = Math.round(ms / 1000)
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  if (mins < 60) return `${mins}m ${String(secs % 60).padStart(2, '0')}s`
  return `${Math.floor(mins / 60)}h ${String(mins % 60).padStart(2, '0')}m`
}

const plural = (n: number, one: string, many = `${one}s`) => `${n} ${n === 1 ? one : many}`

// The stats in a few lines, safe for work or not.
export function summary(s: Stats, sfw: boolean): string[] {
  const got = ACHIEVEMENTS.filter(a => s.earned[a.id] !== undefined)
  return [
    sfw
      ? `${plural(s.turns, 'turn')} shipped · ${span(s.ms)} pairing`
      : `cucked ${plural(s.turns, 'time')} · ${span(s.ms)} in the chair`,
    `longest ${span(s.longest)} · ${sfw ? 'reviewers' : 'friends'} over: ${s.friends}`,
    sfw ? `approved ${s.yeses} · declined ${s.refused}` : `you said yes ${s.yeses} · claude said no ${s.refused}`,
    `tests: ${s.passes} passed, ${s.flops} ${sfw ? 'failed' : 'flopped'}`,
    `${got.length}/${ACHIEVEMENTS.length} achievements${got.length > 0 ? `: ${got.map(a => a.name).join(', ')}` : ''}`,
  ]
}

// A run of text in one colour.
export type Segment = readonly [text: string, colour: number]

const BORDER = 0xff5a6e
const TITLE = 0xffd76a
const TEXT = 0xfbe9d7
const DIMMED = 0x9a8f88

// The stats card for the band: rows of coloured text, `width` cells wide,
// boxed, to screenshot.
export function card(s: Stats, sfw: boolean, width: number): Segment[][] {
  const inner = width - 4
  const fit = (text: string) => (text.length > inner ? `${text.slice(0, inner - 1)}…` : text.padEnd(inner))
  const title = ' CUCK CODE '
  const rows: Segment[][] = [[['╭─', BORDER], [title, TITLE], ['─'.repeat(Math.max(0, width - title.length - 3)) + '╮', BORDER]]]
  for (const line of summary(s, sfw)) rows.push([['│ ', BORDER], [fit(line), TEXT], [' │', BORDER]])
  rows.push([['│ ', BORDER], [fit('github.com/ruskaruma/cuck-code'), DIMMED], [' │', BORDER]])
  rows.push([['╰' + '─'.repeat(width - 2) + '╯', BORDER]])
  return rows
}

// A README badge: the time in the chair, linking here.
export function badge(s: Stats, sfw: boolean): string {
  const label = sfw ? 'paired with claude' : 'cucked for'
  const esc = (t: string) => encodeURIComponent(t.replace(/-/g, '--').replace(/_/g, '__').replace(/ /g, '_'))
  const time = span(s.ms)
  return `[![${label} ${time}](https://img.shields.io/badge/${esc(label)}-${esc(time)}-ff5a6e)](https://github.com/ruskaruma/cuck-code)`
}
