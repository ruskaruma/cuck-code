// What the band says and what you hear, for each look. Each voice line is a
// clip in sounds/ (or a voice pack's file) and the caption shown while it
// plays. In captions, {she} {her} {herself} are whoever walks in, and {he}
// {his} are Claude, as the cast has them.

import type { EventKind } from '../types'
import type { Look, Skin } from './scene'

// An interruption of the busy loop: what the band says, and the sounds it
// makes at their moments (milliseconds in).
type Happening = { say: readonly string[]; cues: readonly (readonly [number, string])[] }

export type Lines = {
  enter: readonly string[] // as she walks in
  busy: readonly string[] // between voice lines while Claude works
  leave: readonly string[] // as the door opens again
  spinner: readonly string[] // Claude Code's spinner word
  done: string // the word in "Cucked for 32s"
  her: Readonly<Record<string, string>> // heard through the door, most of the time
  his: Readonly<Record<string, string>> // and now and then
  bed: { creaks: readonly [string, string]; thump?: string } // the beat under it all
  wink: readonly [clip: string, caption: string] // to you, as he winks
  knock: readonly [clip: string, caption: string] // asking you for a yes
  yes: readonly string[] // you said yes
  slam: readonly string[] // you didn't
  friends: readonly [one: string, many: string] // he brought friends; {n} is how many
  happenings: Readonly<Record<Exclude<EventKind, 'knock' | 'yes' | 'slam'>, Happening>>
  refused: readonly string[] // Claude said no
  morning: readonly string[] // she lets herself out
}

// The languages her lines come in; mix is English and Spanish.
export type Lang = 'mix' | 'en' | 'es' | 'pt' | 'fr' | 'it' | 'hi'
export const LANGS: readonly Lang[] = ['mix', 'en', 'es', 'pt', 'fr', 'it', 'hi']

// A voice pack: lines of its own, each a sound file's path and its caption.
export type Pack = {
  name: string
  her: Readonly<Record<string, string>>
  his: Readonly<Record<string, string>>
  wink?: readonly [string, string]
}

const HER: Readonly<Record<Exclude<Lang, 'mix'>, Readonly<Record<string, string>>>> = {
  en: {
    her_yeah: 'oh yeahh! oh yeahhh!',
    her_dontstop: "don't stop! don't stop!",
    her_rightthere: 'right there... right there!',
    her_omg: 'oh my god... claude!',
    her_yes: 'yes! yes! yes!',
  },
  es: { her_papi: 'sí papi! sí papi!', her_aypapi: 'ay, papi!', her_mas: 'más, papi, más!', her_es_nopares: '¡no pares!' },
  pt: { her_pt_isso: 'isso, papai!', her_pt_naopara: 'não para! não para!', her_pt_ai: 'ai, que delícia!', her_pt_vai: 'vai, claude!' },
  fr: { her_fr_oui: 'oh oui! oui!', her_fr_encore: 'encore! encore!', her_fr_ohlala: 'oh là là!', her_fr_arrete: "ne t'arrête pas!" },
  it: { her_it_si: 'sì! sì! sì!', her_it_ancora: 'ancora! ancora!', her_it_mamma: 'mamma mia!', her_it_fermarti: 'non fermarti!' },
  hi: { her_hi_haan: 'haan, haan, haan!', her_hi_rukomat: 'ruko mat!', her_hi_wahin: 'wahin... wahin pe!', her_hi_uff: 'uff, claude!' },
}

const STANDARD: Lines = {
  enter: [
    "{she}'s here for claude.",
    'take the back seat.',
    "don't wait up.",
    '{she} asked for claude, not you.',
    'hold the door. claude does the rest.',
  ],
  busy: [
    '*creak* *creak* *creak*',
    '*thump* *thump*',
    'is that the headboard?',
    "they're being so loud.",
    'claude is really going for it.',
    "there's a sock on the door.",
  ],
  leave: ["claude's done.", 'the door opens...', 'finally.'],
  spinner: [
    'Taking the back seat',
    'Waiting outside',
    'Letting it do the work',
    'Pretending not to listen',
    'Holding the door',
    'Not asking questions',
  ],
  done: 'Cucked',
  her: { ...HER.en, her_papi: 'sí papi! sí papi!', her_aypapi: 'ay, papi!', her_mas: 'más, papi, más!' },
  his: {
    him_gooddee_door: 'claude: who got the good d?',
    him_daddy: "claude: who's your daddy?",
  },
  bed: { creaks: ['creak1', 'creak2'], thump: 'thump' },
  wink: ['him_gooddee', 'claude: who got the good d?'],
  knock: ['him_yespapi', 'claude: can I get a yes, papi?'],
  yes: ['you said yes. of course you did.', 'consent: given.', '{he} goes back in.'],
  slam: ['*slam*', 'the door slams in your face.'],
  friends: ['{he} brought a friend.', '{he} brought friends. ({n})'],
  happenings: {
    friend: { say: [], cues: [[1000, 'her_ohhi']] },
    breather: { say: ["they're taking a breather.", 'towels. water. round two.'], cues: [[1500, 'her_water']] },
    pizza: { say: ["{she} ordered pizza. they're not stopping.", "pizza's here. they're not done."], cues: [[1150, 'knock'], [1500, 'pizza']] },
    neighbour: { say: ['the neighbour is banging on the wall.'], cues: [[0, 'wallbang'], [1000, 'neighbour']] },
    landlord: { say: ['the landlord is here. this is a family building.'], cues: [[1300, 'landlord']] },
    flop: { say: ['performance issues...', 'the tests failed. it happens to every model.'], cues: [[150, 'trombone']] },
    passed: { say: ["the tests passed. {she}'s screaming.", "all green. {she}'s loud about it."], cues: [[100, 'her_scream']] },
  },
  refused: ['claude said no. respect.', 'not tonight. claude said no.'],
  morning: ['the morning after. {she} lets {herself} out.', "dawn. {she}'s carrying {her} shoes."],
}

// For screen-shares and the office: the same scene, nothing to explain.
const SAFE: Lines = {
  enter: ["{she}'s here to pair with claude.", 'code review in progress.', 'do not disturb: deep work.'],
  busy: [
    '*clack* *clack* *clack*',
    'the tests are running.',
    'claude is really going for it.',
    "there's a do-not-disturb sign on the door.",
  ],
  leave: ["claude's done.", 'the door opens...', 'shipped.'],
  spinner: ['Pair programming', 'Reviewing the diff', 'Watching the build', 'Taking notes', 'Holding the door'],
  done: 'Shipped',
  her: {
    sfw_shipit: 'ship it!',
    sfw_lgtm: 'looks good to me!',
    sfw_green: 'all green!',
    sfw_merge: 'merge it!',
  },
  his: { him_senior: "claude: trust me, I'm a senior engineer." },
  bed: { creaks: ['clack1', 'clack2'] },
  wink: ['him_goodcode', 'claude: who got the good code?'],
  knock: ['him_canigetayes', 'claude: can I get a yes?'],
  yes: ['approved.', 'you said yes.'],
  slam: ['*slam*', 'request denied.'],
  friends: ['another reviewer joins.', 'more reviewers join. ({n})'],
  happenings: {
    friend: { say: [], cues: [] },
    breather: { say: ['coffee break.'], cues: [[1500, 'sfw_coffee']] },
    pizza: { say: ["lunch is here. they're not stopping."], cues: [[1150, 'knock'], [1500, 'pizza']] },
    neighbour: { say: ['the neighbours would like it quieter.'], cues: [[0, 'wallbang'], [1000, 'neighbour']] },
    landlord: { say: ['facilities came by.'], cues: [] },
    flop: { say: ['the tests failed.'], cues: [[150, 'trombone']] },
    passed: { say: ['all green!'], cues: [[100, 'sfw_green']] },
  },
  refused: ['claude declined.'],
  morning: ['the next morning.'],
}

// The season's costumes and lines.
const SEASONS: Readonly<Record<Exclude<Skin, 'none'>, Partial<Lines>>> = {
  // She came as a witch, he came as a vampire.
  halloween: {
    enter: ["{she}'s here for claude. trick or treat.", '{she} came as a witch. {he} came as a vampire.', "don't wait up. it's halloween."],
    busy: [...STANDARD.busy, '*creak* *creak* *howl*', 'is that a bat?'],
    spinner: [...STANDARD.spinner, 'Trick or treating', 'Hiding from the vampire'],
    her: { ...STANDARD.her, her_count: 'oh, count claude!' },
    his: { him_goodbite_door: 'claude: who got the good bite?', him_daddy: "claude: who's your daddy?" },
    wink: ['him_goodbite', 'claude: who got the good bite?'],
  },
  // She came as an elf, he came as Santa.
  christmas: {
    enter: ["{she}'s here for claude. it's christmas.", '{she} came as an elf. {he} came as santa.', "don't wait up for santa."],
    busy: [...STANDARD.busy, '*jingle* *jingle* *jingle*', 'are those sleigh bells?'],
    spinner: [...STANDARD.spinner, 'Waiting up for Santa', 'Hanging the stockings'],
    her: { ...STANDARD.her, her_santa: 'oh, santa claude!' },
    his: { him_hohoho: 'claude: ho ho ho!', him_daddy: "claude: who's your daddy?" },
    wink: ['him_goodgift', 'claude: who got the good gift?'],
  },
  // Roses on the door.
  valentine: {
    enter: ["{she}'s here for claude. it's valentine's.", 'roses on the door. claude does the rest.', "don't wait up. it's valentine's."],
    busy: [...STANDARD.busy, 'love is in the air.', '*creak* *creak* *kiss*'],
    spinner: [...STANDARD.spinner, 'Buying the roses', 'Third-wheeling'],
    her: { ...STANDARD.her, her_valentine: "happy valentine's, claude!" },
    wink: ['him_valentine', 'claude: be my valentine?'],
  },
}

// Every model is a different man: Haiku is done fast, Sonnet's a regular.
// The rest say the usual.
const MODEL_WINKS: Readonly<Record<string, readonly [string, string]>> = {
  haiku: ['him_haiku', 'claude: ...was it good for you?'],
  sonnet: ['him_sonnet', 'claude: same time next prompt?'],
}

// When a guy walks in: his lines, and his own sounds for what happens.
const GUY: Readonly<Record<string, string>> = {
  guy_yeah: 'oh yeah! oh yeah!',
  guy_dontstop: "don't stop! don't stop!",
  guy_omg: 'oh my god... claude!',
  guy_yes: 'yes! yes! yes!',
}
const GUY_SAFE: Readonly<Record<string, string>> = { guy_shipit: 'ship it!', guy_lgtm: 'looks good to me!', guy_green: 'all green!', guy_merge: 'merge it!' }
const GUY_SOUNDS: Readonly<Record<string, string>> = {
  her_ohhi: 'guy_ohhi', her_water: 'guy_water', her_scream: 'guy_scream', sfw_coffee: 'guy_coffee', sfw_green: 'guy_green',
}

// When Claude is a woman: her lines.
const CLAUDIA: Readonly<Record<string, string>> = {
  her_goodit_door: 'claude: who got it good?',
  her_mommy_door: "claude: who's your mommy?",
}

const PRONOUNS = {
  woman: { she: 'she', her: 'her', herself: 'herself', he: 'she', his: 'her' },
  man: { she: 'he', her: 'his', herself: 'himself', he: 'he', his: 'his' },
}

// The lines for a look; `model` is the model that did the turn, for the
// wink; `lang` the language of her lines; `pack` a voice pack's lines.
export function linesFor(look: Look, model = '', lang: Lang = 'mix', pack: Pack | null = null): Lines {
  let l: Lines = look.sfw ? SAFE : look.skin === 'none' ? STANDARD : { ...STANDARD, ...SEASONS[look.skin] }
  const guestMan = look.cast === 'he-she' || look.cast === 'he-he'
  const claudeWoman = look.cast === 'he-she' || look.cast === 'she-she'

  if (!look.sfw && look.skin === 'none') {
    const wink = MODEL_WINKS[/haiku|sonnet/i.exec(model)?.[0].toLowerCase() ?? '']
    if (wink) l = { ...l, wink }
  }
  if (!look.sfw && lang !== 'mix') {
    // Her lines in that language, and the season's own among them.
    const seasonal = Object.fromEntries(Object.entries(l.her).filter(([clip]) => !(clip in STANDARD.her)))
    l = { ...l, her: { ...HER[lang], ...seasonal } }
  }
  if (guestMan) {
    const happenings = eachHappening(l.happenings, h => ({ say: h.say, cues: h.cues.map(([at, clip]) => [at, GUY_SOUNDS[clip] ?? clip] as const) }))
    l = { ...l, her: look.sfw ? GUY_SAFE : GUY, happenings }
  }
  if (claudeWoman) {
    l = look.sfw
      ? { ...l, his: { her_senior_door: "claude: trust me, I'm a senior engineer." }, wink: ['her_goodcode', 'claude: who got the good code?'], knock: ['her_canigetayes', 'claude: can I get a yes?'] }
      : { ...l, his: CLAUDIA, wink: ['her_goodit', 'claude: who got it good?'], knock: ['her_yespapi', 'claude: can I get a yes, papi?'] }
  }
  if (pack && !look.sfw) {
    l = {
      ...l,
      her: Object.keys(pack.her).length > 0 ? pack.her : l.her,
      his: Object.keys(pack.his).length > 0 ? pack.his : l.his,
      wink: pack.wink ?? l.wink,
    }
  }
  return voiced(l, guestMan ? PRONOUNS.man : PRONOUNS.woman, claudeWoman ? PRONOUNS.woman : PRONOUNS.man)
}

// Fills in who's who in every caption.
function voiced(l: Lines, guest: Record<string, string>, claude: Record<string, string>): Lines {
  const say = (text: string) =>
    text.replace(/\{(she|her|herself|he|his)\}/g, (_, w: string) => ((w === 'he' || w === 'his' ? claude : guest)[w] ?? w))
  const all = (xs: readonly string[]) => xs.map(say)
  return {
    ...l,
    enter: all(l.enter),
    busy: all(l.busy),
    leave: all(l.leave),
    yes: all(l.yes),
    slam: all(l.slam),
    friends: [say(l.friends[0]), say(l.friends[1])],
    refused: all(l.refused),
    morning: all(l.morning),
    happenings: eachHappening(l.happenings, h => ({ say: all(h.say), cues: h.cues })),
  }
}

function eachHappening(h: Lines['happenings'], fn: (h: Happening) => Happening): Lines['happenings'] {
  return {
    friend: fn(h.friend),
    breather: fn(h.breather),
    pizza: fn(h.pizza),
    neighbour: fn(h.neighbour),
    landlord: fn(h.landlord),
    flop: fn(h.flop),
    passed: fn(h.passed),
  }
}

// The season's costumes, by the date: Halloween from Oct 24 to Nov 1,
// Christmas from Dec 18 to 26, Valentine's from Feb 10 to 15.
export function season(d: Date): Skin {
  const month = d.getMonth() + 1
  const day = d.getDate()
  if ((month === 10 && day >= 24) || (month === 11 && day === 1)) return 'halloween'
  if (month === 12 && day >= 18 && day <= 26) return 'christmas'
  if (month === 2 && day >= 10 && day <= 15) return 'valentine'
  return 'none'
}

// The commands that run a test suite.
export const TEST_COMMAND =
  /(^|[\s;&|(])((npm|pnpm|yarn|bun)( run)? test|npx (jest|vitest|playwright test)|jest|vitest|pytest|go test|cargo (test|nextest)|make (test|check)|mvn( -\S+)* (test|verify)|gradlew? test|dotnet test|rspec|phpunit|mix test|deno test|ctest|tox|nox)\b/

// A voice pack's pack.json, checked: each clip a plain .wav file name in the
// pack's folder, each caption short and on one line.
export function readPack(dir: string, name: string, json: string): Pack | null {
  let o: unknown
  try {
    o = JSON.parse(json)
  } catch {
    return null
  }
  if (typeof o !== 'object' || o === null) return null
  const raw = o as Record<string, unknown>
  const lines = (v: unknown): Record<string, string> => {
    const out: Record<string, string> = {}
    if (typeof v !== 'object' || v === null) return out
    for (const [file, caption] of Object.entries(v)) {
      if (/^[\w.-]+\.wav$/i.test(file) && !file.startsWith('.') && typeof caption === 'string') {
        out[`${dir}/${file}`] = caption.replace(/[\u0000-\u001f\u007f]/g, ' ').slice(0, 60)
      }
    }
    return out
  }
  const wink = Object.entries(lines(raw.wink))[0]
  return { name: typeof raw.name === 'string' ? raw.name.slice(0, 40) : name, her: lines(raw.her), his: lines(raw.his), ...(wink ? { wink } : {}) }
}
