import { useNavigate } from 'react-router'
import { useAuth } from '../hooks/useAuth'

// Shown when the server says our token is no longer valid (401), e.g. after the demo
// database was reset. One click logs out and goes to the login page, then back here.
export default function SessionExpired() {
  const auth = useAuth()
  const navigate = useNavigate()
  return (
    <main className="page">
      <div className="empty-state">
        <h1>Please sign in again</h1>
        <p className="muted">Your session has expired.</p>
        <button
          className="btn btn-primary"
          onClick={() => {
            auth.logout()
            navigate('/login?' + new URLSearchParams({ next: window.location.pathname }))
          }}
        >
          Sign in
        </button>
      </div>
    </main>
  )
}
