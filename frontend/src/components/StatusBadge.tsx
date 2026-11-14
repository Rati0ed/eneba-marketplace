import type { Order } from '../api'

// A coloured label for an order's status (colours in pages/OrdersPage.css).
export default function StatusBadge({ status }: { status: Order['status'] }) {
  const labels = { paid: 'Completed', pending: 'Awaiting payment', failed: 'Failed' }
  return <span className={`status-badge ${status}`}>{labels[status]}</span>
}
