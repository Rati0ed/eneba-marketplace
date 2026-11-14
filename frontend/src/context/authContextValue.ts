import { createContext } from 'react'
import type { User } from '../api'

export type AuthContextValue = {
  user: User | null // null = not signed in
  token: string | null // the JWT, sent with protected requests
  // "checking" while we ask the server whether a saved token is still valid (after a refresh).
  status: 'checking' | 'ready'
  login: (email: string, password: string) => Promise<void>
  register: (email: string, password: string, displayName: string) => Promise<void>
  logout: () => void
}

// Shared login state: the header, protected pages and the checkout all read it via useAuth().
export const AuthContext = createContext<AuthContextValue | null>(null)
