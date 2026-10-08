import type { CSSProperties } from 'react'
import type { Route } from './protocol'

export type Pulse = { id: number; direction: 'out' | 'in'; route: Route }

type Props = {
  yourServer: string | null
  peerServer: string | null
  peerOnline: boolean
  pulse: Pulse | null
  loop?: boolean
}

type Node = { kind: 'node'; key: string; label: string; sub: string; dim?: boolean; lookup?: boolean }
type Item = Node | { kind: 'link'; key: string }

const STEP_MS = 110

export function RouteMap({ yourServer, peerServer, peerOnline, pulse, loop }: Props) {
  const same = peerServer !== null && peerServer === yourServer

  const you: Node = { kind: 'node', key: 'you', label: 'you', sub: 'device' }
  const peer: Node = { kind: 'node', key: 'peer', label: 'peer', sub: peerOnline ? 'device' : 'offline', dim: !peerOnline }
  const mine: Node = { kind: 'node', key: 'yourServer', label: yourServer ?? '…', sub: 'go', lookup: !same }

  const nodes: Node[] = same
    ? [you, mine, peer]
    : [
        you,
        mine,
        { kind: 'node', key: 'rabbit', label: 'rabbitmq', sub: 'exchange' },
        {
          kind: 'node',
          key: 'peerServer',
          label: peerServer ?? '?',
          sub: 'go',
          dim: !peerServer,
          lookup: true,
        },
        peer,
      ]

  const items: Item[] = nodes.flatMap((n, i): Item[] => (i === 0 ? [n] : [{ kind: 'link', key: `l${i}` }, n]))

  const order = pulse?.direction === 'in' ? [...items].reverse() : items
  const step = new Map(order.map((item, i) => [item.key, i]))
  const sender = pulse?.direction === 'in' ? 'peerServer' : 'yourServer'

  const delay = (s: number | undefined) =>
    pulse && s !== undefined ? ({ '--d': `${s * STEP_MS}ms` } as CSSProperties) : undefined
  const lit = pulse ? '' : undefined

  return (
    <div className={`route${loop ? ' route-loop' : ''}`} key={pulse?.id ?? 'idle'}>
      {items.map((item) =>
        item.kind === 'link' ? (
          <div key={item.key} className="link" data-lit={lit} data-dir={pulse?.direction} style={delay(step.get(item.key))} />
        ) : (
          <div key={item.key} className="node" data-dim={item.dim || undefined}>
            <span className="node-box" data-lit={lit} style={delay(step.get(item.key))}>
              {item.label}
            </span>
            <span className="node-sub">{item.sub}</span>
            {item.lookup && (
              <span
                className="lookup"
                data-lit={pulse && item.key === sender ? '' : undefined}
                style={delay((step.get(item.key) ?? 0) + 0.5)}
              >
                redis
              </span>
            )}
          </div>
        ),
      )}
    </div>
  )
}
