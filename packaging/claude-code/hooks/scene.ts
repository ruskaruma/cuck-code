// The strip drawn above the prompt: a door, her, and him (Claude). Pure
// drawing code. A frame is a grid of pixels, two to a terminal cell (upper
// and lower half blocks), plus one row of caption text underneath.

import type { Kind } from '../types'

export const ROWS = 9 // 8 rows of pixels (16 high) and the caption row
export const MAX_COLUMNS = 64
export const MIN_COLUMNS = 40
// busy loops until Claude finishes. In leave, he winks at WINK_AT.
export const DURATION: Record<Kind, number> = { enter: 1800, busy: Infinity, leave: 5000 }
export const WINK_AT = 1900

const PX_H = 16
const DEFAULT = 0x01000000 // the terminal's own colour
const CAPTION_X = 2

export const ENTER_LINES = [
  "she's here for claude.",
  'take the back seat.',
  "don't wait up.",
  'she asked for claude, not you.',
  'hold the door. claude does the rest.',
]

export const LEAVE_LINES = [
  "claude's done.",
  'the door opens...',
  'finally.',
]

// What he says to you as he winks.
export const WINK_LINE = 'claude: who got the good d?'

// What the band says while Claude works, between her lines.
export const BUSY_LINES = [
  '*creak* *creak* *creak*',
  '*thump* *thump*',
  'is that the headboard?',
  "they're being so loud.",
  'claude is really going for it.',
  "there's a sock on the door.",
]

// The lines heard through the door (the clip's name, then what the band
// says): hers most of the time, now and then his.
export const HER_LINES: Record<string, string> = {
  her_yeah: 'oh yeahh! oh yeahhh!',
  her_dontstop: "don't stop! don't stop!",
  her_rightthere: 'right there... right there!',
  her_omg: 'oh my god... claude!',
  her_yes: 'yes! yes! yes!',
  her_papi: 'sí papi! sí papi!',
  her_aypapi: 'ay, papi!',
  her_mas: 'más, papi, más!',
}
export const HIS_LINES: Record<string, string> = {
  him_gooddee_door: 'claude: who got the good d?',
  him_daddy: "claude: who's your daddy?",
}

// What Claude Code's spinner says while it works ("Sauteing...").
export const SPINNER_WORDS = [
  'Taking the back seat',
  'Waiting outside',
  'Letting it do the work',
  'Pretending not to listen',
  'Holding the door',
  'Not asking questions',
]

const palette: Record<string, number> = {
  // her
  h: 0x2b1a14, s: 0xf1c3a0, r: 0xe0444d, k: 0x7d1820,
  // him: hair, skin, white tank top, its name tag, jeans, shoes, eyes and grin
  g: 0x4a2c1a, t: 0xd9a27c, w: 0xf2efe9, T: 0xe0444d, j: 0x3a5a8c, n: 0x2a2a2a, e: 0x1b1210, m: 0x7d1820,
  // the shine in his eyes, bright and dimmer as it twinkles
  W: 0xffffff, V: 0xbfe6ff,
  // the door: frame, panel, knob, the dark room and its glow
  D: 0x5a3a22, P: 0x8c5e34, o: 0xffd76a, I: 0x1a0d10, G: 0x6b1f37,
  // a heart, a sparkle, the sock on the knob
  H: 0xff5a6e, '*': 0xffd76a, x: 0xf2efe9, X: 0xe0444d,
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
const sock = ['xx.', 'XX.', 'xx.', 'xxx']

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))
const ramp = (t: number, from: number, to: number) => clamp((t - from) / (to - from), 0, 1)

class Canvas {
  w: number
  px: Int32Array
  constructor(w: number) {
    this.w = w
    this.px = new Int32Array(w * PX_H).fill(-1)
  }
  set(x: number, y: number, c: number) {
    if (x >= 0 && x < this.w && y >= 0 && y < PX_H) this.px[y * this.w + x] = c
  }
  sprite(rows: readonly string[], x0: number, y0: number, fade = 1, clipRight = Infinity) {
    rows.forEach((row, y) => {
      for (let i = 0; i < row.length; i++) {
        const c = palette[row[i] ?? '.']
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

// open: 0 closed, 1 wide open. shake: the door rattles in its frame.
function drawDoor(cv: Canvas, x0: number, open: number, shake: number, withSock: boolean) {
  const W = 10
  const panel = Math.round((1 - open) * 7) + 1 // columns of panel still across the doorway
  for (let y = 0; y < PX_H; y++) {
    for (let x = 0; x < W; x++) {
      let ch: string
      if (y === 0 || x === 0 || x === W - 1) ch = 'D'
      else if (x <= panel) {
        const inset = y === 2 || y === 6 || y === 9 || y === 13
        ch = (x === 2 || x === panel - 1) && y > 1 && y < 14 ? 'D' : inset && x > 1 && x < panel ? 'D' : 'P'
        if (y === 8 && x === panel - 1 && panel > 3) ch = 'o'
      } else ch = y < 4 ? 'G' : 'I'
      cv.set(x0 + x + (x > 0 && x <= panel ? shake : 0), y, palette[ch] ?? 0)
    }
  }
  if (withSock && open === 0) cv.sprite(sock, x0 + 6 + shake, 9)
}

function drawHearts(cv: Canvas, doorX: number, t: number, period: number) {
  for (const [phase, dx] of [[0, -7], [period / 2, -13]] as const) {
    const p = ((t + phase) % period) / (period * 0.7)
    if (p < 1) cv.sprite(heart, doorX + dx, Math.round(9 - p * 8), 1 - p * 0.7)
  }
}

// One frame of a clip t milliseconds in, `columns` wide, as Raster cells:
// row-major [codePoint, foreground, background] triples. captionAge is how
// long the caption has been up (it types itself out).
export function frame(kind: Kind, t: number, columns: number, caption: string, captionAge = t): Uint32Array {
  const cv = new Canvas(columns)
  const doorX = columns - 13
  let fade = 1
  let shown = 0 // caption characters shown

  if (kind === 'enter') {
    // She walks in from the left and through the door; it shuts, a sock goes on the knob.
    const open = ramp(t, 650, 850) - ramp(t, 1150, 1350)
    drawDoor(cv, doorX, open, 0, t > 1350)
    const x = Math.round(-9 + ramp(t, 0, 1100) * (doorX + 12))
    if (t < 1150) cv.sprite(her.concat(herLegs[Math.floor(t / 120) % 2] ?? []), x, 2, 1, open > 0.5 ? Infinity : doorX + 1)
    if (t > 1300) {
      const rise = Math.floor(ramp(t, 1300, 1800) * 6)
      cv.sprite(heart, doorX - 7, 7 - rise, 1 - ramp(t, 1550, 1800))
    }
    fade = 1 - ramp(t, 1650, 1800)
    shown = Math.floor((captionAge - 150) / 22)
  } else if (kind === 'busy') {
    // Behind the closed door, sock on the knob: it rattles in bursts, hearts drift out.
    drawDoor(cv, doorX, 0, t % 800 < 350 ? Math.floor(t / 90) % 2 : 0, true)
    drawHearts(cv, doorX, t, 1300)
    shown = Math.floor(captionAge / 22)
  } else {
    // A last rattle; he walks out, stops, turns to you and winks.
    const shake = t < 550 ? Math.floor(t / 60) % 2 : 0
    const open = ramp(t, 550, 750) - ramp(t, 1700, 1900)
    drawDoor(cv, doorX, open, shake, t < 550)
    if (t < 600) drawHearts(cv, doorX, t, 600)
    const stopX = Math.max(2, doorX - 22)
    if (t >= 700 && t < 1500) {
      const x = Math.round(doorX + 1 - ramp(t, 700, 1500) * (doorX + 1 - stopX))
      cv.sprite(him.concat(himLegs[Math.floor(t / 120) % 2] ?? []), x, 1)
    } else if (t >= 1500) {
      const winking = t > WINK_AT
      const rows = himFront.slice()
      ;[rows[3], rows[4], rows[5], rows[6]] = winking ? faces.wink : faces.open
      // The shine twinkles.
      const twinkle = Math.floor(t / 150) % 3 === 2
      cv.sprite(twinkle ? rows.map(r => r.replace(/W/g, 'V')) : rows, stopX - 1, 1)
      if (Math.floor(t / 180) % 2 === 0) cv.sprite(sparkle, winking ? stopX + 9 : stopX - 4, winking ? 1 : 2)
    }
    fade = 1 - ramp(t, 4500, 5000)
    if (t > WINK_AT) {
      // Typed out as slowly as he says it.
      caption = WINK_LINE
      shown = 8 + Math.floor((t - WINK_AT) / 95)
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
