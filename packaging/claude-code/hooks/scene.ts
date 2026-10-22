// The strip drawn above the prompt: a door, her, and him (Claude). Pure
// drawing code. A frame is a grid of pixels, two to a terminal cell (upper
// and lower half blocks), plus one row of caption text underneath.

import type { EventKind, Kind } from '../types'
import type { Segment } from './stats'

export type Skin = 'none' | 'halloween'
// How the scene is dressed: the season's costumes, and whether it's safe for work.
export type Look = { skin: Skin; sfw: boolean }

// One frame: the clip, how far into it, and anything interrupting the busy loop.
export type Shot = {
  kind: Kind
  t: number
  columns: number
  caption: string
  captionAge: number // how long the caption has been up (it types itself out)
  look: Look
  wink: string // what he says as he winks
  friends: number // how many friends he brought this turn: they leave after him
  event?: { kind: EventKind; t: number; count: number }
  card?: Segment[][] // the stats card, drawn as text
}

export const ROWS = 9 // 8 rows of pixels (16 high) and the caption row
export const MAX_COLUMNS = 64
export const MIN_COLUMNS = 40
// busy loops until Claude finishes. In leave, he winks at WINK_AT.
export const DURATION: Record<Kind, number> = { enter: 1800, busy: Infinity, leave: 5000, breather: 5200, refused: 3500, morning: 4600, card: 15000 }
export const WINK_AT = 1900
const MAX_FRIENDS = 3 // drawn at once; there may be more

// How long each interruption of the busy loop lasts: a knock, until you
// answer; friends, by how many walk in.
export function eventDuration(kind: EventKind, count = 1): number {
  switch (kind) {
    case 'knock':
      return Infinity
    case 'friend':
      return 1300 + Math.min(count, MAX_FRIENDS) * 350
    case 'breather':
      return DURATION.breather
  }
  return { yes: 900, slam: 1000, pizza: 3800, neighbour: 2600, landlord: 4200, flop: 2800, passed: 2400 }[kind]
}

const PX_H = 16
const DEFAULT = 0x01000000 // the terminal's own colour
const CAPTION_X = 2

const palette: Record<string, number> = {
  // her
  h: 0x2b1a14, s: 0xf1c3a0, r: 0xe0444d, k: 0x7d1820,
  // him: hair, skin, white tank top, its name tag, jeans, shoes, eyes and grin
  g: 0x4a2c1a, t: 0xd9a27c, w: 0xf2efe9, T: 0xe0444d, j: 0x3a5a8c, n: 0x2a2a2a, e: 0x1b1210, m: 0x7d1820,
  // the shine in his eyes, bright and dimmer as it twinkles
  W: 0xffffff, V: 0xbfe6ff,
  // the door: frame, panel, knob, the dark room and its glow
  D: 0x5a3a22, P: 0x8c5e34, o: 0xffd76a, I: 0x1a0d10, G: 0x6b1f37,
  // a heart, a sparkle, the sock on the knob, the do-not-disturb sign, a green check
  H: 0xff5a6e, '*': 0xffd76a, x: 0xf2efe9, X: 0xe0444d, S: 0xf2efe9, Q: 0x62d26f,
  // Halloween: the witch's hat and its buckle, a jack-o'-lantern, a bat, his fangs
  B: 0x4b3270, u: 0xf0b840, O: 0xf08a24, y: 0xffd34d, L: 0x4c8a2f, b: 0x9a6ad6, F: 0xffffff,
  // his friends' shirts and hair
  E: 0x4caf50, J: 0x3f7fd6, K: 0xe0b13a, l: 0xd9b45a, v: 0x231a17, f: 0xb5562a,
  // the pizza guy's red, the landlord's suit, a water bottle, the neighbour's
  // wall, the morning sun, the no-entry sign, a heart gone grey
  R: 0xd8402f, A: 0x7a7f87, a: 0x4a4d52, Z: 0x5ab0ff, Y: 0xa39484, U: 0xffc94a, N: 0xc62828, c: 0x8a8a8a,
}

// Costumes recolour the cast: for Halloween she's a witch in purple, and he's
// a vampire, pale with slicked hair, red eyes and a cape lined in red.
const costumes: Record<Skin, Record<string, number>> = {
  none: {},
  halloween: { r: 0x6f3fa0, k: 0x2b2238, t: 0xe6ddd5, g: 0x2e2a3a, w: 0x3d3350, T: 0xb3121f, e: 0xc0182a, V: 0xff9a9a },
}

// Her, facing right, plus legs standing and mid-step.
const her = [
  '...hhh..',
  '..hhhhh.',
  '..hhsss.',
  '..hhsss.',
  '..hh.ss.',
  '..h..s..',
  '...rrrr.',
  '..srrrrs',
  '..srrrr.',
  '...rrr..',
  '..rrrrr.',
]
const herLegs = [
  ['...s.s..', '...s.s..', '..kk.kk.'],
  ['...s..s.', '..s...s.', '.kk...kk'],
]
// The witch's hat: two rows above her head, and its brim over her hair.
const hat = ['.....B..', '....BB..', '.BBBuBBB']

// Him, facing left as he walks out, plus legs standing and mid-step.
const him = [
  '..gggg..',
  '.gggggg.',
  '.ttttgg.',
  '.etttgg.',
  '.tttttg.',
  '..ttt...',
  '..tt....',
  '.wwwwww.',
  'twwwwwwt',
  'twwTwwwt',
  'twwwwwwt',
  '.jjjjjj.',
]
const himLegs = [
  ['.jj..jj.', '.jj..jj.', '.nn..nn.'],
  ['.jj...jj', 'jj....jj', 'nn....nn'],
]

// Him, turned to face you. Rows 3-6 are his face: big shiny eyes, then the
// wink and grin. W marks the shine in each eye.
const himFront = [
  '..gggggg..',
  '.gggggggg.',
  '.gttttttg.',
  '.tWettWet.',
  '.teetteet.',
  '.tttttttt.',
  '..tmmmmt..',
  '...tttt...',
  'twwwwwwwwt',
  'twwwwwTwwt',
  'twwwwwwwwt',
  '..jjjjjj..',
  '..jj..jj..',
  '..jj..jj..',
  '..nn..nn..',
]
const faces: Record<'open' | 'wink', [string, string, string, string]> = {
  open: ['.tWettWet.', '.teetteet.', '.tttttttt.', '..tmmmmt..'],
  wink: ['.tWettttt.', '.teetteee.', '.tmttttmt.', '..tmmmmt..'],
}
const sparkle = ['.*.', '*.*', '.*.']

const heart = ['.H.H.', 'HHHHH', '.HHH.', '..H..']
const greyHeart = ['.c.c.', 'ccccc', '.ccc.', '..c..']
const bat = ['b...b', 'bb.bb', '.bbb.']
const check = ['....Q', '...Q.', 'Q.Q..', '.Q...']
const sock = ['xx.', 'XX.', 'xx.', 'xxx']
const sign = ['.S.', 'SSS', 'SXS', 'SSS']
const noEntry = ['.NNN.', 'NxxxN', '.NNN.']
const pumpkin = ['..L..', '.OOO.', 'OyOyO', 'OOOOO']
const pizza = ['PPPP', 'DDDD']
const clipboard = ['xx', 'xx', 'aa']
const sun = ['..UUUU..', '.UUUUUU.', 'UUUUUUUU', 'UUUUUUUU', '.UUUUUU.', '..UUUU..']

// Who else walks through the door, as him in other clothes: his friends,
// the pizza guy in red, the landlord in a grey suit with a tie.
const FRIENDS = [{ g: 'l', w: 'E', T: 'E' }, { g: 'v', w: 'J', T: 'J' }, { g: 'f', w: 'K', T: 'K' }]
const PIZZA_GUY = { g: 'R', w: 'R', T: 'x' }
const LANDLORD = { g: 't', w: 'A', T: 'J', j: 'a' }

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))
const ramp = (t: number, from: number, to: number) => clamp((t - from) / (to - from), 0, 1)
const mirror = (rows: readonly string[]) => rows.map(r => [...r].reverse().join(''))
const recolour = (rows: readonly string[], swap: Record<string, string>) => rows.map(r => r.replace(/./g, ch => swap[ch] ?? ch))

class Canvas {
  w: number
  px: Int32Array
  colours: Record<string, number>
  constructor(w: number, skin: Skin) {
    this.w = w
    this.px = new Int32Array(w * PX_H).fill(-1)
    this.colours = { ...palette, ...costumes[skin] }
  }
  set(x: number, y: number, c: number) {
    if (x >= 0 && x < this.w && y >= 0 && y < PX_H) this.px[y * this.w + x] = c
  }
  sprite(rows: readonly string[], x0: number, y0: number, fade = 1, clipRight = Infinity) {
    rows.forEach((row, y) => {
      for (let i = 0; i < row.length; i++) {
        const c = this.colours[row[i] ?? '.']
        if (c !== undefined && x0 + i < clipRight) this.set(x0 + i, y0 + y, dim(c, fade))
      }
    })
  }
}

function dim(c: number, f: number): number {
  if (f >= 1) return c
  const r = Math.round(((c >> 16) & 255) * f)
  const g = Math.round(((c >> 8) & 255) * f)
  const b = Math.round((c & 255) * f)
  return (r << 16) | (g << 8) | b
}

// Her, walking right (or left), mid-step or not; for Halloween in her hat;
// the morning after, barefoot with her shoes in her hand. Drawn from the top
// of her hat, two pixels above her head.
function drawHer(cv: Canvas, look: Look, x: number, step: number, opts: { left?: boolean; clipRight?: number; shoes?: boolean; towel?: boolean } = {}) {
  const legs = (herLegs[step % 2] ?? []).map(r => (opts.shoes ? r.replace(/k/g, 's') : r))
  const rows = her.concat(legs)
  if (opts.towel) rows[6] = '..xxxxx.'
  if (look.skin === 'halloween') rows[0] = hat[2] ?? ''
  const drawn = opts.left ? mirror(rows) : rows
  const clipRight = opts.clipRight ?? Infinity
  if (look.skin === 'halloween') cv.sprite(opts.left ? mirror(hat.slice(0, 2)) : hat.slice(0, 2), x, 0, 1, clipRight)
  cv.sprite(drawn, x, 2, 1, clipRight)
  if (opts.shoes) cv.sprite(['kk'], opts.left ? x - 1 : x + 7, 10)
}

// A man in profile, walking: him, or anyone in his clothes. Facing left by
// default.
function drawMan(cv: Canvas, x: number, step: number, opts: { swap?: Record<string, string>; right?: boolean; clipRight?: number; y?: number } = {}) {
  let rows = him.concat(himLegs[step % 2] ?? [])
  if (opts.swap) rows = recolour(rows, opts.swap)
  cv.sprite(opts.right ? mirror(rows) : rows, x, opts.y ?? 1, 1, opts.clipRight ?? Infinity)
}

// open: 0 closed, 1 wide open. shake: the door rattles in its frame. The knob
// carries a sock, or the do-not-disturb sign when it's safe for work.
function drawDoor(cv: Canvas, look: Look, x0: number, open: number, shake: number, hanging: boolean) {
  const W = 10
  const panel = doorPanel(open)
  if (look.skin === 'halloween') cv.sprite(pumpkin, x0 - 6, 12)
  for (let y = 0; y < PX_H; y++) {
    for (let x = 0; x < W; x++) {
      let ch: string
      if (y === 0 || x === 0 || x === W - 1) ch = 'D'
      else if (x <= panel) {
        const inset = y === 2 || y === 6 || y === 9 || y === 13
        ch = (x === 2 || x === panel - 1) && y > 1 && y < 14 ? 'D' : inset && x > 1 && x < panel ? 'D' : 'P'
        if (y === 8 && x === panel - 1 && panel > 3) ch = 'o'
      } else ch = y < 4 ? 'G' : 'I'
      cv.set(x0 + x + (x > 0 && x <= panel ? shake : 0), y, cv.colours[ch] ?? 0)
    }
  }
  if (hanging && open === 0) cv.sprite(look.sfw ? sign : sock, x0 + 6 + shake, 9)
}

// Columns of door panel still across the doorway.
const doorPanel = (open: number) => Math.round((1 - open) * 7) + 1

// What drifts out of the room: hearts, bats among them for Halloween, and
// green checks when it's safe for work.
function floater(look: Look, n: number): readonly string[] {
  if (look.sfw) return check
  return look.skin === 'halloween' && n % 2 === 1 ? bat : heart
}

function drawFloaters(cv: Canvas, look: Look, doorX: number, t: number, period: number) {
  for (const [n, phase, dx] of [[0, 0, -7], [1, period / 2, -13]] as const) {
    const cycle = Math.floor((t + phase) / period)
    const p = ((t + phase) % period) / (period * 0.7)
    if (p < 1) cv.sprite(floater(look, n + cycle), doorX + dx, Math.round(9 - p * 8), 1 - p * 0.7)
  }
}

// His head and a hand round the door, asking you for a yes. out: 0 still
// inside, 1 all the way out.
function drawPeek(cv: Canvas, doorX: number, open: number, out: number, t: number) {
  const edge = doorX + doorPanel(open)
  const x = edge - 3 + Math.round((1 - out) * 4)
  const head = him.slice(0, 7)
  // Big pleading eyes: the shine comes and goes.
  if (Math.floor(t / 300) % 2 === 0) head[3] = '.Wtttgg.'
  cv.sprite(head, x, 3 + (Math.floor(t / 600) % 2), 1, doorX + 9) // never past the door frame
  if (out > 0.5) cv.sprite(['tt'], edge, 10)
}

// Something walks up from the left, waits at the door, then walks back off:
// where it is at time t, and whether it's facing the door.
function visit(t: number, doorX: number, arrive: number, leave: number, end: number): { x: number; right: boolean; moving: boolean } {
  const stop = doorX - 10
  if (t < arrive) return { x: Math.round(-9 + (t / arrive) * (stop + 9)), right: true, moving: true }
  if (t < leave) return { x: stop, right: true, moving: false }
  return { x: Math.round(stop - ramp(t, leave, end) * (stop + 10)), right: false, moving: true }
}

// The busy loop, interrupted by one event t milliseconds in.
function drawEvent(cv: Canvas, look: Look, doorX: number, kind: EventKind, t: number, count: number) {
  const step = Math.floor(t / 120)
  switch (kind) {
    case 'knock': {
      // He leans out to ask you; the room goes quiet while he waits.
      const open = ramp(t, 0, 250) * 0.5
      drawDoor(cv, look, doorX, open, 0, false)
      drawPeek(cv, doorX, open, ramp(t, 150, 450), t)
      return
    }
    case 'yes': {
      // Back in he goes, and the door shuts behind him.
      const open = 0.5 * (1 - ramp(t, 250, 500))
      drawDoor(cv, look, doorX, open, 0, open === 0)
      if (t < 400) drawPeek(cv, doorX, open, 1 - ramp(t, 0, 300), t)
      return
    }
    case 'slam':
      // Shut in his face: the whole door shakes.
      drawDoor(cv, look, doorX, 0, t < 450 ? (Math.floor(t / 45) % 3) - 1 : 0, true)
      return
    case 'friend': {
      // His friends file in, one behind the other.
      const n = Math.min(count, MAX_FRIENDS)
      const last = 900 + (n - 1) * 350
      const open = ramp(t, 600, 750) - ramp(t, last + 150, last + 300)
      drawDoor(cv, look, doorX, open, 0, open === 0)
      for (let i = 0; i < n; i++) {
        const x = Math.round(-9 + ramp(t, i * 350, 900 + i * 350) * (doorX + 10))
        if (x < doorX + 9) drawMan(cv, x, step + i, { swap: FRIENDS[i], right: true, clipRight: open > 0.5 ? doorX + 9 : doorX + 1 })
      }
      return
    }
    case 'breather': {
      // They come out with towels and water, catch their breath, and go back in.
      const open = ramp(t, 0, 250) - ramp(t, 4700, 4950)
      drawDoor(cv, look, doorX, open, 0, open === 0)
      const herX = doorX - 9
      const hisX = doorX - 20
      const out = (from: number, to: number, x: number) =>
        Math.round(doorX + 1 - (ramp(t, from, to) - ramp(t, 3800 + from - 300, 4400 + from - 300)) * (doorX + 1 - x))
      const pant = Math.floor(t / 400) % 2
      if (t > 300 && t < 4700) {
        const hx = out(500, 1100, hisX)
        drawMan(cv, hx, t < 1100 || t > 3800 ? step : 0, { swap: { g: 'x' }, right: t > 3800, y: 1 + (t > 1100 && t < 3800 ? pant : 0) })
        if (t > 1100 && t < 3800) cv.sprite(['Z', 'Z', 'Z'], hx - 1, 8)
        drawHer(cv, look, out(300, 900, herX), t < 900 || t > 3800 ? step : 0, { left: t <= 3800, towel: true })
      }
      return
    }
    case 'pizza': {
      // A delivery: he walks up, knocks, hands the box through the door, and goes.
      const v = visit(t, doorX, 1100, 2800, 3800)
      const open = (ramp(t, 2400, 2550) - ramp(t, 2800, 2950)) * 0.3
      drawDoor(cv, look, doorX, open, 0, open === 0)
      drawMan(cv, v.x, v.moving ? step : 0, { swap: PIZZA_GUY, right: v.right })
      if (t < 2650) cv.sprite(pizza, v.x + 7 + Math.round(ramp(t, 2450, 2650) * 3), 9)
      return
    }
    case 'landlord': {
      // He walks up with his clipboard, taps his foot, and leaves.
      const v = visit(t, doorX, 1200, 3000, 4200)
      drawDoor(cv, look, doorX, 0, 0, true)
      drawMan(cv, v.x, v.moving ? step : Math.floor(t / 300), { swap: LANDLORD, right: v.right })
      cv.sprite(clipboard, v.right ? v.x + 7 : v.x - 1, 9)
      return
    }
    case 'neighbour': {
      // Banging on the wall: three thumps, the plaster jumping with each.
      drawDoor(cv, look, doorX, 0, 0, true)
      const bang = [0, 300, 600].some(at => t >= at && t < at + 120)
      for (let y = 0; y < PX_H; y++) for (let x = 0; x < 3; x++) cv.set(x + (bang ? 1 : 0), y, cv.colours.Y ?? 0)
      if (bang) for (const y of [5, 8, 11]) cv.sprite(['cc'], 5, y)
      return
    }
    case 'flop':
      // Performance issues: the door goes still and a grey heart sinks.
      drawDoor(cv, look, doorX, 0, 0, true)
      cv.sprite(look.sfw ? recolour(check, { Q: 'c' }) : greyHeart, doorX - 8, Math.round(1 + ramp(t, 0, 2400) * 10), 1 - ramp(t, 1800, 2700))
      return
    case 'passed': {
      // Tests passed: the door shakes hard and hearts pour out.
      drawDoor(cv, look, doorX, 0, Math.floor(t / 60) % 2, true)
      for (let i = 0; i < 6; i++) {
        const p = ramp(t, i * 180, i * 180 + 1100)
        if (p > 0 && p < 1) cv.sprite(floater(look, i), doorX - 8 - ((i * 7) % 26), Math.round(11 - p * 11), 1 - p * 0.6)
      }
    }
  }
}

// One frame, as Raster cells: row-major [codePoint, foreground, background].
export function frame(shot: Shot): Uint32Array {
  const { kind, t, columns, look } = shot
  let { caption, captionAge } = shot
  const cv = new Canvas(columns, look.skin)
  const doorX = columns - 13
  const step = Math.floor(t / 120)
  let fade = 1
  let shown = Math.floor(captionAge / 22) // caption characters shown

  if (kind === 'enter') {
    // She walks in from the left and through the door; it shuts, a sock goes on the knob.
    const open = ramp(t, 650, 850) - ramp(t, 1150, 1350)
    drawDoor(cv, look, doorX, open, 0, t > 1350)
    const x = Math.round(-9 + ramp(t, 0, 1100) * (doorX + 12))
    if (t < 1150) drawHer(cv, look, x, step, { clipRight: open > 0.5 ? Infinity : doorX + 1 })
    if (t > 1300) {
      const rise = Math.floor(ramp(t, 1300, 1800) * 6)
      cv.sprite(floater(look, 1), doorX - 7, 7 - rise, 1 - ramp(t, 1550, 1800))
    }
    fade = 1 - ramp(t, 1650, 1800)
    shown = Math.floor((captionAge - 150) / 22)
  } else if (kind === 'busy') {
    const ev = shot.event
    if (ev) drawEvent(cv, look, doorX, ev.kind, ev.t, ev.count)
    else {
      // Behind the closed door, sock on the knob: it rattles in bursts, hearts drift out.
      drawDoor(cv, look, doorX, 0, t % 800 < 350 ? Math.floor(t / 90) % 2 : 0, true)
      drawFloaters(cv, look, doorX, t, 1300)
    }
  } else if (kind === 'breather') {
    // A breather with no turn running (a /compact): the same as in one.
    drawEvent(cv, look, doorX, 'breather', t, 1)
    fade = 1 - ramp(t, 4950, 5200)
  } else if (kind === 'refused') {
    // Claude said no: the door stays shut, a no-entry sign on it.
    drawDoor(cv, look, doorX, 0, 0, false)
    cv.sprite(noEntry, doorX + 3, 5)
    fade = 1 - ramp(t, 3000, 3500)
  } else if (kind === 'card') {
    // Your stats, boxed, and him beside them, winking at you.
    const width = shot.card?.[0]?.reduce((n, [text]) => n + text.length, 0) ?? 0
    if (columns >= width + 12) {
      const rows = himFront.slice()
      ;[rows[3], rows[4], rows[5], rows[6]] = Math.floor(t / 1400) % 3 === 2 ? faces.wink : faces.open
      if (look.skin === 'halloween') rows[6] = '..tFmmFt..'
      cv.sprite(Math.floor(t / 150) % 3 === 2 ? rows.map(r => r.replace(/W/g, 'V')) : rows, width + 2, 1)
    }
    fade = 1 - ramp(t, 14000, 15000)
  } else if (kind === 'morning') {
    // The morning after: the sun comes up and she lets herself out, shoes in hand.
    cv.sprite(sun, 4, Math.round(14 - ramp(t, 0, 2500) * 12), 0.9)
    const open = ramp(t, 300, 600) - ramp(t, 1900, 2200)
    drawDoor(cv, look, doorX, open, 0, false)
    if (t > 450) drawHer(cv, look, Math.round(doorX + 1 - ramp(t, 600, 3800) * (doorX + 12)), step, { left: true, shoes: true })
    fade = 1 - ramp(t, 4100, 4600)
  } else {
    // A last rattle; he walks out, stops, turns to you and winks. His friends
    // follow him out and walk off past him.
    const friends = Math.min(shot.friends, MAX_FRIENDS)
    const shake = t < 550 ? Math.floor(t / 60) % 2 : 0
    const closeAt = friends > 0 ? 1600 + (friends - 1) * 450 + 450 : 1700
    const open = ramp(t, 550, 750) - ramp(t, closeAt, closeAt + 200)
    drawDoor(cv, look, doorX, open, shake, t < 550)
    if (t < 600) drawFloaters(cv, look, doorX, t, 600)
    const stopX = Math.max(2, doorX - 22)
    for (let i = 0; i < friends; i++) {
      const from = 1600 + i * 450
      if (t >= from && t < from + 1300) drawMan(cv, Math.round(doorX + 1 - ramp(t, from, from + 1300) * (doorX + 11)), step + i, { swap: FRIENDS[i] })
    }
    if (t >= 700 && t < 1500) {
      drawMan(cv, Math.round(doorX + 1 - ramp(t, 700, 1500) * (doorX + 1 - stopX)), step)
    } else if (t >= 1500) {
      const winking = t > WINK_AT
      const rows = himFront.slice()
      ;[rows[3], rows[4], rows[5], rows[6]] = winking ? faces.wink : faces.open
      // A vampire grins with his fangs.
      if (look.skin === 'halloween') rows[6] = '..tFmmFt..'
      // The shine twinkles.
      const twinkle = Math.floor(t / 150) % 3 === 2
      cv.sprite(twinkle ? rows.map(r => r.replace(/W/g, 'V')) : rows, stopX - 1, 1)
      if (Math.floor(t / 180) % 2 === 0) cv.sprite(sparkle, winking ? stopX + 9 : stopX - 4, winking ? 1 : 2)
    }
    fade = 1 - ramp(t, 4500, 5000)
    if (t > WINK_AT) {
      // Typed out as slowly as he says it.
      caption = shot.wink
      captionAge = t - WINK_AT
      shown = 8 + Math.floor(captionAge / 95)
    } else {
      shown = Math.floor((captionAge - 300) / 22)
    }
  }

  // Fold pixel pairs into half-block cells; the last row is the caption.
  const words = new Uint32Array(columns * ROWS * 3)
  let i = 0
  for (let row = 0; row < ROWS - 1; row++) {
    for (let x = 0; x < columns; x++) {
      const top = cv.px[row * 2 * columns + x] ?? -1
      const bot = cv.px[(row * 2 + 1) * columns + x] ?? -1
      if (top < 0 && bot < 0) words.set([0x20, DEFAULT, DEFAULT], i)
      else if (bot < 0) words.set([0x2580, dim(top, fade), DEFAULT], i)
      else if (top < 0) words.set([0x2584, dim(bot, fade), DEFAULT], i)
      else words.set([0x2580, dim(top, fade), dim(bot, fade)], i)
      i += 3
    }
  }
  // The card's text over the pixels, cell by cell.
  shot.card?.forEach((segments, row) => {
    let x = 0
    for (const [text, colour] of segments) {
      for (const ch of text) {
        if (x < columns && row < ROWS - 1) words.set([ch.codePointAt(0) ?? 0x20, dim(colour, fade), DEFAULT], (row * columns + x) * 3)
        x += 1
      }
    }
  })
  const text = caption.slice(0, clamp(shown, 0, columns - CAPTION_X - 1))
  for (let x = 0; x < columns; x++) {
    const ch = x >= CAPTION_X && x - CAPTION_X < text.length ? text.charCodeAt(x - CAPTION_X) : 0x20
    words.set([ch, dim(0xfbe9d7, fade * 0.8), DEFAULT], i)
    i += 3
  }
  return words
}

// Raster cells are base64 of little-endian u32s.
export function encode(words: Uint32Array): string {
  const bytes = new Uint8Array(words.buffer, words.byteOffset, words.byteLength)
  const withBase64 = bytes as Uint8Array & { toBase64?: () => string }
  if (typeof withBase64.toBase64 === 'function') return withBase64.toBase64()
  let s = ''
  for (let j = 0; j < bytes.length; j += 0x8000) s += String.fromCharCode(...bytes.subarray(j, j + 0x8000))
  return btoa(s)
}

export const pick = <T,>(xs: readonly T[]): T => xs[Math.floor(Math.random() * xs.length)] as T
