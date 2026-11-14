import { Link } from 'react-router'

export default function NotFoundPage() {
  return (
    <main className="page">
      <div className="empty-state">
        <h1>Page not found</h1>
        <p className="muted">The page you're looking for doesn't exist.</p>
        <Link to="/store" className="btn btn-primary">
          Go to the store
        </Link>
      </div>
    </main>
  )
}
