import { useEffect, useState } from 'react'
import QRCode from 'qrcode'

export function InviteCard({ room, serverLabel }: { room: string; serverLabel: string }) {
  const link = `${location.origin}${location.pathname}?room=${room}&as=b`
  const [qr, setQr] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    QRCode.toDataURL(link, { margin: 2, width: 320, color: { dark: '#000000', light: '#e8e8e6' } })
      .then(setQr)
      .catch(() => setQr(null))
  }, [link])

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {}
  }

  return (
    <div className="invite">
      {qr && <img className="qr" src={qr} alt="QR code for the invite link" width={160} height={160} />}
      <div className="invite-body">
        <p className="label">invite</p>
        <p>
          Scan with a second device or share the link. It joins room <code>{room}</code> on{' '}
          <code>{serverLabel}</code>.
        </p>
        <div className="invite-link">
          <input readOnly value={link} onFocus={(e) => e.target.select()} aria-label="Invite link" />
          <button className="btn" onClick={copy}>
            {copied ? 'copied' : 'copy'}
          </button>
        </div>
        <a className="text-link" href={link} target="_blank" rel="noreferrer">
          open in new tab ↗
        </a>
      </div>
    </div>
  )
}
