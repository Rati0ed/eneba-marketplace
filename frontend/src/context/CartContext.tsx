import { useEffect, useState, type ReactNode } from 'react'
import type { CartLine } from '../api'
import { CartContext, MAX_QUANTITY } from './cartContextValue'

// The cart lives in the browser (localStorage), not in the database:
//   - guests can fill a cart before creating an account (like on Eneba), and
//   - it costs no server requests to add/remove items.
// It only stores offer ids and quantities. Prices and stock always come from the server
// (/api/cart/quote and /api/orders), so editing localStorage by hand can't change a price.

const STORAGE_KEY = 'gamekey.cart.v1' // ".v1": if the format ever changes, old data is simply ignored

// Read the saved cart. localStorage can be blocked (private mode) or hold broken or
// hand-edited data, so we check every line and fall back to an empty cart.
function loadCart(): CartLine[] {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]')
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (line): line is CartLine =>
        Number.isInteger(line?.offer_id) &&
        line.offer_id > 0 &&
        Number.isInteger(line?.quantity) &&
        line.quantity >= 1 &&
        line.quantity <= MAX_QUANTITY,
    )
  } catch {
    return []
  }
}

export function CartProvider({ children }: { children: ReactNode }) {
  // useState(loadCart): passing the FUNCTION (not loadCart()) means React calls it only once,
  // on the first render, instead of reading localStorage on every render.
  const [lines, setLines] = useState<CartLine[]>(loadCart)

  // Save on every change.
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(lines))
    } catch {
      // Storage blocked or full: the cart still works for this visit, it just won't be remembered.
    }
  }, [lines])

  // Another tab changed the cart -> update this tab too. The "storage" event fires only in OTHER tabs.
  useEffect(() => {
    function onStorage(event: StorageEvent) {
      if (event.key === STORAGE_KEY) setLines(loadCart())
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  // Every update creates a NEW array (map/filter/spread) instead of changing the old one.
  // React compares old and new state by reference; changing the old array in place
  // would look like "nothing changed" and the screen wouldn't update.
  function add(offerId: number, maxQuantity: number) {
    const limit = Math.min(maxQuantity, MAX_QUANTITY)
    setLines((current) => {
      const existing = current.find((line) => line.offer_id === offerId)
      if (!existing) return [...current, { offer_id: offerId, quantity: 1 }]
      return current.map((line) =>
        line.offer_id === offerId ? { ...line, quantity: Math.min(line.quantity + 1, limit) } : line,
      )
    })
  }

  function setQuantity(offerId: number, quantity: number) {
    const safe = Math.max(1, Math.min(Math.floor(quantity), MAX_QUANTITY))
    setLines((current) => current.map((line) => (line.offer_id === offerId ? { ...line, quantity: safe } : line)))
  }

  function remove(offerId: number) {
    setLines((current) => current.filter((line) => line.offer_id !== offerId))
  }

  function clear() {
    setLines([])
  }

  // reduce adds up all quantities: [{q:2},{q:1}] -> 3. Like sum(l.quantity for l in lines) in Python.
  const count = lines.reduce((sum, line) => sum + line.quantity, 0)

  return (
    <CartContext.Provider value={{ lines, count, add, setQuantity, remove, clear }}>{children}</CartContext.Provider>
  )
}
