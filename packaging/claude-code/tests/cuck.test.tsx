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
  const heard = { sounds: [] as string[], players: [] as string[], notices: [] as string[] }
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
  expect(await spinner.find({ type: 'Text', text: /chair|seat|work|listen|door|questions/ })).toBeDefined()

  const line = await $.ui.mount({
    plugin: 'cuck-code', component: 'TurnDuration', requestId: 'done', surface: 'terminal',
    props: { word: 'Baked', durationMs: 32000 },
  })
  expect((await line.find({ type: 'Text', text: /Cucked/ }))?.text).toBe('Cucked')
})
