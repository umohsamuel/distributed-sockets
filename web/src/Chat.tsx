import { useEffect, useRef, useState } from 'react'
import type { Session } from './App'
import { ARTICLE_URL, GITHUB_URL, SERVERS } from './config'
import { InviteCard } from './InviteCard'
import type { Kind, Route, ServerFrame } from './protocol'
import { uid } from './protocol'
import { RouteMap, type Pulse } from './RouteMap'
import { useSocket } from './useSocket'

type ChatMessage = {
  id: string
  mine: boolean
  body: string
  status: 'sending' | 'routed' | 'failed' | 'received'
  fromServer?: string
  toServer?: string
  route?: Route
  rtt?: number
  error?: string
}

type LogEntry = { id: number; at: string; dir: 'in' | 'out' | 'sys'; text: string }

const QUIET_KINDS: Kind[] = ['typing']

type Props = {
  session: Session
  onLeave: () => void
  onServerChange: (serverId: string) => void
}

export function Chat({ session, onLeave, onServerChange }: Props) {
  const { room, role } = session
  const me = `${room}-${role}`
  const peer = `${room}-${role === 'a' ? 'b' : 'a'}`
  const server = SERVERS.find((s) => s.id === session.serverId) ?? SERVERS[0]
  const peerDefault = SERVERS[role === 'a' ? Math.min(1, SERVERS.length - 1) : 0].id

  const [yourServer, setYourServer] = useState<string | null>(null)
  const [peerServer, setPeerServer] = useState<string | null>(null)
  const [peerOnline, setPeerOnline] = useState(false)
  const [peerTyping, setPeerTyping] = useState(false)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [log, setLog] = useState<LogEntry[]>([])
  const [pulse, setPulse] = useState<Pulse | null>(null)
  const [draft, setDraft] = useState('')
  const [sheetOpen, setSheetOpen] = useState(false)

  const pending = useRef(new Map<string, { kind: Kind; startedAt: number }>())
  const logSeq = useRef(0)
  const pulseSeq = useRef(0)
  const typingTimer = useRef<ReturnType<typeof setTimeout>>(undefined)
  const lastTypingSent = useRef(0)
  const listRef = useRef<HTMLDivElement>(null)

  const addLog = (dir: LogEntry['dir'], text: string) => {
    const at = new Date().toLocaleTimeString([], { hour12: false })
    setLog((l) => [...l.slice(-79), { id: ++logSeq.current, at, dir, text }])
  }

  const firePulse = (direction: Pulse['direction'], route: Route) =>
    setPulse({ id: ++pulseSeq.current, direction, route })

  const updateMessage = (id: string, patch: Partial<ChatMessage>) =>
    setMessages((ms) => ms.map((m) => (m.id === id ? { ...m, ...patch } : m)))

  const onFrame = (frame: ServerFrame) => {
    switch (frame.type) {
      case 'welcome':
        addLog('in', JSON.stringify(frame))
        setYourServer(frame.server_id)
        emit('hello')
        return

      case 'ack': {
        const p = pending.current.get(frame.id)
        pending.current.delete(frame.id)
        setPeerServer(frame.to_server)
        if (!p || !QUIET_KINDS.includes(p.kind)) addLog('in', JSON.stringify(frame))
        if (p?.kind === 'chat') {
          updateMessage(frame.id, {
            status: 'routed',
            route: frame.route,
            toServer: frame.to_server,
            rtt: Math.round(performance.now() - p.startedAt),
          })
          firePulse('out', frame.route)
        }
        return
      }

      case 'error': {
        const p = pending.current.get(frame.id)
        pending.current.delete(frame.id)
        if (p && QUIET_KINDS.includes(p.kind)) return
        addLog('in', JSON.stringify(frame))
        setPeerOnline(false)
        if (p?.kind === 'chat') updateMessage(frame.id, { status: 'failed', error: frame.error })
        return
      }

      case 'message': {
        if (frame.from !== peer) return
        const kind = frame.kind ?? 'chat'
        if (!QUIET_KINDS.includes(kind)) addLog('in', JSON.stringify(frame))
        setPeerServer(frame.from_server)
        setPeerOnline(kind !== 'bye')

        if (kind === 'hello') emit('hello-ack')
        if (kind === 'typing') {
          setPeerTyping(true)
          clearTimeout(typingTimer.current)
          typingTimer.current = setTimeout(() => setPeerTyping(false), 3000)
        }
        if (kind === 'chat') {
          setPeerTyping(false)
          setMessages((ms) => [
            ...ms,
            {
              id: frame.id || uid(),
              mine: false,
              body: frame.body,
              status: 'received',
              fromServer: frame.from_server,
              toServer: frame.to_server,
              route: frame.route,
            },
          ])
          firePulse('in', frame.route)
        }
        return
      }
    }
  }

  const { status, send } = useSocket(`${server.url}?user_id=${encodeURIComponent(me)}`, onFrame)

  function emit(kind: Kind, body = '') {
    const id = uid()
    const msg = { id, kind, to: peer, body }
    if (!send(msg)) return null
    pending.current.set(id, { kind, startedAt: performance.now() })
    if (!QUIET_KINDS.includes(kind)) addLog('out', JSON.stringify(msg))
    return id
  }

  useEffect(() => {
    if (status === 'connecting') addLog('sys', `connecting to ${server.url}`)
    if (status === 'closed') {
      addLog('sys', 'disconnected, retrying')
      setYourServer(null)
    }
  }, [status, server.url])

  const emitRef = useRef(emit)
  useEffect(() => {
    emitRef.current = emit
  })
  useEffect(() => {
    const bye = () => emitRef.current('bye')
    window.addEventListener('pagehide', bye)
    return () => window.removeEventListener('pagehide', bye)
  }, [])

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' })
  }, [messages, peerTyping])

  const leave = () => {
    emit('bye')
    onLeave()
  }

  const submit = () => {
    const body = draft.trim()
    if (!body) return
    const id = emit('chat', body)
    if (!id) return
    setMessages((ms) => [...ms, { id, mine: true, body, status: 'sending', fromServer: yourServer ?? undefined }])
    setDraft('')
  }

  const onDraftChange = (value: string) => {
    setDraft(value)
    const now = Date.now()
    if (value && peerOnline && now - lastTypingSent.current > 2000) {
      lastTypingSent.current = now
      emit('typing')
    }
  }

  const connected = status === 'open' && yourServer !== null
  const routePeerServer = peerServer ?? (peerOnline ? null : peerDefault)
  const sameServer = peerOnline && peerServer === yourServer

  const wire = (
    <>
      <p className="label wire-head">wire · {server.id}</p>
      <div className="log" role="log">
        {log.map((e) => (
          <div key={e.id} className={`log-row log-${e.dir}`}>
            <span className="log-at">{e.at}</span>
            <span className="log-dir">{e.dir === 'in' ? '←' : e.dir === 'out' ? '→' : '·'}</span>
            <span className="log-text">{e.text}</span>
          </div>
        ))}
      </div>
      <div className="wire-foot">
        <p>
          Redis maps each user to the server holding their socket. The sending server publishes to RabbitMQ
          with routing key <code>server.&lt;id&gt;</code>, which only the recipient's server is bound to.
        </p>
        <p className="foot-links">
          <a href={ARTICLE_URL} target="_blank" rel="noreferrer">
            article ↗
          </a>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer">
            github ↗
          </a>
        </p>
      </div>
    </>
  )

  return (
    <div className="chat">
      <header className="bar">
        <button className="wordmark" onClick={leave} title="Leave room">
          distributed-sockets
        </button>
        <span className="bar-sep">/</span>
        <span className="bar-room">{room}</span>
        <div className="spacer" />
        <button className="btn btn-quiet only-narrow" onClick={() => setSheetOpen(true)}>
          wire
        </button>
      </header>

      <div className="chat-grid">
        <aside className="rail">
          <dl className="spec">
            <div>
              <dt>server</dt>
              <dd>
                <select value={server.id} onChange={(e) => onServerChange(e.target.value)} aria-label="Server">
                  {SERVERS.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.id}
                    </option>
                  ))}
                </select>
              </dd>
            </div>
            <div>
              <dt>socket</dt>
              <dd>
                <span className={`state ${connected ? 'state-on' : ''}`} />
                {connected ? 'open' : status === 'connecting' ? 'connecting' : 'retrying'}
              </dd>
            </div>
            <div>
              <dt>peer</dt>
              <dd>
                <span className={`state ${peerOnline ? 'state-on' : ''}`} />
                {peerOnline ? peerServer : 'waiting'}
              </dd>
            </div>
          </dl>

          <div className="rail-route">
            <p className="label">route</p>
            <RouteMap
              yourServer={yourServer ?? server.id}
              peerServer={routePeerServer}
              peerOnline={peerOnline}
              pulse={pulse}
            />
            {sameServer && <p className="note">Both devices share a server, so RabbitMQ is skipped.</p>}
          </div>

          <button className="btn btn-quiet rail-leave" onClick={leave}>
            leave room
          </button>
        </aside>

        <main className="thread">
          <div className="messages" ref={listRef}>
            {messages.length === 0 &&
              (!peerOnline && role === 'a' ? (
                <InviteCard room={room} serverLabel={peerDefault} />
              ) : (
                <p className="empty">{peerOnline ? 'Connected. Send something.' : 'Waiting for the other device.'}</p>
              ))}
            {messages.map((m) => (
              <Bubble key={m.id} m={m} />
            ))}
            {peerTyping && <p className="typing">peer is typing…</p>}
          </div>

          <form
            className="composer"
            onSubmit={(e) => {
              e.preventDefault()
              submit()
            }}
          >
            <input
              value={draft}
              onChange={(e) => onDraftChange(e.target.value)}
              placeholder={connected ? 'message' : 'connecting…'}
              maxLength={1000}
              disabled={!connected}
              aria-label="Message"
              enterKeyHint="send"
            />
            <button className="btn btn-solid" disabled={!connected || !draft.trim()}>
              send
            </button>
          </form>
        </main>

        <aside className="wire">{wire}</aside>
      </div>

      {sheetOpen && (
        <div className="sheet-backdrop" onClick={() => setSheetOpen(false)}>
          <div className="sheet wire" onClick={(e) => e.stopPropagation()}>
            <button className="btn btn-quiet sheet-close" onClick={() => setSheetOpen(false)}>
              close
            </button>
            {wire}
          </div>
        </div>
      )}
    </div>
  )
}

function Bubble({ m }: { m: ChatMessage }) {
  let meta: string
  if (m.status === 'sending') meta = 'sending…'
  else if (m.status === 'failed') meta = `not delivered · ${m.error ?? 'error'}`
  else {
    meta = `${m.fromServer ?? '?'} → ${m.route === 'rabbitmq' ? 'rabbitmq' : 'direct'} → ${m.toServer ?? '?'}`
    if (m.rtt !== undefined) meta += ` · ${m.rtt}ms`
  }

  return (
    <div className={`msg ${m.mine ? 'mine' : 'theirs'}${m.status === 'failed' ? ' failed' : ''}`}>
      <div className="msg-body">{m.body}</div>
      <div className="msg-meta">{meta}</div>
    </div>
  )
}
