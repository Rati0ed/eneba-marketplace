import { useContext } from 'react'
import { AuthContext } from '../context/authContextValue'

// useAuth() gives any component the login state: const { user, token, login, logout } = useAuth()
export function useAuth() {
  const auth = useContext(AuthContext)
  if (!auth) throw new Error('useAuth must be used inside <AuthProvider>')
  return auth
}

// After login we go back to where the user came from: /login?next=/checkout -> /checkout.
// `next` comes from the URL, so anyone could craft a link like /login?next=https://evil.com.
// Following it blindly would be an "open redirect" (our trusted login page sending users to a
// phishing site). We only accept paths on our own site.
export function safeNextPath(next: string | null): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/'
  return next
}
