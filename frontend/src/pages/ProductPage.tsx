import { Link, useParams } from 'react-router'

// Placeholder: the full product page (details + seller offers) is built on Day 4.
export default function ProductPage() {
  const { slug } = useParams() // reads ":slug" from the URL /product/:slug
  return (
    <main className="page">
      <div className="empty-state">
        <h1>Product page coming soon</h1>
        <p className="muted">{slug}</p>
        <Link to="/store" className="btn btn-primary">
          Back to store
        </Link>
      </div>
    </main>
  )
}
