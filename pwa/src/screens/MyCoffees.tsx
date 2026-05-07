import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { listCoffeesAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import { relativeTime } from '../services/format'
import type { CoffeeSummary } from '../types'

function Stars({ rating }: { rating?: number }) {
  if (!rating) return <span className="my-coffee__rating my-coffee__rating--empty">unrated</span>
  return (
    <span className="my-coffee__rating">
      {'★'.repeat(rating)}{'☆'.repeat(5 - rating)}
    </span>
  )
}

export default function MyCoffees() {
  const { user, ready, isAnonymous, anonError } = useAuth()
  const [coffees, setCoffees] = useState<CoffeeSummary[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Re-fetch when uid becomes known (or changes via sign-in/sign-out). Using
  // user?.uid rather than the User object means silent token refreshes don't
  // re-fire this effect every hour. In DISABLE_AUTH mode user is null but
  // ready is true, so the effect still fires once.
  const uid = user?.uid
  useEffect(() => {
    if (!ready) return
    listCoffeesAPI()
      .then(r => setCoffees(r.coffees))
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [ready, uid])

  const fatalErr = anonError ?? (error ? new Error(error) : null)

  if (!ready || coffees === null) {
    return (
      <div className="screen my-coffees-screen">
        <h1 className="my-coffees-screen__heading">My coffees</h1>
        {fatalErr
          ? <p className="coffee-section__error">{fatalErr.message}</p>
          : <p className="coffee-section__muted">Loading…</p>}
      </div>
    )
  }

  return (
    <div className="screen my-coffees-screen">
      <h1 className="my-coffees-screen__heading">My coffees</h1>

      {isAnonymous && (
        <div className="anon-banner">
          You're signed in as a guest on this device.
          {' '}
          <Link to="/signin" state={{ next: '/coffees' }}>Sign in with Google</Link>
          {' '}to keep your coffees if you switch devices or clear your browser.
        </div>
      )}

      {coffees.length === 0 ? (
        <p className="coffee-section__muted">
          No saved coffees yet. Scan a bag and tap “Save to my coffees” after dialling in.
        </p>
      ) : (
        <ul className="my-coffees-list">
          {coffees.map(c => (
            <li key={c.coffee_id} className="my-coffee">
              <Link to={`/coffees/${c.coffee_id}`} className="my-coffee__link">
                <div className="my-coffee__title">
                  {c.bean_card.roaster_name || 'Unknown roaster'}
                  {c.bean_card.producer && (
                    <span className="my-coffee__producer"> · {c.bean_card.producer}</span>
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
