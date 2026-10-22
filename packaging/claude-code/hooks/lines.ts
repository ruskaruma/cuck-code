// What the band says and what you hear, for each look. Each voice line is a
// clip in sounds/ and the caption shown while it plays.

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
  friends: ['he brought a friend.', 'he brought friends. ({n})'],
  happenings: {
    friend: { say: [], cues: [[1000, 'her_ohhi']] },
    breather: { say: ["they're taking a breather.", 'towels. water. round two.'], cues: [[1500, 'her_water']] },
    pizza: { say: ["she ordered pizza. they're not stopping.", "pizza's here. they're not done."], cues: [[1150, 'knock'], [1500, 'pizza']] },
    neighbour: { say: ['the neighbour is banging on the wall.'], cues: [[0, 'wallbang'], [1000, 'neighbour']] },
    landlord: { say: ['the landlord is here. this is a family building.'], cues: [[1300, 'landlord']] },
    flop: { say: ['performance issues...', 'the tests failed. it happens to every model.'], cues: [[150, 'trombone']] },
    passed: { say: ["the tests passed. she's screaming.", "all green. she's loud about it."], cues: [[100, 'her_scream']] },
  },
  refused: ['claude said no. respect.', 'not tonight. claude said no.'],
  morning: ['the morning after. she lets herself out.', "dawn. she's carrying her shoes."],
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

// She came as a witch, he came as a vampire.
const HALLOWEEN: Partial<Lines> = {
  enter: ["she's here for claude. trick or treat.", 'she came as a witch. he came as a vampire.', "don't wait up. it's halloween."],
  busy: [...STANDARD.busy, '*creak* *creak* *howl*', 'is that a bat?'],
  spinner: [...STANDARD.spinner, 'Trick or treating', 'Hiding from the vampire'],
  her: { ...STANDARD.her, her_count: 'oh, count claude!' },
  his: { him_goodbite_door: 'claude: who got the good bite?', him_daddy: "claude: who's your daddy?" },
  wink: ['him_goodbite', 'claude: who got the good bite?'],
}

// Every model is a different man: Haiku is done fast, Sonnet's a regular.
// The rest say the usual.
const MODEL_WINKS: Readonly<Record<string, readonly [string, string]>> = {
  haiku: ['him_haiku', 'claude: ...was it good for you?'],
  sonnet: ['him_sonnet', 'claude: same time next prompt?'],
}

// The lines for a look; `model` is the model that did the turn, for the wink.
export function linesFor(look: Look, model = ''): Lines {
  if (look.sfw) return SAFE
  if (look.skin === 'halloween') return { ...STANDARD, ...HALLOWEEN }
  const wink = MODEL_WINKS[/haiku|sonnet/i.exec(model)?.[0].toLowerCase() ?? '']
  return wink ? { ...STANDARD, wink } : STANDARD
}

// The season's costumes, by the date: Halloween from Oct 24 to Nov 1.
export function season(d: Date): Skin {
  const month = d.getMonth() + 1
  const day = d.getDate()
  if ((month === 10 && day >= 24) || (month === 11 && day === 1)) return 'halloween'
  return 'none'
}

// The commands that run a test suite.
export const TEST_COMMAND =
  /(^|[\s;&|(])((npm|pnpm|yarn|bun)( run)? test|npx (jest|vitest|playwright test)|jest|vitest|pytest|go test|cargo (test|nextest)|make (test|check)|mvn( -\S+)* (test|verify)|gradlew? test|dotnet test|rspec|phpunit|mix test|deno test|ctest|tox|nox)\b/
