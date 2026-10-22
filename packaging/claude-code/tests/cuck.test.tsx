import { expect, mock, test } from 'claude-code/testing'
import type { On } from 'claude-code'

const BAND = {
  plugin: 'cuck-code',
  component: 'AbovePrompt',
  requestId: 'band',
  props: { hasSurvey: false, isWorking: true, maxRows: 20, bodyColumns: 100, scroll: { offset: 0, bodyRows: 19 }, view: {} },
} as const

const PROMPT = { text: 'fix the login bug', wait: false, origin: { kind: 'composer' } } as const
const TURN = { answer: 'done', durationMs: 32000, turnId: 't1' }
const QUIET = { command: 'cuck-quiet', args: '', origin: { kind: 'composer' }, presentation: { isFullscreen: false, columns: 100 } } as const

// The engine beneath: it takes prompts and turns, runs commands, and draws
// its own band. Every sound the mod plays is recorded.
function engine(on: On): string[] {
  const played: string[] = []
  on('prompt.submit', (_, e) => ({ text: e.text }))
  on('turn.complete', () => ({ text: 'done' }))
  on('command.register', () => ({ value: { command: 'cuck-quiet' } }))
  on('session.start', (_, e) => ({ cwd: e.cwd }))
  on('process.run', (_, e) => {
    played.push((e.argv.at(-1) ?? '').replace(/^.*\//, '').replace('.wav', ''))
    return { value: { exitCode: 0, stdout: '', stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)
    return <Text>engine</Text>
  })
  return played
}

test('she walks in, they carry on while Claude works, she walks out, the band clears', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  const played = engine(on)

  await $.prompt.submit(PROMPT)
  const first = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await first.find({ key: 'strip' })).toBeDefined()
  await first.unmount()

  // Well past the walk-in: still going while the turn runs, with sound.
  await clock.advance(15000)
  const busy = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await busy.find({ key: 'strip' })).toBeDefined()
  await busy.unmount()
  expect(played.some(s => s.startsWith('creak'))).toBe(true)
  expect(played.some(s => s.startsWith('her_') || s.startsWith('him_'))).toBe(true)

  await $.turn.complete({ ...TURN, isAborted: false, reason: 'answer' })
  expect(played.at(-1)).toBe('ding')
  const leaving = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await leaving.find({ key: 'strip' })).toBeDefined()
  await leaving.unmount()

  // He turns to you, winks, and says it.
  await clock.advance(2200)
  expect(played.at(-1)).toBe('him_gooddee')

  await clock.advance(3200)
  const after = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await after.find({ key: 'strip' })).toBeUndefined()
  expect((await after.find({ type: 'Text', text: /engine/ }))?.text).toBe('engine')
  await after.unmount()
})

test('narrow terminals and other surfaces get the caption alone', async ($, on) => {
  mock.clock(on, { now: 1000 })
  engine(on)
  await $.prompt.submit(PROMPT)
  const narrow = await $.ui.mount({ ...BAND, surface: 'terminal', props: { ...BAND.props, bodyColumns: 30 } })
  expect(await narrow.find({ type: 'Text', text: /claude|seat|wait|door/ })).toBeDefined()
  await narrow.unmount()
  const desktop = await $.ui.mount({ ...BAND, surface: 'desktop' })
  expect(await desktop.find({ type: 'Text', text: /claude|seat|wait|door/ })).toBeDefined()
  await desktop.unmount()
})

test('an interrupt clears the band; subagents and empty prompts play nothing', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  engine(on)

  await $.prompt.submit({ ...PROMPT, text: '   ' })
  await $.turn.complete({ ...TURN, isAborted: false, reason: 'answer', agentId: 'sub' })
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

test('/cuck-quiet mutes everything, and unmutes', async ($, on) => {
  const clock = mock.clock(on, { now: 1000 })
  mock.store(on)
  const played = engine(on)
  await $.session.start({ cwd: '/', surface: 'terminal', isInteractive: true })

  const muted = await $.command.run(QUIET)
  expect(muted.text).toMatch(/muted/)
  await $.prompt.submit(PROMPT)
  await clock.advance(15000)
  await $.turn.complete({ ...TURN, isAborted: false, reason: 'answer' })
  expect(played).toEqual([])

  const back = await $.command.run(QUIET)
  expect(back.text).toMatch(/back on/)
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
