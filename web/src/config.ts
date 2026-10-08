export type ServerConfig = { id: string; url: string }

const DEFAULT_SERVERS = 'server-1=ws://localhost:9001/ws,server-2=ws://localhost:9002/ws'

export const SERVERS: ServerConfig[] = (import.meta.env.VITE_SERVERS || DEFAULT_SERVERS)
  .split(',')
  .map((entry: string) => entry.trim())
  .filter(Boolean)
  .map((entry: string) => {
    const eq = entry.indexOf('=')
    return { id: entry.slice(0, eq).trim(), url: entry.slice(eq + 1).trim() }
  })

export const GITHUB_URL =
  import.meta.env.VITE_GITHUB_URL || 'https://github.com/umohsamuel/distributed-sockets'
export const ARTICLE_URL =
  import.meta.env.VITE_ARTICLE_URL ||
  'https://www.umohsg.com/blog/distributed-sockets-b54b91e0-adc1-4333-9190-66e28f7b7b19'
export const PORTFOLIO_URL = 'https://www.umohsg.com/'
