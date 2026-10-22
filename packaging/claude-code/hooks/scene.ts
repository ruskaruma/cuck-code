// The strip drawn above the prompt: a door, her, and him (Claude). Pure
// drawing code. A frame is a grid of pixels, two to a terminal cell (upper
// and lower half blocks), plus one row of caption text underneath.

import type { EventKind, Kind } from '../types'

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
  event?: { kind: EventKind; t: number }
}

export const ROWS = 9 // 8 rows of pixels (16 high) and the caption row
export const MAX_COLUMNS = 64
export const MIN_COLUMNS = 40
// busy loops until Claude finishes. In leave, he winks at WINK_AT.
export const DURATION: Record<Kind, number> = { enter: 1800, busy: Infinity, leave: 5000 }
export const WINK_AT = 1900
// How long each interruption of the busy loop lasts; a knock, until you answer.
export const EVENT_DURATION: Record<EventKind, number> = { knock: Infinity, yes: 900, slam: 1000 }

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
const bat = ['b...b', 'bb.bb', '.bbb.']
const check = ['....Q', '...Q.', 'Q.Q..', '.Q...']
const sock = ['xx.', 'XX.', 'xx.', 'xxx']
const sign = ['.S.', 'SSS', 'SXS', 'SSS']
const pumpkin = ['..L..', '.OOO.', 'OyOyO', 'OOOOO']

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))
const ramp = (t: number, from: number, to: number) => clamp((t - from) / (to - from), 0, 1)

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

// Her, walking right; for Halloween in her hat.
function drawHer(cv: Canvas, look: Look, x: number, step: number, clipRight: number) {
  const rows = her.concat(herLegs[step % 2] ?? [])
  if (look.skin === 'halloween') {
    cv.sprite(hat.slice(0, 2), x, 0, 1, clipRight)
    rows[0] = hat[2] ?? ''
  }
  cv.sprite(rows, x, 2, 1, clipRight)
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

// One frame, as Raster cells: row-major [codePoint, foreground, background].
export function frame(shot: Shot): Uint32Array {
  const { kind, t, columns, look } = shot
  let { caption, captionAge } = shot
  const cv = new Canvas(columns, look.skin)
  const doorX = columns - 13
  let fade = 1
  let shown = 0 // caption characters shown

  if (kind === 'enter') {
    // She walks in from the left and through the door; it shuts, a sock goes on the knob.
    const open = ramp(t, 650, 850) - ramp(t, 1150, 1350)
    drawDoor(cv, look, doorX, open, 0, t > 1350)
    const x = Math.round(-9 + ramp(t, 0, 1100) * (doorX + 12))
    if (t < 1150) drawHer(cv, look, x, Math.floor(t / 120), open > 0.5 ? Infinity : doorX + 1)
    if (t > 1300) {
      const rise = Math.floor(ramp(t, 1300, 1800) * 6)
      cv.sprite(floater(look, 1), doorX - 7, 7 - rise, 1 - ramp(t, 1550, 1800))
    }
    fade = 1 - ramp(t, 1650, 1800)
    shown = Math.floor((captionAge - 150) / 22)
  } else if (kind === 'busy') {
    const ev = shot.event
    if (ev?.kind === 'knock') {
      // He leans out to ask you; the room goes quiet while he waits.
      const open = ramp(ev.t, 0, 250) * 0.5
      drawDoor(cv, look, doorX, open, 0, false)
      drawPeek(cv, doorX, open, ramp(ev.t, 150, 450), ev.t)
    } else if (ev?.kind === 'yes') {
      // Back in he goes, and the door shuts behind him.
      const open = 0.5 * (1 - ramp(ev.t, 250, 500))
      drawDoor(cv, look, doorX, open, 0, open === 0)
      if (ev.t < 400) drawPeek(cv, doorX, open, 1 - ramp(ev.t, 0, 300), ev.t)
    } else if (ev?.kind === 'slam') {
      // Shut in his face: the whole door shakes.
      drawDoor(cv, look, doorX, 0, ev.t < 450 ? (Math.floor(ev.t / 45) % 3) - 1 : 0, true)
    } else {
      // Behind the closed door, sock on the knob: it rattles in bursts, hearts drift out.
      drawDoor(cv, look, doorX, 0, t % 800 < 350 ? Math.floor(t / 90) % 2 : 0, true)
      drawFloaters(cv, look, doorX, t, 1300)
    }
    shown = Math.floor(captionAge / 22)
  } else {
    // A last rattle; he walks out, stops, turns to you and winks.
    const shake = t < 550 ? Math.floor(t / 60) % 2 : 0
    const open = ramp(t, 550, 750) - ramp(t, 1700, 1900)
    drawDoor(cv, look, doorX, open, shake, t < 550)
    if (t < 600) drawFloaters(cv, look, doorX, t, 600)
    const stopX = Math.max(2, doorX - 22)
    if (t >= 700 && t < 1500) {
      const x = Math.round(doorX + 1 - ramp(t, 700, 1500) * (doorX + 1 - stopX))
      cv.sprite(him.concat(himLegs[Math.floor(t / 120) % 2] ?? []), x, 1)
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
