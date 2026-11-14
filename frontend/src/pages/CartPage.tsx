import { Link } from 'react-router'
import { formatPrice, imageUrl, type QuoteLine } from '../api'
import { MAX_QUANTITY } from '../context/cartContextValue'
import { useCart } from '../hooks/useCart'
import { useQuote } from '../hooks/useQuote'
import './CartPage.css'

// /cart: the browser knows WHICH offers and HOW MANY; the server tells us the price and stock.
export default function CartPage() {
  const cart = useCart()
  const { quote, loading, error, refresh } = useQuote(cart.lines)

  if (cart.lines.length === 0) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Your cart is empty</h1>
          <p className="muted">Find a game you like and pick the best offer.</p>
          <Link to="/store" className="btn btn-primary">
            Browse the store
          </Link>
        </div>
      </main>
    )
  }

  return (
    <main className="page">
      <h1>Your cart</h1>

      {error && (
        <p className="form-error" role="alert">
          Couldn't check prices: {error}{' '}
          <button className="link-button" onClick={refresh}>
            Try again
          </button>
        </p>
      )}

      {!quote ? (
        <p className="muted">Checking prices and stock…</p>
      ) : (
        <div className={loading ? 'cart-layout refreshing' : 'cart-layout'}>
          <ul className="cart-lines">
            {quote.items.map((line) => (
              <CartLineRow key={line.offer_id} line={line} />
            ))}
          </ul>

          <aside className="cart-summary">
            <div className="cart-total">
              <span>Total</span>
              <strong>{formatPrice(quote.total_cents)}</strong>
            </div>
            <p className="muted small">Prices and stock were just checked with the sellers.</p>
            {quote.can_checkout ? (
              // /checkout needs a login; RequireAuth sends guests to /login?next=/checkout and back.
              <Link to="/checkout" className="btn btn-primary cart-checkout">
                Go to checkout
              </Link>
            ) : (
              <>
                <button className="btn btn-primary cart-checkout" disabled>
                  Go to checkout
                </button>
                <p className="small error-text">Fix the items marked in red first.</p>
              </>
            )}
          </aside>
        </div>
      )}
    </main>
  )
}

// One cart line with quantity buttons and any problem the server reported.
function CartLineRow({ line }: { line: QuoteLine }) {
  const cart = useCart()
  // Can't add more than the seller has, or more than the per-offer limit.
  const maxQuantity = Math.min(line.stock, MAX_QUANTITY)

  if (line.problem === 'not_found') {
    return (
      <li className="cart-line problem">
        <div className="cart-line-info">
          <strong>This offer is no longer available.</strong>
          <span className="muted small">The seller removed it, or the demo store was reset.</span>
        </div>
        <button className="btn btn-secondary btn-small" onClick={() => cart.remove(line.offer_id)}>
          Remove
        </button>
      </li>
    )
  }

  return (
    <li className={line.problem ? 'cart-line problem' : 'cart-line'}>
      <img src={imageUrl(line.image_url)} alt="" width={60} height={80} />
      <div className="cart-line-info">
        <Link to={`/product/${line.product_slug}`} className="cart-line-title">
          {line.product_title}
        </Link>
        <span className="muted small">
          Sold by {line.seller_name} · {formatPrice(line.unit_price_cents)} each
        </span>
        {line.problem === 'out_of_stock' && <span className="error-text small">Sold out. Please remove it or pick another seller.</span>}
        {line.problem === 'not_enough_stock' && (
          <span className="error-text small">
            Only {line.stock} left.{' '}
            <button className="link-button" onClick={() => cart.setQuantity(line.offer_id, line.stock)}>
              Change to {line.stock}
            </button>
          </span>
        )}
      </div>

      <div className="quantity" aria-label={`Quantity for ${line.product_title}`}>
        <button onClick={() => cart.setQuantity(line.offer_id, line.quantity - 1)} disabled={line.quantity <= 1} aria-label="One less">
          −
        </button>
        <span aria-live="polite">{line.quantity}</span>
        <button
          onClick={() => cart.setQuantity(line.offer_id, line.quantity + 1)}
          disabled={line.quantity >= maxQuantity}
          aria-label="One more"
        >
          +
        </button>
      </div>

      <div className="cart-line-total">{formatPrice(line.line_total_cents)}</div>

      <button className="cart-remove" onClick={() => cart.remove(line.offer_id)} aria-label={`Remove ${line.product_title}`}>
        ×
      </button>
    </li>
  )
}
