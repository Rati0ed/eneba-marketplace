import { createContext } from 'react'
import type { CartLine } from '../api'

// The cart's context object and its type live in this separate file, so CartContext.tsx
// only exports a component (Vite's hot reload needs component files to export only components).

export const MAX_QUANTITY = 10 // per offer; the server enforces the same limit

export type CartContextValue = {
  lines: CartLine[]
  count: number // total number of keys, for the header badge
  add: (offerId: number, maxQuantity: number) => void
  setQuantity: (offerId: number, quantity: number) => void
  remove: (offerId: number) => void
  clear: () => void
}

// Context = a value that any component below the Provider can read, without passing it
// through every component in between ("prop drilling"). The header badge, the product page
// and the cart page all read the same cart. useCart() in hooks/useCart.ts reads it.
export const CartContext = createContext<CartContextValue | null>(null)
