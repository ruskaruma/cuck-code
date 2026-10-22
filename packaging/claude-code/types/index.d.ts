export type Kind = 'enter' | 'busy' | 'leave'

/** The clip playing in the band above the prompt. */
export type Clip = { kind: Kind; caption: string; startedAt: number }

declare module 'claude-code' {
  interface PluginState {
    'cuck-code': { clip: Clip | null }
  }
}
