export type Route = 'local' | 'rabbitmq'
export type Kind = 'chat' | 'hello' | 'hello-ack' | 'typing' | 'bye'

export type OutgoingMessage = { id: string; kind: Kind; to: string; body: string }

export type WelcomeFrame = { type: 'welcome'; server_id: string; user_id: string }
export type AckFrame = { type: 'ack'; id: string; to: string; to_server: string; route: Route }
export type ErrorFrame = { type: 'error'; id: string; to: string; error: string }
export type MessageFrame = {
  type: 'message'
  id: string
  kind?: Kind
  from: string
  to: string
  body: string
  from_server: string
  to_server: string
  route: Route
  sent_at: number
}

export type ServerFrame = WelcomeFrame | AckFrame | ErrorFrame | MessageFrame

export function uid(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto && window.isSecureContext) {
    return crypto.randomUUID()
  }
  return Math.random().toString(36).slice(2) + Date.now().toString(36)
}
