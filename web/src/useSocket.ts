import { useCallback, useEffect, useRef, useState } from 'react'
import type { OutgoingMessage, ServerFrame } from './protocol'

export type SocketStatus = 'connecting' | 'open' | 'closed'

export function useSocket(url: string, onFrame: (frame: ServerFrame) => void) {
  const [status, setStatus] = useState<SocketStatus>('connecting')
  const wsRef = useRef<WebSocket | null>(null)
  const handlerRef = useRef(onFrame)

  useEffect(() => {
    handlerRef.current = onFrame
  })

  useEffect(() => {
    let stopped = false
    let attempt = 0
    let timer: ReturnType<typeof setTimeout> | undefined

    const connect = () => {
      setStatus('connecting')
      const ws = new WebSocket(url)
      wsRef.current = ws
      ws.onopen = () => {
        attempt = 0
        setStatus('open')
      }
      ws.onmessage = (e) => {
        try {
          handlerRef.current(JSON.parse(e.data))
        } catch {}
      }
      ws.onclose = () => {
        if (wsRef.current === ws) wsRef.current = null
        if (stopped) return
        setStatus('closed')
        timer = setTimeout(connect, Math.min(1000 * 2 ** attempt++, 10_000))
      }
    }

    connect()
    return () => {
      stopped = true
      clearTimeout(timer)
      wsRef.current?.close()
      wsRef.current = null
    }
  }, [url])

  const send = useCallback((msg: OutgoingMessage) => {
    const ws = wsRef.current
    if (ws?.readyState !== WebSocket.OPEN) return false
    ws.send(JSON.stringify(msg))
    return true
  }, [])

  return { status, send }
}
