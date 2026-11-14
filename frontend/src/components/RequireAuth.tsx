import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router'
import { useAuth } from '../hooks/useAuth'

// Wrap a page with <RequireAuth> to make it members-only:
//   <Route path="/orders" element={<RequireAuth><OrdersPage /></RequireAuth>} />
// Not signed in -> /login?next=/orders, and after login the user comes back here.
//
// This is only about USER EXPERIENCE. The real protection is on the server: every order
// endpoint checks the token itself (RequireAuth in backend/handlers/auth.go). Frontend code
// can always be bypassed, so it must never be the only guard.
export default function RequireAuth({ children }: { children: ReactNode }) {
  const { user, status } = useAuth()
  const location = useLocation()

  if (status === 'checking') {
    return (
      <main className="page">
        <p className="muted">Checking your session…</p>
      </main>
    )
  }

  if (!user) {
    const next = location.pathname + location.search
    return <Navigate to={'/login?' + new URLSearchParams({ next })} replace />
  }

  return <>{children}</>
}
