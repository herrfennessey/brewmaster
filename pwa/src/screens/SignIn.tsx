import { useState } from 'react'
import { Link, Navigate, useLocation, useNavigate } from 'react-router-dom'
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
        <Link to="/" className="results-back">← Home</Link>
        <h1 className="sign-in-screen__heading">Sign in</h1>
        <p className="sign-in-screen__copy">
          Auth is not configured in this build. Set <code>VITE_FIREBASE_*</code>
          {' '}envs and rebuild, or run the API with <code>DISABLE_AUTH=true</code>.
        </p>
      </div>
    )
  }

  if (user && !isAnonymous) {
    return <Navigate to={next} replace />
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
      <Link to="/" className="results-back">← Home</Link>
      <h1 className="sign-in-screen__heading">
        {isAnonymous ? 'Keep your coffees' : 'Sign in'}
      </h1>
      <p className="sign-in-screen__copy">
        {isAnonymous
          ? 'Right now your saved coffees only live on this device. Link a Google account so they follow you across browsers and survive cache clears.'
          : 'Save your coffees, log brew sessions, and pick up where you left off when you reorder a bag.'}
      </p>
      <button className="action-btn action-btn--primary sign-in-screen__button" disabled={busy} onClick={handleSignIn}>
        {busy ? 'Signing in…' : 'Continue with Google'}
      </button>
      {error && <p className="coffee-section__error">{error}</p>}
    </div>
  )
}
