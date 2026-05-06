/* eslint-disable react-refresh/only-export-components --
 * AuthProvider lives next to its hook so callers have a single import surface.
 * HMR isn't worth the file split. */
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { onIdTokenChanged, type User } from 'firebase/auth'
import { authBackendConfigured, ensureAnonymous, firebaseAuth, googleSignOut, upgradeToGoogle } from './firebase'

interface AuthState {
  user: User | null
  loading: boolean
  isAnonymous: boolean
  signIn: () => Promise<void>
  signOut: () => Promise<void>
  getIdToken: () => Promise<string | null>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(authBackendConfigured)

  useEffect(() => {
    if (!firebaseAuth) return
    const unsubscribe = onIdTokenChanged(firebaseAuth, next => {
      setUser(next)
      setLoading(false)
      // No user yet on this device — bootstrap an anonymous one so the app
      // works without a sign-in wall. The next id-token-changed fires when
      // the anon user is ready.
      if (!next) {
        ensureAnonymous().catch(err => {
          console.error('anonymous sign-in failed', err)
        })
      }
    })
    return unsubscribe
  }, [])

  const isAnonymous = Boolean(user?.isAnonymous)

  const value = useMemo<AuthState>(() => ({
    user,
    loading,
    isAnonymous,
    signIn: upgradeToGoogle,
    signOut: googleSignOut,
    async getIdToken() {
      return firebaseAuth?.currentUser ? firebaseAuth.currentUser.getIdToken() : null
    },
  }), [user, loading, isAnonymous])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}

export { authBackendConfigured }
