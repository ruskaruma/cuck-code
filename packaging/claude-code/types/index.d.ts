/**
 * A clip: she walks in, the busy loop, he walks out; a breather with no turn
 * running (a /compact); Claude saying no; the morning after.
 */
export type Kind = 'enter' | 'busy' | 'leave' | 'breather' | 'refused' | 'morning'

/**
 * What interrupts the busy loop: he asks you for a yes and you answer, his
 * friends arrive, they take a breather, visitors, the tests.
 */
export type EventKind = 'knock' | 'yes' | 'slam' | 'friend' | 'breather' | 'pizza' | 'neighbour' | 'landlord' | 'flop' | 'passed'

/** The clip playing in the band above the prompt. */
export type Clip = { kind: Kind; caption: string; startedAt: number }

declare module 'claude-code' {
  interface PluginState {
    'cuck-code': { clip: Clip | null }
  }
}
