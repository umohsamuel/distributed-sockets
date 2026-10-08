import { useEffect, useState } from 'react'
import { ARTICLE_URL, GITHUB_URL, SERVERS } from './config'
import { Chat } from './Chat'
import { RouteMap } from './RouteMap'

export type Role = 'a' | 'b'
export type Session = { room: string; role: Role; serverId: string }

function defaultServer(role: Role): string {
  return SERVERS[role === 'a' ? 0 : Math.min(1, SERVERS.length - 1)].id
}

function readSession(): Session | null {
  const params = new URLSearchParams(location.search)
  const room = params.get('room')?.toLowerCase().replace(/[^a-z0-9]/g, '')
  if (!room) return null
  const role: Role = params.get('as') === 'b' ? 'b' : 'a'
  const requested = params.get('server')
  const serverId = SERVERS.some((s) => s.id === requested) ? requested! : defaultServer(role)
  return { room, role, serverId }
}

function sessionUrl(s: Session | null): string {
  if (!s) return location.pathname
  const params = new URLSearchParams({ room: s.room, as: s.role })
  if (s.serverId !== defaultServer(s.role)) params.set('server', s.serverId)
  return `${location.pathname}?${params}`
}

function newRoomCode(): string {
  const alphabet = 'abcdefghjkmnpqrstuvwxyz23456789'
  return Array.from({ length: 6 }, () => alphabet[Math.floor(Math.random() * alphabet.length)]).join('')
}

export function App() {
  const [session, setSession] = useState<Session | null>(readSession)

  useEffect(() => {
    const onPop = () => setSession(readSession())
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  const go = (next: Session | null, replace = false) => {
    history[replace ? 'replaceState' : 'pushState'](null, '', sessionUrl(next))
    setSession(next)
  }

  if (session) {
    return (
      <Chat
        key={`${session.room}-${session.role}`}
        session={session}
        onLeave={() => go(null)}
        onServerChange={(serverId) => go({ ...session, serverId }, true)}
      />
    )
  }

  return (
    <Landing
      onStart={() => go({ room: newRoomCode(), role: 'a', serverId: defaultServer('a') })}
      onJoin={(room) => go({ room, role: 'b', serverId: defaultServer('b') })}
    />
  )
}

function Landing({ onStart, onJoin }: { onStart: () => void; onJoin: (room: string) => void }) {
  const [code, setCode] = useState('')
  const cleaned = code.toLowerCase().replace(/[^a-z0-9]/g, '')
  const a = defaultServer('a')
  const b = defaultServer('b')

  return (
    <div className="landing">
      <header className="bar">
        <span className="wordmark">distributed-sockets</span>
        <nav className="bar-links">
          <a href={ARTICLE_URL} target="_blank" rel="noreferrer">
            article
          </a>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer">
            github
          </a>
        </nav>
      </header>

      <main className="hero">
        <section className="hero-copy">
          <h1>
            Two devices.
            <br />
            Two servers.
            <br />
            <span className="dim">One conversation.</span>
          </h1>
          <p className="lede">
            A chat where each participant is connected to a different Go server. Neither server holds both
            sockets, so every message is resolved through Redis and handed across through RabbitMQ.
          </p>

          <div className="cta">
            <button className="btn btn-solid" onClick={onStart}>
              start a chat →
            </button>
            <form
              className="join"
              onSubmit={(e) => {
                e.preventDefault()
                if (cleaned) onJoin(cleaned)
              }}
            >
              <input
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="room code"
                aria-label="Room code"
                maxLength={12}
                autoCapitalize="off"
                autoComplete="off"
                spellCheck={false}
              />
              <button className="btn" disabled={!cleaned}>
                join
              </button>
            </form>
          </div>
        </section>

        <section className="hero-figure" aria-label="Message route">
          <p className="label">message route</p>
          <RouteMap yourServer={a} peerServer={b} peerOnline pulse={{ id: 0, direction: 'out', route: 'rabbitmq' }} loop />
        </section>
      </main>

      <ol className="steps">
        <li>
          <span className="step-n">01</span>
          <span>Start a chat. You connect to {a}.</span>
        </li>
        <li>
          <span className="step-n">02</span>
          <span>Scan the QR code on another device. It connects to {b}.</span>
        </li>
        <li>
          <span className="step-n">03</span>
          <span>Send a message and watch the route it takes between servers.</span>
        </li>
      </ol>
    </div>
  )
}
