import { useContext } from 'react'
import { CartContext } from '../context/cartContextValue'

// useCart() gives any component the shared cart: const { lines, add, count } = useCart()
export function useCart() {
  const cart = useContext(CartContext)
  // A clear error if someone forgets <CartProvider> around the app, instead of "cannot read property of null".
  if (!cart) throw new Error('useCart must be used inside <CartProvider>')
  return cart
}
