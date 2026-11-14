import { useEffect, useState, type ReactNode } from 'react'
import { ApiError, fetchMe, login as apiLogin, register as apiRegister, type User } from '../api'
import { AuthContext } from './authContextValue'

const TOKEN_KEY = 'gamekey.token'

// Safe localStorage helpers: storage can be blocked (private mode) or throw.
function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

function writeToken(token: string | null) {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token)
    else localStorage.removeItem(TOKEN_KEY)
  } catch {
    // Can't save: the user stays signed in until they close the tab.
  }
}

// AuthProvider keeps the login state for the whole app.
//
// Surviving a refresh: the token is saved in localStorage. On page load we read it and ask
// GET /api/auth/me "who is this?". The server checks the signature and expiry, and that the
// account still exists. If it says 401, the token is useless and we forget it.
//
// Trade-off (also in ARCHITECTURE.md): localStorage can be read by JavaScript, so an XSS bug
// could steal the token. React escapes all text we render, we never inject raw HTML, and
// tokens expire after 24 h. A cookie would be safer but is blocked between our two domains.
export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(readToken)
  const [user, setUser] = useState<User | null>(null)
  // Lazy initial state: if there's a saved token, we start in "checking" until /me answers.
  const [status, setStatus] = useState<'checking' | 'ready'>(() => (readToken() ? 'checking' : 'ready'))

  // On first load only: check a saved token. (The empty [] list = run once.)
  useEffect(() => {
    const saved = readToken()
    if (!saved) return
    const controller = new AbortController()

    fetchMe(saved, controller.signal)
      .then((me) => setUser(me))
      .catch((err) => {
        if (controller.signal.aborted) return
        // 401 = token expired/invalid, or the account is gone (the demo database resets on
        // restart). Forget it. Other errors (server asleep) keep the token, but show signed-out.
        if (err instanceof ApiError && err.status === 401) {
          writeToken(null)
          setToken(null)
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setStatus('ready')
      })

    return () => controller.abort()
  }, [])

  function saveSession(newToken: string, newUser: User) {
    writeToken(newToken)
    setToken(newToken)
    setUser(newUser)
    setStatus('ready')
  }

  async function login(email: string, password: string) {
    const res = await apiLogin(email, password) // throws ApiError with the server's message on failure
    saveSession(res.token, res.user)
  }

  async function register(email: string, password: string, displayName: string) {
    const res = await apiRegister(email, password, displayName)
    saveSession(res.token, res.user)
  }

  // Logging out with JWTs = the browser forgets the token. The server keeps no sessions,
  // so there's nothing to delete there (a stolen token stays valid until it expires: 24 h).
  function logout() {
    writeToken(null)
    setToken(null)
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, token: user ? token : null, status, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
