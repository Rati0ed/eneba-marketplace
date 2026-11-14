import { useState } from 'react'

// One digital key: hidden by default (people share screenshots), with Reveal/Hide and Copy.
export default function KeyBox({ code }: { code: string }) {
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState<'yes' | 'failed' | null>(null)

  async function copy() {
    try {
      // The Clipboard API only works on HTTPS (or localhost) and may ask for permission.
      await navigator.clipboard.writeText(code)
      setCopied('yes')
    } catch {
      setCopied('failed') // e.g. permission denied: the user can still reveal and copy by hand
      setRevealed(true)
    }
    setTimeout(() => setCopied(null), 2000)
  }

  // Replace every letter/digit with a dot but keep the dashes: "ABCDE-FGHJK" -> "•••••-•••••"
  const masked = code.replace(/[^-]/g, '•')

  return (
    <div className="key-box">
      {/* aria-live: screen readers announce the key when it's revealed */}
      <code className="key-code" aria-live="polite">
        {revealed ? code : masked}
      </code>
      <span className="key-demo-tag" title="Generated demo key: it won't work on any store">
        Demo key
      </span>
      <button type="button" className="btn btn-secondary btn-small" onClick={() => setRevealed(!revealed)}>
        {revealed ? 'Hide' : 'Reveal'}
      </button>
      <button type="button" className="btn btn-primary btn-small" onClick={copy}>
        {copied === 'yes' ? 'Copied ✓' : copied === 'failed' ? 'Copy failed' : 'Copy'}
      </button>
    </div>
  )
}
