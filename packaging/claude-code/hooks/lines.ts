// What the band says and what you hear, for each look. Each voice line is a
// clip in sounds/ and the caption shown while it plays.

import type { Look, Skin } from './scene'

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
}

const STANDARD: Lines = {
  enter: [
    "she's here for claude.",
    'take the back seat.',
    "don't wait up.",
    'she asked for claude, not you.',
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
  her: {
    her_yeah: 'oh yeahh! oh yeahhh!',
    her_dontstop: "don't stop! don't stop!",
    her_rightthere: 'right there... right there!',
    her_omg: 'oh my god... claude!',
    her_yes: 'yes! yes! yes!',
    her_papi: 'sí papi! sí papi!',
    her_aypapi: 'ay, papi!',
    her_mas: 'más, papi, más!',
  },
  his: {
    him_gooddee_door: 'claude: who got the good d?',
    him_daddy: "claude: who's your daddy?",
  },
  bed: { creaks: ['creak1', 'creak2'], thump: 'thump' },
  wink: ['him_gooddee', 'claude: who got the good d?'],
  knock: ['him_yespapi', 'claude: can I get a yes, papi?'],
  yes: ['you said yes. of course you did.', 'consent: given.', 'he goes back in.'],
  slam: ['*slam*', 'the door slams in your face.'],
}

// For screen-shares and the office: the same scene, nothing to explain.
const SAFE: Lines = {
  enter: ["she's here to pair with claude.", 'code review in progress.', 'do not disturb: deep work.'],
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
}

// She came as a witch, he came as a vampire.
const HALLOWEEN: Partial<Lines> = {
  enter: ["she's here for claude. trick or treat.", 'she came as a witch. he came as a vampire.', "don't wait up. it's halloween."],
  busy: [...STANDARD.busy, '*creak* *creak* *howl*', 'is that a bat?'],
  spinner: [...STANDARD.spinner, 'Trick or treating', 'Hiding from the vampire'],
  her: { ...STANDARD.her, her_count: 'oh, count claude!' },
  his: { him_goodbite_door: 'claude: who got the good bite?', him_daddy: "claude: who's your daddy?" },
  wink: ['him_goodbite', 'claude: who got the good bite?'],
}

export function linesFor(look: Look): Lines {
  if (look.sfw) return SAFE
  if (look.skin === 'halloween') return { ...STANDARD, ...HALLOWEEN }
  return STANDARD
}

// The season's costumes, by the date: Halloween from Oct 24 to Nov 1.
export function season(d: Date): Skin {
  const month = d.getMonth() + 1
  const day = d.getDate()
  if ((month === 10 && day >= 24) || (month === 11 && day === 1)) return 'halloween'
  return 'none'
}
