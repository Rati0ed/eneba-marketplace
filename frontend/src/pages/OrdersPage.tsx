import { Link } from 'react-router'
import { formatDate, formatPrice, type Order } from '../api'
import { useApiData } from '../hooks/useApiData'
import { useAuth } from '../hooks/useAuth'
import SessionExpired from '../components/SessionExpired'
import StatusBadge from '../components/StatusBadge'
import './OrdersPage.css'

// /orders: order history, newest first. Each order links to its page with the keys.
export default function OrdersPage() {
  const { token } = useAuth()
  const orders = useApiData<{ items: Order[] }>('/api/orders', token)

  if (orders.error?.status === 401) return <SessionExpired />

  return (
    <main className="page">
      <h1>Your orders</h1>

      {orders.error && (
        <p className="form-error">
          {orders.error.message}{' '}
          <button className="link-button" onClick={orders.retry}>
            Try again
          </button>
        </p>
      )}
      {orders.loading && <p className="muted">Loading orders…</p>}

      {orders.data && orders.data.items.length === 0 && (
        <div className="empty-state">
          <h2>No orders yet</h2>
          <p className="muted">When you buy something, your keys will be kept here.</p>
          <Link to="/store" className="btn btn-primary">
            Browse the store
          </Link>
        </div>
      )}

      {orders.data && orders.data.items.length > 0 && (
        <ul className="order-list">
          {orders.data.items.map((order) => {
            const keyCount = order.items.reduce((sum, item) => sum + item.keys.length, 0)
            return (
              <li key={order.id}>
                <Link to={`/orders/${order.id}`} className="order-row">
                  <div>
                    <strong>Order #{order.id}</strong>
                    <div className="muted small">{formatDate(order.paid_at ?? order.created_at)}</div>
                  </div>
                  <div className="order-row-items">
                    {order.items.map((item) => (
                      <div key={item.id}>
                        {item.quantity} × {item.product_title}
                      </div>
                    ))}
                  </div>
                  <StatusBadge status={order.status} />
                  <div className="order-row-total">
                    <strong>{formatPrice(order.total_cents)}</strong>
                    {keyCount > 0 && (
                      <div className="small link-like">
                        View {keyCount} {keyCount === 1 ? 'key' : 'keys'} ›
                      </div>
                    )}
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      )}
    </main>
  )
}
