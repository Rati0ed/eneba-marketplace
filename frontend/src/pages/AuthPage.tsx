import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router'
import { safeNextPath, useAuth } from '../hooks/useAuth'
import './AuthPage.css'

// One component for both /login and /register; `mode` decides which fields and texts to show.
export default function AuthPage({ mode }: { mode: 'login' | 'register' }) {
  const auth = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const next = safeNextPath(searchParams.get('next'))

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const isRegister = mode === 'register'

  // Already signed in (e.g. pressed Back after logging in): go straight to the target page.
  if (auth.user) return <Navigate to={next} replace />

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    // Quick check in the browser for instant feedback. The SERVER checks again, because
    // anything in the browser can be bypassed.
    if (isRegister && password.length < 8) {
      setError('Password must be at least 8 characters long.')
      return
    }
    setError(null)
    setSubmitting(true) // disables the button: no double submits
    try {
      if (isRegister) await auth.register(email, password, displayName)
      else await auth.login(email, password)
      navigate(next, { replace: true }) // replace: "Back" won't return to the login form
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong.')
      setSubmitting(false)
    }
  }

  // Keep ?next= when switching between login and register.
  const otherLink = (isRegister ? '/login' : '/register') + (next !== '/' ? '?' + new URLSearchParams({ next }) : '')

  return (
    <main className="page auth-page">
      <form className="auth-card" onSubmit={handleSubmit} noValidate>
        <h1>{isRegister ? 'Create your account' : 'Sign in'}</h1>
        {next.startsWith('/checkout') && <p className="muted">Sign in to finish your purchase. Your cart is saved.</p>}

        {/* role="alert" makes screen readers announce the error as soon as it appears */}
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}

        <label className="field">
          <span>Email</span>
          <input
            type="email"
            autoComplete="email" // lets password managers fill it in
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            maxLength={254}
          />
        </label>

        {isRegister && (
          <label className="field">
            <span>
              Display name <span className="muted">(optional)</span>
            </span>
            <input
              type="text"
              autoComplete="nickname"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              maxLength={40}
            />
          </label>
        )}

        {/* The Show/Hide button must not sit INSIDE the <label> (screen readers would read
            "Password Show"), so the label points at the input with htmlFor + id instead. */}
        <div className="field">
          <label htmlFor="password">Password</label>
          <div className="password-row">
            <input
              id="password"
              type={showPassword ? 'text' : 'password'}
              // "new-password" tells password managers to suggest a strong one on sign-up
              autoComplete={isRegister ? 'new-password' : 'current-password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              minLength={isRegister ? 8 : undefined}
              maxLength={72}
            />
            <button
              type="button"
              className="link-button"
              onClick={() => setShowPassword(!showPassword)}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>
          {isRegister && <small className="muted">At least 8 characters.</small>}
        </div>

        <button type="submit" className="btn btn-primary auth-submit" disabled={submitting}>
          {submitting ? 'Please wait…' : isRegister ? 'Create account' : 'Sign in'}
        </button>

        <p className="muted auth-switch">
          {isRegister ? 'Already have an account?' : 'New here?'} <Link to={otherLink}>{isRegister ? 'Sign in' : 'Create an account'}</Link>
        </p>
      </form>
    </main>
  )
}
