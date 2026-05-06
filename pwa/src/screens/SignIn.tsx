import { useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth, authBackendConfigured } from '../services/auth-context'

export default function SignIn() {
  const { signIn, user, isAnonymous } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const next = (location.state as { next?: string } | null)?.next ?? '/coffees'

  if (!authBackendConfigured) {
    return (
      <div className="screen sign-in-screen">
        <h1>Sign in</h1>
        <p style={{ color: 'var(--text-2)' }}>
          Auth is not configured in this build. Set <code>VITE_FIREBASE_*</code>
          {' '}envs and rebuild, or run the API with <code>DISABLE_AUTH=true</code>.
        </p>
        <Link to="/">← Home</Link>
      </div>
    )
  }

  // Already a real (non-anonymous) account — nothing to do.
  if (user && !isAnonymous) {
    navigate(next, { replace: true })
    return null
  }

  async function handleSignIn() {
    setBusy(true)
    setError(null)
    try {
      await signIn()
      navigate(next, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="screen sign-in-screen">
      <h1>{isAnonymous ? 'Keep your coffees' : 'Sign in'}</h1>
      <p style={{ color: 'var(--text-2)' }}>
        {isAnonymous
          ? 'Right now your saved coffees only live on this device. Link a Google account so they follow you across browsers and survive cache clears.'
          : 'Save your coffees, log brew sessions, and pick up where you left off when you reorder a bag.'}
      </p>
      <button className="action-btn action-btn--primary" disabled={busy} onClick={handleSignIn}>
        {busy ? 'Signing in…' : 'Continue with Google'}
      </button>
      {error && <p style={{ color: 'var(--accent-error, #c33)' }}>{error}</p>}
      <Link to="/">← Back</Link>
    </div>
  )
}
