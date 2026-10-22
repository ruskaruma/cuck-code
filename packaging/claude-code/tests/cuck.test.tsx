import { expect, mock, test } from 'claude-code/testing'
import type { Engine } from 'claude-code/testing'
import type { On } from 'claude-code'

const BAND = {
  plugin: 'cuck-code',
  component: 'AbovePrompt',
  requestId: 'band',
  props: { hasSurvey: false, isWorking: true, maxRows: 20, bodyColumns: 100, scroll: { offset: 0, bodyRows: 19 }, view: {} },
} as const

const SESSION = { cwd: '/', surface: 'terminal', isInteractive: true } as const
const PROMPT = { text: 'fix the login bug', wait: false, origin: { kind: 'composer' } } as const
const TURN = { answer: 'done', durationMs: 32000, turnId: 't1' }
const DONE = { ...TURN, isAborted: false, reason: 'answer' } as const
const command = (name: string, args = '') =>
  ({ command: name, args, origin: { kind: 'composer' }, presentation: { isFullscreen: false, columns: 100 } }) as const

// The engine beneath: it takes prompts and turns, runs commands and tools,
// and draws its own band. It records every sound the mod plays, every line
// it puts under a dialog, and answers the check for a call in progress with
// `onCall`.
function engine(on: On, { onCall = false, os = 'Linux', env = {} } = {}) {
  mock.env(on, env)
  const heard = { sounds: [] as string[], players: [] as string[], notices: [] as string[], toasts: [] as string[], copied: [] as string[] }
  on('prompt.submit', (_, e) => ({ text: e.text }))
  on('turn.complete', () => ({ text: 'done' }))
  on('command.register', () => ({ value: { command: 'cuck' } }))
  on('session.start', (_, e) => ({ cwd: e.cwd }))
  on('process.run', (_, e) => {
    const wav = /([\w-]+)\.wav/.exec(e.argv.at(-1) ?? '')
    if (wav?.[1]) {
      heard.sounds.push(wav[1])
      heard.players.push(e.argv[0] ?? '')
    }
    const exitCode = e.argv[0] === 'uname' || wav || onCall ? 0 : 1
    const stdout = e.argv[0] === 'uname' ? `${os}\n` : ''
    return { value: { exitCode, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('session.id', () => ({ value: 'this-session' }))
  on('ui.toast', (_, e) => {
    heard.toasts.push(e.text)
    return { value: undefined }
  })
  on('ui.copy', (_, e) => {
    heard.copied.push(e.text)
    return { value: { isCopied: true } }
  })
  on('attribution.text', (_, e) => ({ text: e.text }))
  on('agent.spawn', () => ({ model: 'claude-haiku-5-5', agentId: 'helper' }))
  on('session.compact', () => ({ skip: 'nothing to compact' }))
  on('ui.notice', (_, e) => {
    heard.notices.push(e.text ?? '')
    return { value: undefined }
  })
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)
    return <Text>engine</Text>
  })
  return heard
}

// What the band says now, as a surface that draws text alone shows it.
const caption = async ($: Engine) => {
  const band = await $.ui.mount({ ...BAND, surface: 'desktop' })
  const text = (await band.find({ type: 'Text' }))?.text
  await band.unmount()
  return text
}

test('she walks in, they carry on while Claude works, he walks out and winks, the band clears', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)

  await $.prompt.submit(PROMPT)
  const first = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await first.find({ key: 'strip' })).toBeDefined()
  await first.unmount()

  // Well past the walk-in: still going while the turn runs, with sound.
  await clock.advance(15000)
  const busy = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await busy.find({ key: 'strip' })).toBeDefined()
  await busy.unmount()
  expect(heard.sounds.some(s => s.startsWith('creak'))).toBe(true)
  expect(heard.sounds.some(s => s.startsWith('her_') || s.startsWith('him_'))).toBe(true)

  await $.turn.complete(DONE)
  expect(heard.sounds.at(-1)).toBe('ding')

  // He turns to you, winks, and says it.
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_gooddee')
  expect(await caption($)).toBe('claude: who got the good d?')

  await clock.advance(3200)
  const after = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await after.find({ key: 'strip' })).toBeUndefined()
  expect((await after.find({ type: 'Text', text: /engine/ }))?.text).toBe('engine')
  await after.unmount()
})

test('narrow terminals and other surfaces get the caption alone', async ($, on) => {
  mock.clock(on, { now: 1000 })
  mock.store(on)
  engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  const narrow = await $.ui.mount({ ...BAND, surface: 'terminal', props: { ...BAND.props, bodyColumns: 30 } })
  expect(await narrow.find({ type: 'Text', text: /claude|seat|wait|door/ })).toBeDefined()
  await narrow.unmount()
  expect(await caption($)).toMatch(/claude|seat|wait|door/)
})

test('an interrupt clears the band; subagents and empty prompts play nothing', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  engine(on)
  await $.session.start(SESSION)

  await $.prompt.submit({ ...PROMPT, text: '   ' })
  await $.turn.complete({ ...DONE, agentId: 'sub' })
  const idle = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await idle.find({ key: 'strip' })).toBeUndefined()
  await idle.unmount()

  await $.prompt.submit(PROMPT)
  await clock.advance(4000)
  await $.turn.complete({ ...TURN, isAborted: true, reason: 'aborted' })
  const stopped = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await stopped.find({ key: 'strip' })).toBeUndefined()
  await stopped.unmount()
})

test('/cuck quiet and /cuck-quiet mute everything, and unmute', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)

  expect((await $.command.run(command('cuck', 'quiet'))).text).toMatch(/muted/)
  await $.prompt.submit(PROMPT)
  await clock.advance(15000)
  await $.turn.complete(DONE)
  await clock.advance(5000)
  expect(heard.sounds).toEqual([])

  expect((await $.command.run(command('cuck-quiet'))).text).toMatch(/back on/)
  expect((await $.command.run(command('cuck'))).text).toMatch(/\/cuck sfw/)
})

test('when Claude needs a yes he knocks, asks under the dialog, and goes back in when you give it', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  let decision: 'allow' | 'ask' = 'ask'
  on('tool.check', () => ({ decision }))
  on('tool.call', () => ({ result: { stdout: '', stderr: '', interrupted: false } }))
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(5000)

  await $.tool.check({ tool: 'Bash', input: { command: 'rm -rf build' }, tool_use_id: 'tu1', approver: 'person' })
  await clock.advance(1500)
  expect(heard.sounds).toContain('knock')
  expect(heard.sounds.at(-1)).toBe('him_yespapi')
  expect(heard.notices).toContain('claude: can I get a yes, papi?')
  expect(await caption($)).toBe('claude: can I get a yes, papi?')

  decision = 'allow' // you said yes
  await $.tool.call({ tool: 'Bash', command: 'rm -rf build', tool_use_id: 'tu1' })
  await clock.advance(100)
  expect(await caption($)).toMatch(/yes|consent|back in/)
})

test('turn him down and the door slams', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  on('tool.check', () => ({ decision: 'ask' }))
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(5000)

  await $.tool.check({ tool: 'Edit', input: { file_path: '/a' }, tool_use_id: 'tu2', approver: 'person' })
  await clock.advance(1500)
  await $.turn.complete({ ...TURN, isAborted: true, reason: 'aborted' })
  await clock.advance(200)
  expect(heard.sounds.at(-1)).toBe('slam')
  await clock.advance(1500)
  expect(await caption($)).toBe('engine')
})

test('a call the rules allow, or nobody can be asked about, gets no knock', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  let decision: 'allow' | 'ask' = 'allow'
  on('tool.check', () => ({ decision }))
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.tool.check({ tool: 'Read', input: { file_path: '/a' }, tool_use_id: 'tu3', approver: 'person' })
  decision = 'ask'
  await $.tool.check({ tool: 'Bash', input: { command: 'ls' }, tool_use_id: 'tu4', approver: 'host' })
  await clock.advance(2000)
  expect(heard.sounds).not.toContain('knock')
  expect(heard.notices).toEqual([])
})

test('safe for work: clean lines, typing for a bed, and the good code', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  expect((await $.command.run(command('cuck', 'sfw'))).text).toMatch(/Safe for work/)

  await $.prompt.submit(PROMPT)
  await clock.advance(20000)
  expect(heard.sounds.some(s => s.startsWith('clack'))).toBe(true)
  expect(heard.sounds.filter(s => /^(creak|thump|her_)/.test(s))).toEqual([])
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_goodcode')
  expect(await caption($)).toBe('claude: who got the good code?')
})

test('on a call it goes quiet and safe for work by itself, unless told not to', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on, { onCall: true })
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(15000)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds).toEqual([])
  expect(await caption($)).toBe('claude: who got the good code?')

  await $.command.run(command('cuck', 'meetings off'))
  await $.prompt.submit(PROMPT)
  await clock.advance(15000)
  expect(heard.sounds.length).toBeGreaterThan(0)
})

test('Halloween: costumes by the calendar, and who got the good bite', async ($, on) => {
  const clock = mock.clock(on, { now: new Date(2026, 9, 30, 21, 0).getTime() })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_goodbite')

  await $.command.run(command('cuck', 'skin off'))
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_gooddee')
})

test('settings are kept for the next session', async ($, on) => {
  mock.clock(on, { now: 1000 })
  mock.store(on, { settings: { quiet: false, sfw: true, skin: 'halloween', meetings: false } })
  engine(on)
  await $.session.start(SESSION)
  expect((await $.command.run(command('cuck'))).text).toMatch(/sfw .*\(now on\)[\s\S]*now halloween[\s\S]*\(now off\)/)
})

test('on Windows the sounds play through PowerShell', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on, { env: { OS: 'Windows_NT' } })
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(5000)
  expect(heard.players.length).toBeGreaterThan(0)
  expect(heard.players.every(p => p === 'powershell')).toBe(true)
})

const SPAWN = {
  tool_use_id: 'agent1',
  prompt: 'look around',
  description: 'Look around',
  subagentType: 'Explore',
  provider: { plugin: 'engine', tier: 'core' },
  parentModel: 'claude-opus-5-5',
  background: false,
  fork: false,
} as const
const USAGE = { input_tokens: 10, output_tokens: 10, cache_creation_input_tokens: 0, cache_read_input_tokens: 0 }

test('he brought friends: they walk in while Claude works, and leave after him', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.agent.spawn(SPAWN)
  await $.agent.spawn({ ...SPAWN, tool_use_id: 'agent2' })
  await clock.advance(100)
  expect(await caption($)).toBe('he brought friends. (2)')
  await clock.advance(1200)
  expect(heard.sounds).toContain('her_ohhi')
  await clock.advance(2000)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_gooddee')
})

test('the tests: a failing run is performance issues, a passing one makes her scream', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  let failing = true
  on('tool.check', () => ({ decision: 'allow' }))
  on('tool.call', () =>
    failing
      ? { isError: true, result: 'Error: Exit code 1', text: 'Exit code 1\nFAIL src/login.test.ts' }
      : { result: { stdout: 'ok', stderr: '', interrupted: false } },
  )
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)

  await $.tool.call({ tool: 'Bash', command: 'npm test', tool_use_id: 'b1' })
  await clock.advance(300)
  expect(heard.sounds.at(-1)).toBe('trombone')
  expect(await caption($)).toMatch(/performance issues|tests failed/)

  await clock.advance(3000)
  failing = false
  await $.tool.call({ tool: 'Bash', command: 'cd api && go test ./...', tool_use_id: 'b2' })
  await clock.advance(200)
  expect(heard.sounds.at(-1)).toBe('her_scream')

  // Anything else run in a shell, or a test run sent to the background, is nobody's business.
  await clock.advance(3000)
  const before = heard.sounds.filter(n => n === 'trombone' || n === 'her_scream').length
  failing = true
  await $.tool.call({ tool: 'Bash', command: 'ls -la', tool_use_id: 'b3' })
  await $.tool.call({ tool: 'Bash', command: 'pytest', run_in_background: true, tool_use_id: 'b4' })
  await clock.advance(1000)
  expect(heard.sounds.filter(n => n === 'trombone' || n === 'her_scream').length).toBe(before)
})

test('compacting is a breather, during a turn or on its own', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.session.compact({ trigger: 'manual', messages: [] })
  await clock.advance(100)
  expect(await caption($)).toMatch(/breather|towels/)
  await clock.advance(1600)
  expect(heard.sounds).toContain('her_water')
  await clock.advance(4000)
  expect(await caption($)).toBe('engine')

  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.session.compact({ trigger: 'auto', messages: [] })
  await clock.advance(100)
  expect(await caption($)).toMatch(/breather|towels/)
})

test('five minutes in she orders pizza', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(5 * 60000 + 2000)
  expect(heard.sounds).toContain('pizza')
  expect(heard.sounds).not.toContain('neighbour')
})

test('when Claude refuses, the door stays shut', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete({
    ...TURN,
    isAborted: false,
    reason: 'refusal',
    refusal: { category: null, explanation: null },
  })
  await clock.advance(400)
  expect(heard.sounds.at(-1)).toBe('lock')
  expect(heard.sounds).not.toContain('ding')
  expect(await caption($)).toMatch(/said no/)
})

test('she stays the night: the next session sees her out', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on, { morning: 'last-session' })
  engine(on)
  await $.session.start(SESSION)
  await clock.advance(100)
  expect(await caption($)).toMatch(/morning|dawn/)
  await clock.advance(5000)
  expect(await caption($)).toBe('engine')
})

test('every model is a different man', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete({ ...DONE, usage: { ...USAGE, model: 'claude-haiku-5-5' } })
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_haiku')
  expect(await caption($)).toBe('claude: ...was it good for you?')
})

test('your stats: counted turn by turn, kept, and shown as a card', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  for (const ms of [32000, 95000]) {
    await $.prompt.submit(PROMPT)
    await clock.advance(3000)
    await $.turn.complete({ ...DONE, durationMs: ms })
    await clock.advance(5500)
  }
  expect(heard.toasts).toEqual(['Achievement unlocked: First Time. You watched Claude do it.'])
  expect(heard.sounds).toContain('fanfare')

  const shown = await $.command.run(command('cuck', 'stats'))
  expect(shown.text).toMatch(/cucked 2 times · 2m 07s in the chair/)
  expect(shown.text).toMatch(/longest 1m 35s/)
  expect(shown.text).toMatch(/1\/12 achievements: First Time/)
  const band = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await band.find({ key: 'strip' })).toBeDefined()
  await band.unmount()
  expect(await caption($)).toBe('/cuck badge puts it in your README.')
})

test('the badge: your time in the chair, on your clipboard', async ($, on) => {
  mock.clock(on, { now: 1000 })
  mock.store(on, { stats: { turns: 40, ms: 3 * 3600000 + 12 * 60000 } })
  const heard = engine(on)
  await $.session.start(SESSION)
  const out = await $.command.run(command('cuck', 'badge'))
  const md = '[![cucked for 3h 12m](https://img.shields.io/badge/cucked_for-3h_12m-ff5a6e)](https://github.com/ruskaruma/cuck-code)'
  expect(out.text).toMatch(/Copied/)
  expect(heard.copied).toEqual([md])
})

test('a turn that finishes at 3 in the morning is the Night Shift', async ($, on) => {
  const clock = mock.clock(on, { now: new Date(2026, 10, 12, 3, 10).getTime() })
  mock.store(on, { stats: { turns: 5, earned: { first: 1 } } })
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete(DONE)
  expect(heard.toasts).toEqual(['Achievement unlocked: Night Shift. A turn finished between 2 and 5 in the morning.'])
})

test('with the trailer on, Claude signs its commits and pull requests', async ($, on) => {
  mock.clock(on, { now: 1000 })
  mock.store(on)
  engine(on)
  await $.session.start(SESSION)
  const coAuthor = 'Co-Authored-By: Claude <noreply@anthropic.com>'
  expect((await $.attribution.text({ kind: 'commit', text: coAuthor })).text).toBe(coAuthor)

  expect((await $.command.run(command('cuck', 'trailer on'))).text).toMatch(/Cucked-By: Claude/)
  expect((await $.attribution.text({ kind: 'commit', text: coAuthor })).text).toBe(`${coAuthor}\nCucked-By: Claude`)
  expect((await $.attribution.text({ kind: 'commit', text: '' })).text).toBe('Cucked-By: Claude')
  expect((await $.attribution.text({ kind: 'pr', text: 'Generated with Claude Code' })).text).toMatch(/Generated with Claude Code\n\nCucked with \[Cuck Code\]/)
  expect((await $.attribution.text({ kind: 'remedy', text: 'x' })).text).toBe('x')
})

test('the cast: a guy walks in, Claude is a woman, and every line follows', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  expect((await $.command.run(command('cuck', 'cast he-she'))).text).toMatch(/he walks in, and Claude is her/)
  await $.prompt.submit(PROMPT)
  expect(await caption($)).not.toMatch(/\bshe\b/)
  await clock.advance(40000)
  expect(heard.sounds.filter(n => n.startsWith('her_') && !/^her_(goodit|mommy)/.test(n))).toEqual([])
  expect(heard.sounds.some(n => n.startsWith('guy_'))).toBe(true)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('her_goodit')
  expect(await caption($)).toBe('claude: who got it good?')
})

test('her lines in French', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  expect((await $.command.run(command('cuck', 'lang fr'))).text).toMatch(/French/)
  await $.prompt.submit(PROMPT)
  await clock.advance(60000)
  const hers = heard.sounds.filter(n => n.startsWith('her_'))
  expect(hers.length).toBeGreaterThan(0)
  expect(hers.every(n => n.startsWith('her_fr_'))).toBe(true)
})

test('Christmas: an elf, Santa, and the good gift', async ($, on) => {
  const clock = mock.clock(on, { now: new Date(2026, 11, 20, 20, 0).getTime() })
  mock.store(on)
  const heard = engine(on)
  await $.session.start(SESSION)
  await $.prompt.submit(PROMPT)
  await clock.advance(3000)
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('him_goodgift')
})

test('a voice pack: its own lines, from where cuck pack add put it', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const heard = engine(on, { env: { HOME: '/home/u' } })
  const dir = '/home/u/.config/cuck-code/packs/pirate'
  const read: string[] = []
  on('fs.list', () => ({ value: [{ name: 'pirate', kind: 'dir', size: 0, mtimeMs: 0, isLink: false }] }))
  on('fs.read', (_, e) => {
    read.push(e.path)
    const pack = { name: 'Pirate', her: { 'arr.wav': 'arr, claude!', '../evil.wav': 'no', 'x.mp3': 'no' }, wink: { 'booty.wav': 'claude: who got the good booty?' } }
    return { value: JSON.stringify(pack) }
  })
  await $.session.start(SESSION)
  expect((await $.command.run(command('cuck', 'pack'))).text).toMatch(/Voice packs: pirate/)
  expect((await $.command.run(command('cuck', 'pack pirate'))).text).toBe('Voice pack: Pirate.')
  expect(read).toEqual([`${dir}/pack.json`])
  expect((await $.command.run(command('cuck', 'pack ../../etc'))).text).toMatch(/No voice pack/)

  await $.prompt.submit(PROMPT)
  await clock.advance(40000)
  expect(heard.sounds).toContain('arr')
  expect(heard.sounds).not.toContain('evil')
  await $.turn.complete(DONE)
  await clock.advance(2200)
  expect(heard.sounds.at(-1)).toBe('booty')
  expect(await caption($)).toBe('claude: who got the good booty?')
})

test('the spinner and the closing line are rewritten', async ($, on) => {
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)
    const word = (e.props as { word?: string }).word ?? ''
    return <Text>{word}</Text>
  })
  const spinner = await $.ui.mount({
    plugin: 'cuck-code', component: 'Spinner', requestId: 'main', surface: 'terminal',
    props: { word: 'Sauteing', message: null, suffix: '…', mode: 'thinking' },
  })
  expect(await spinner.find({ type: 'Text', text: /seat|outside|work|listen|door|questions/ })).toBeDefined()

  const line = await $.ui.mount({
    plugin: 'cuck-code', component: 'TurnDuration', requestId: 'done', surface: 'terminal',
    props: { word: 'Baked', durationMs: 32000 },
  })
  expect((await line.find({ type: 'Text', text: /Cucked/ }))?.text).toBe('Cucked')
})
