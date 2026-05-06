import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getCoffeeAPI, patchCoffeeAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import type { Coffee } from '../types'

function StarRating({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <div className="rating-row">
      {[1, 2, 3, 4, 5].map(n => (
        <button
          key={n}
          type="button"
          className={`rating-star${n <= value ? ' rating-star--filled' : ''}`}
          aria-label={`Rate ${n} stars`}
          onClick={() => onChange(n)}
        >
          {n <= value ? '★' : '☆'}
        </button>
      ))}
      {value > 0 && (
        <button type="button" className="rating-clear" onClick={() => onChange(0)} aria-label="Clear rating">
          clear
        </button>
      )}
    </div>
  )
}

export default function CoffeeDetail() {
  const { id } = useParams<{ id: string }>()
  const { user, loading: authLoading } = useAuth()
  const [coffee, setCoffee] = useState<Coffee | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [notesDraft, setNotesDraft] = useState('')

  useEffect(() => {
    if (authLoading || !user || !id) return
    getCoffeeAPI(id)
      .then(c => {
        setCoffee(c)
        setNotesDraft(c.notes ?? '')
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [id, user, authLoading])

  async function applyPatch(patch: { rating?: number; notes?: string }) {
    if (!id) return
    setSaving(true)
    try {
      const updated = await patchCoffeeAPI(id, patch)
      setCoffee(updated)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  if (!coffee) {
    return (
      <div className="screen coffee-detail-screen">
        <Link to="/coffees" className="results-back">← My coffees</Link>
        {error
          ? <p style={{ color: 'var(--accent-error, #c33)' }}>{error}</p>
          : <p style={{ color: 'var(--text-2)' }}>Loading…</p>}
      </div>
    )
  }

  const parsed = coffee.bean_profile.parsed
  const subtitle = [parsed.varietal, parsed.process, parsed.roast_level].filter(Boolean).join(' · ')

  return (
    <div className="screen coffee-detail-screen">
      <Link to="/coffees" className="results-back">← My coffees</Link>

      <div className="coffee-detail__header">
        <div className="coffee-detail__roaster">{parsed.roaster_name ?? 'Unknown roaster'}</div>
        <h1 className="coffee-detail__title">
          {parsed.producer ?? [parsed.origin_region, parsed.origin_country].filter(Boolean).join(', ') ?? 'Bean'}
        </h1>
        {subtitle && <div className="coffee-detail__meta">{subtitle}</div>}
      </div>

      <section className="coffee-detail__section">
        <h2>Rating</h2>
        <StarRating
          value={coffee.rating ?? 0}
          onChange={n => applyPatch({ rating: n === 0 ? undefined : n })}
        />
      </section>

      <section className="coffee-detail__section">
        <h2>Notes</h2>
        <textarea
          rows={4}
          value={notesDraft}
          onChange={e => setNotesDraft(e.target.value)}
          placeholder="Tasting notes, dial-in observations…"
        />
        <div>
          <button
            type="button"
            className="action-btn"
            disabled={saving || notesDraft === (coffee.notes ?? '')}
            onClick={() => applyPatch({ notes: notesDraft })}
          >
            {saving ? 'Saving…' : 'Save notes'}
          </button>
        </div>
      </section>

      <section className="coffee-detail__section">
        <h2>Bags</h2>
        <ul className="bag-list">
          {coffee.bags.map(b => (
            <li key={b.bag_id}>
              {b.roast_date ? `Roasted ${b.roast_date}` : 'No roast date'}
              {' · '}opened {new Date(b.opened_at).toLocaleDateString()}
              {b.finished_at && ` · finished ${new Date(b.finished_at).toLocaleDateString()}`}
            </li>
          ))}
        </ul>
      </section>

      {coffee.session_count > 0 && (
        <section className="coffee-detail__section">
          <h2>Brew sessions</h2>
          <p style={{ color: 'var(--text-2)' }}>
            {coffee.session_count} session{coffee.session_count === 1 ? '' : 's'} logged.
            Session detail comes in the next iteration.
          </p>
        </section>
      )}
    </div>
  )
}
