import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getCoffeeAPI, patchCoffeeAPI } from '../services/api'
import { useAuth } from '../services/auth-context'
import type { Coffee } from '../types'

// metaJoin filters anything not displayable. The AI parser sometimes produces
// the literal string "null" or "undefined" for missing fields; filter(Boolean)
// alone lets those through, so we strip them explicitly.
function metaJoin(parts: (string | null | undefined)[], sep = ' · '): string {
  return parts
    .filter((p): p is string => Boolean(p) && p !== 'null' && p !== 'undefined')
    .join(sep)
}

function formatDate(iso: string): string {
  // Anything we display dates from is either YYYY-MM-DD (roast date) or a full
  // ISO timestamp (opened_at). Coerce both to YYYY-MM-DD so the screen reads
  // consistently regardless of locale.
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const y = d.getUTCFullYear()
  const m = String(d.getUTCMonth() + 1).padStart(2, '0')
  const day = String(d.getUTCDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

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
  const { user, ready } = useAuth()
  const [coffee, setCoffee] = useState<Coffee | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [notesDraft, setNotesDraft] = useState('')

  const uid = user?.uid
  useEffect(() => {
    if (!ready || !id) return
    getCoffeeAPI(id)
      .then(c => {
        setCoffee(c)
        setNotesDraft(c.notes ?? '')
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load'))
  }, [id, ready, uid])

  async function applyPatch(patch: { rating?: number; notes?: string; clear?: ('rating' | 'notes')[] }) {
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
          ? <p className="coffee-section__error">{error}</p>
          : <p className="coffee-section__muted">Loading…</p>}
      </div>
    )
  }

  const parsed = coffee.bean_profile.parsed
  const title = parsed.producer
    || metaJoin([parsed.origin_region, parsed.origin_country], ', ')
    || 'Bean'
  const subtitle = metaJoin([parsed.varietal, parsed.process, parsed.roast_level])
  const notesDirty = notesDraft !== (coffee.notes ?? '')

  return (
    <div className="screen coffee-detail-screen">
      <Link to="/coffees" className="results-back">← My coffees</Link>

      <div className="results-bean">
        {parsed.roaster_name && (
          <div className="results-roaster">{parsed.roaster_name}</div>
        )}
        <div className="results-title coffee-detail__title">{title}</div>
        {subtitle && <div className="results-meta">{subtitle}</div>}
      </div>

      <section className="coffee-section">
        <div className="coffee-section__label">Rating</div>
        <StarRating
          value={coffee.rating ?? 0}
          onChange={n => applyPatch(n === 0 ? { clear: ['rating'] } : { rating: n })}
        />
      </section>

      <section className="coffee-section">
        <div className="coffee-section__label">Notes</div>
        <textarea
          className="notes-textarea"
          rows={4}
          value={notesDraft}
          onChange={e => setNotesDraft(e.target.value)}
          placeholder="Tasting notes, dial-in observations…"
        />
        <div className="coffee-section__actions">
          <button
            type="button"
            className="action-btn"
            disabled={saving || !notesDirty}
            onClick={() => applyPatch(notesDraft === '' ? { clear: ['notes'] } : { notes: notesDraft })}
          >
            {saving ? 'Saving…' : notesDirty ? 'Save notes' : 'Saved'}
          </button>
        </div>
      </section>

      <section className="coffee-section">
        <div className="coffee-section__label">Bags</div>
        <ul className="bag-list">
          {coffee.bags.map(b => {
            const opened = formatDate(b.opened_at)
            const finished = b.finished_at ? formatDate(b.finished_at) : null
            const roast = b.roast_date ?? null
            return (
              <li key={b.bag_id} className="bag-list__item">
                <span className="bag-list__roast">
                  {roast ? `Roasted ${roast}` : 'Roast date unknown'}
                </span>
                <span className="bag-list__dates">
                  Opened {opened}{finished && ` · finished ${finished}`}
                </span>
              </li>
            )
          })}
        </ul>
      </section>

      {coffee.session_count > 0 && (
        <section className="coffee-section">
          <div className="coffee-section__label">Brew sessions</div>
          <p className="coffee-section__muted">
            {coffee.session_count} session{coffee.session_count === 1 ? '' : 's'} logged.
            Session detail comes in the next iteration.
          </p>
        </section>
      )}
    </div>
  )
}
