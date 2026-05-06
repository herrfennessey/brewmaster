import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listCoffeesAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import type { CoffeeSummary } from '../types'

function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const days = Math.floor(ms / (1000 * 60 * 60 * 24))
  if (days <= 0) return 'today'
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days}d ago`
  if (days < 30) return `${Math.floor(days / 7)}w ago`
  return `${Math.floor(days / 30)}mo ago`
}

function Stars({ rating }: { rating?: number }) {
  if (!rating) return <span className="my-coffee__rating my-coffee__rating--empty">unrated</span>
  return (
    <span className="my-coffee__rating">
      {'★'.repeat(rating)}{'☆'.repeat(5 - rating)}
    </span>
  )
}

export default function MyCoffees() {
  const { user, loading: authLoading, isAnonymous } = useAuth()
  const [coffees, setCoffees] = useState<CoffeeSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (authLoading || !user) return
    listCoffeesAPI()
      .then(r => setCoffees(r.coffees))
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [user, authLoading])

  if (authLoading || coffees === null) {
    return (
      <div className="screen my-coffees-screen">
        <Link to="/" className="results-back">← Home</Link>
        <h1>My coffees</h1>
        {error
          ? <p style={{ color: 'var(--accent-error, #c33)' }}>{error}</p>
          : <p style={{ color: 'var(--text-2)' }}>Loading…</p>}
      </div>
    )
  }

  return (
    <div className="screen my-coffees-screen">
      <Link to="/" className="results-back">← Home</Link>
      <h1>My coffees</h1>

      {isAnonymous && coffees.length > 0 && (
        <div className="anon-banner">
          You're signed in as a guest on this device.
          {' '}
          <Link to="/signin" state={{ next: '/coffees' }}>Sign in with Google</Link>
          {' '}to keep your coffees if you switch devices or clear your browser.
        </div>
      )}

      {coffees.length === 0 ? (
        <p style={{ color: 'var(--text-2)' }}>
          No saved coffees yet. Scan a bag and tap “Save to my coffees” after dialling in.
        </p>
      ) : (
        <ul className="my-coffees-list">
          {coffees.map(c => (
            <li key={c.coffee_id} className="my-coffee">
              <Link to={`/coffees/${encodeURIComponent(c.coffee_id)}`} className="my-coffee__link">
                <div className="my-coffee__title">
                  {c.bean_profile.parsed.roaster_name ?? 'Unknown roaster'}
                  {c.bean_profile.parsed.producer && (
                    <span className="my-coffee__producer"> · {c.bean_profile.parsed.producer}</span>
                  )}
                </div>
                <div className="my-coffee__meta">
                  <Stars rating={c.rating} />
                  <span>{c.session_count} sessions</span>
                  <span>{relativeTime(c.last_seen_at)}</span>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
