import { Link, useParams, useSearchParams } from 'react-router'
import { formatDate, formatPrice, imageUrl, type Order } from '../api'
import KeyBox from '../components/KeyBox'
import SessionExpired from '../components/SessionExpired'
import StatusBadge from '../components/StatusBadge'
import { useApiData } from '../hooks/useApiData'
import { useAuth } from '../hooks/useAuth'
import './OrdersPage.css'

// /orders/:id: one order with its keys. After checkout we land here with ?new=1,
// which adds the "Payment successful" banner.
export default function OrderPage() {
  const { id = '' } = useParams()
  const [searchParams] = useSearchParams()
  const { token } = useAuth()
  const order = useApiData<Order>(`/api/orders/${encodeURIComponent(id)}`, token)
  const justPaid = searchParams.get('new') === '1'

  if (order.error?.status === 401) return <SessionExpired />
  if (order.error?.status === 404) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Order not found</h1>
          <Link to="/orders" className="btn btn-primary">
            Your orders
          </Link>
        </div>
      </main>
    )
  }
  if (order.error) {
    return (
      <main className="page">
        <div className="empty-state">
          <h1>Something went wrong</h1>
          <p className="muted">{order.error.message}</p>
          <button className="btn btn-primary" onClick={order.retry}>
            Try again
          </button>
        </div>
      </main>
    )
  }
  if (!order.data) {
    return (
      <main className="page">
        <p className="muted">Loading order…</p>
      </main>
    )
  }

  const o = order.data
  return (
    <main className="page order-page">
      {justPaid && o.status === 'paid' && (
        <div className="success-banner" role="status">
          <h1>Payment successful 🎉</h1>
          <p>Your keys are ready below. We also saved them in your order history.</p>
        </div>
      )}

      <div className="order-header">
        <h2>Order #{o.id}</h2>
        <StatusBadge status={o.status} />
      </div>
      <p className="muted small">
        {o.paid_at ? `Paid ${formatDate(o.paid_at)}` : `Created ${formatDate(o.created_at)}`}
        {o.card_last4 && ` · Card •••• ${o.card_last4}`} · Total {formatPrice(o.total_cents)}
      </p>

      {o.status === 'pending' && (
        <p className="form-error">This order hasn't been paid yet, so there are no keys. You can pay from the checkout.</p>
      )}
      {o.status === 'failed' && (
        <p className="form-error">This order couldn't be completed because an item sold out. You were not charged.</p>
      )}

      <ul className="order-items">
        {o.items.map((item) => (
          <li key={item.id} className="order-item">
            <img src={imageUrl(item.image_url)} alt="" width={60} height={80} />
            <div className="order-item-body">
              <Link to={`/product/${item.product_slug}`} className="order-item-title">
                {item.product_title}
              </Link>
              <span className="muted small">
                {item.quantity} × {formatPrice(item.unit_price_cents)}
              </span>
              {item.keys.map((code) => (
                <KeyBox key={code} code={code} />
              ))}
            </div>
          </li>
        ))}
      </ul>

      <Link to="/orders" className="link-button">
        ← All orders
      </Link>
    </main>
  )
}
