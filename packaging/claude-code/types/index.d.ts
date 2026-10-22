export type Kind = 'enter' | 'busy' | 'leave'

/** What interrupts the busy loop: he asks you for a yes, and you answer. */
export type EventKind = 'knock' | 'yes' | 'slam'

/** The clip playing in the band above the prompt. */
export type Clip = { kind: Kind; caption: string; startedAt: number }

declare module 'claude-code' {
  interface PluginState {
    'cuck-code': { clip: Clip | null }
  }
}
