import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate } from 'react-router'
import { ApiError, createOrder, formatPrice, payOrder } from '../api'
import { useAuth } from '../hooks/useAuth'
import { useCart } from '../hooks/useCart'
import { useQuote } from '../hooks/useQuote'
import './AuthPage.css' // shared form field styles
import './CheckoutPage.css'

// Test cards, like the ones real payment providers publish for sandboxes.
const TEST_CARDS = [
  { label: 'Payment succeeds', number: '4242 4242 4242 4242' },
  { label: 'Card is declined', number: '4000 0000 0000 0002' },
]

// "4242424242424242" -> "4242 4242 4242 4242" while typing.
function formatCardNumber(value: string): string {
  const digits = value.replace(/\D/g, '').slice(0, 19) // \D = anything that's not a digit
  return digits.replace(/(\d{4})(?=\d)/g, '$1 ') // a space after every 4 digits (if more follow)
}

// "1229" -> "12/29" while typing.
function formatExpiry(value: string): string {
  const digits = value.replace(/\D/g, '').slice(0, 4)
  return digits.length > 2 ? `${digits.slice(0, 2)}/${digits.slice(2)}` : digits
}

// /checkout (members only): order summary + fake card form.
//
// Paying = two API calls:
//   1. POST /api/orders           -> pending order, prices from the database
//   2. POST /api/orders/{id}/pay  -> simulated payment; the server delivers the keys
// If the card is declined we keep the order id, so a retry pays the SAME order
// instead of creating a new one each time.
export default function CheckoutPage() {
  const auth = useAuth()
  const cart = useCart()
  const navigate = useNavigate()
  const { quote, loading: quoteLoading, refresh: refreshQuote } = useQuote(cart.lines)

  const [cardholder, setCardholder] = useState(auth.user?.display_name ?? '')
  const [cardNumber, setCardNumber] = useState('')
  const [expiry, setExpiry] = useState('')
  const [cvc, setCvc] = useState('')

  // The pending order made for THIS cart. If the cart changes, the old order no longer
  // matches, so we remember which cart it was made for.
  const cartKey = JSON.stringify(cart.lines)
  const [pendingOrder, setPendingOrder] = useState<{ id: number; cartKey: string } | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [paidOrderId, setPaidOrderId] = useState<number | null>(null)

  // Order matters: after paying we empty the cart, so the "empty cart" rule below would
  // send the buyer to /cart. Checking for a paid order FIRST sends them to their keys instead.
  // (replace: the Back button won't return to the finished checkout.)
  if (paidOrderId !== null) return <Navigate to={`/orders/${paidOrderId}?new=1`} replace />
  if (cart.lines.length === 0) return <Navigate to="/cart" replace />

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    if (submitting || !auth.token) return
    setSubmitting(true) // the button is disabled now: a double-click can't pay twice
    setError(null)

    try {
      // Step 1: reuse the pending order if it was made for this exact cart, otherwise create one.
      let orderId = pendingOrder?.cartKey === cartKey ? pendingOrder.id : null
      if (orderId === null) {
        const order = await createOrder(cart.lines, auth.token)
        orderId = order.id
        setPendingOrder({ id: order.id, cartKey })
      }

      // Step 2: pay.
      const paid = await payOrder(orderId, { cardholder, card_number: cardNumber, expiry, cvc }, auth.token)
      setPaidOrderId(paid.id) // the next render redirects to the order page (see above)
      cart.clear()
    } catch (err) {
      setSubmitting(false)
      if (!(err instanceof ApiError)) {
        setError('Something went wrong. Please try again.')
        return
      }
      switch (err.status) {
        case 401: // session expired: sign in again, then come back here (the cart is kept)
          auth.logout()
          navigate('/login?' + new URLSearchParams({ next: '/checkout' }))
          return
        case 409: // something sold out / not enough stock: that order is dead, re-check the cart
          setPendingOrder(null)
          refreshQuote()
          setError(err.message)
          return
        default: // 402 declined, 400 invalid card, ...: show the server's message, keep the order
          setError(err.message)
      }
    }
  }

  return (
    <main className="page">
      <h1>Checkout</h1>
      <div className="checkout-layout">
        <form className="checkout-card" onSubmit={handleSubmit} noValidate>
          <h2>Payment</h2>

          <div className="test-cards">
            <strong>This is a demo: no real money is charged.</strong>
            <span className="muted">Use any name, a future expiry date and any 3-digit CVC with a test card:</span>
            {TEST_CARDS.map((card) => (
              <button key={card.number} type="button" className="test-card" onClick={() => setCardNumber(card.number)}>
                <code>{card.number}</code> <span className="muted">{card.label}</span>
              </button>
            ))}
          </div>

          {error && (
            <p className="form-error" role="alert">
              {error}
            </p>
          )}

          <label className="field">
            <span>Name on card</span>
            <input value={cardholder} onChange={(e) => setCardholder(e.target.value)} autoComplete="cc-name" maxLength={60} required />
          </label>
          <label className="field">
            <span>Card number</span>
            <input
              value={cardNumber}
              onChange={(e) => setCardNumber(formatCardNumber(e.target.value))}
              inputMode="numeric" // number keyboard on phones
              autoComplete="cc-number"
              placeholder="4242 4242 4242 4242"
              required
            />
          </label>
          <div className="field-row">
            <label className="field">
              <span>Expiry (MM/YY)</span>
              <input
                value={expiry}
                onChange={(e) => setExpiry(formatExpiry(e.target.value))}
                inputMode="numeric"
                autoComplete="cc-exp"
                placeholder="12/29"
                required
              />
            </label>
            <label className="field">
              <span>CVC</span>
              <input
                value={cvc}
                onChange={(e) => setCvc(e.target.value.replace(/\D/g, '').slice(0, 4))}
                inputMode="numeric"
                autoComplete="cc-csc"
                placeholder="123"
                required
              />
            </label>
          </div>

          <button
            type="submit"
            className="btn btn-primary checkout-pay"
            disabled={submitting || !quote?.can_checkout || quoteLoading}
          >
            {submitting ? 'Processing payment…' : quote ? `Pay ${formatPrice(quote.total_cents)}` : 'Pay'}
          </button>
        </form>

        <aside className="checkout-card">
          <h2>Order summary</h2>
          {!quote ? (
            <p className="muted">Checking prices…</p>
          ) : (
            <>
              <ul className="summary-lines">
                {quote.items.map((line) => (
                  <li key={line.offer_id}>
                    <span>
                      {line.quantity} × {line.product_title}
                      <span className="muted small"> · {line.seller_name}</span>
                      {line.problem && <span className="error-text small"> · not available</span>}
                    </span>
                    <strong>{formatPrice(line.line_total_cents)}</strong>
                  </li>
                ))}
              </ul>
              <div className="cart-total">
                <span>Total</span>
                <strong>{formatPrice(quote.total_cents)}</strong>
              </div>
              {!quote.can_checkout && (
                <p className="error-text small">
                  Some items are no longer available. <Link to="/cart">Review your cart</Link>.
                </p>
              )}
            </>
          )}
        </aside>
      </div>
    </main>
  )
}
