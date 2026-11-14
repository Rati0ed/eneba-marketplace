import { useEffect, useState } from 'react'
import { fetchQuote, type CartLine, type Quote } from '../api'

// useQuote asks the server to price the cart whenever the cart changes:
// current prices, stock and problems always come from /api/cart/quote, never from the browser.
// Returns the latest quote (the previous one stays visible while a new one loads).
export function useQuote(lines: CartLine[]) {
  // JSON.stringify turns the cart into a string, so the effect re-runs only when its CONTENT
  // changes (a new array with the same items would otherwise count as "changed").
  const cartKey = JSON.stringify(lines)
  const [retryCount, setRetryCount] = useState(0)
  const requestKey = `${cartKey}#${retryCount}`
  const [loaded, setLoaded] = useState<{ key: string; quote: Quote | null; error: string | null }>({
    key: '',
    quote: null,
    error: null,
  })

  useEffect(() => {
    const controller = new AbortController()
    const items = JSON.parse(cartKey) as CartLine[]
    fetchQuote(items, controller.signal)
      .then((quote) => setLoaded({ key: requestKey, quote, error: null }))
      .catch((err: Error) => {
        if (!controller.signal.aborted) setLoaded((prev) => ({ key: requestKey, quote: prev.quote, error: err.message }))
      })
    return () => controller.abort()
  }, [cartKey, requestKey])

  return {
    quote: loaded.quote,
    loading: loaded.key !== requestKey,
    error: loaded.key === requestKey ? loaded.error : null,
    refresh: () => setRetryCount((n) => n + 1),
  }
}
