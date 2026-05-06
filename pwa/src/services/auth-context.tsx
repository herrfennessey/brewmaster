/* eslint-disable react-refresh/only-export-components --
 * AuthProvider lives next to its hook so callers have a single import surface.
 * HMR isn't worth the file split. */
import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { onIdTokenChanged, type User } from 'firebase/auth'
import { authBackendConfigured, ensureAnonymous, firebaseAuth, googleSignOut, upgradeToGoogle } from './firebase'
import { setAuthFailureHandler } from './api'

interface AuthState {
  user: User | null
  loading: boolean
  // ready means the API can be called: either Firebase has settled with a
  // user (anon or otherwise), or auth is unconfigured and the server runs
  // with DISABLE_AUTH=true. Screens should gate their fetches on this, not
  // on user.uid — otherwise they wedge in dev mode where user is null forever.
  ready: boolean
  isAnonymous: boolean
  anonError: Error | null
  signIn: () => Promise<void>
  signOut: () => Promise<void>
  getIdToken: () => Promise<string | null>
}

const AuthContext = createContext<AuthState | null>(null)

// recoverFromRevokedToken is invoked by api.ts when the server returns 401.
// Most likely cause: the cached anonymous user was deleted server-side
// (admin action, GDPR cleanup) and the local SDK still holds a stale token.
// Signing out forces onIdTokenChanged to fire with null, which mints a fresh
// anonymous user instead of leaving the app wedged on the dead identity.
async function recoverFromRevokedToken(): Promise<void> {
  if (!firebaseAuth?.currentUser) return
  await googleSignOut()
}

setAuthFailureHandler(recoverFromRevokedToken)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(authBackendConfigured)
  const [anonError, setAnonError] = useState<Error | null>(null)

  useEffect(() => {
    if (!firebaseAuth) return
    const unsubscribe = onIdTokenChanged(firebaseAuth, next => {
      setUser(next)
      setLoading(false)
      // No user yet on this device — bootstrap an anonymous one so the app
      // works without a sign-in wall. The next id-token-changed fires when
      // the anon user is ready.
      if (!next) {
        ensureAnonymous()
          .then(() => setAnonError(null))
          .catch(err => {
            console.error('anonymous sign-in failed', err)
            setAnonError(err instanceof Error ? err : new Error('Anonymous sign-in failed'))
          })
      } else {
        setAnonError(null)
      }
    })
    return unsubscribe
  }, [])

  const isAnonymous = Boolean(user?.isAnonymous)
  const ready = !authBackendConfigured || (!loading && user !== null)

  const value = useMemo<AuthState>(() => ({
    user,
    loading,
    ready,
    isAnonymous,
    anonError,
    signIn: upgradeToGoogle,
    signOut: googleSignOut,
    async getIdToken() {
      return firebaseAuth?.currentUser ? firebaseAuth.currentUser.getIdToken() : null
    },
  }), [user, loading, ready, isAnonymous, anonError])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}

export { authBackendConfigured }
