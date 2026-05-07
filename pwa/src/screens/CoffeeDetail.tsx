import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  deleteCoffeeAPI, getCoffeeAPI, patchCoffeeAPI, setBagFinishedAPI,
} from '../services/api'
import { useAuth } from '../services/auth-context'
import { formatDate, metaJoin } from '../services/format'
import type { Bag, Coffee } from '../types'

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
  const navigate = useNavigate()
  const [coffee, setCoffee] = useState<Coffee | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)
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

  async function patch(p: { rating?: number; notes?: string; clear?: ('rating' | 'notes')[] }) {
    if (!id) return
    setSaving(true)
    try {
      setCoffee(await patchCoffeeAPI(id, p))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  function saveRating(n: number) {
    // Skip the round-trip when the user clicks the already-selected star.
    if ((coffee?.rating ?? 0) === n) return
    return patch(n === 0 ? { clear: ['rating'] } : { rating: n })
  }

  function saveNotes() {
    return patch(notesDraft === '' ? { clear: ['notes'] } : { notes: notesDraft })
  }

  async function handleDelete() {
    if (!id) return
    const ok = window.confirm('Delete this coffee? Every bag, rating and note will be removed.')
    if (!ok) return
    setDeleting(true)
    try {
      await deleteCoffeeAPI(id)
      navigate('/coffees')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete')
      setDeleting(false)
    }
  }

  async function setBagFinished(bagID: string, finished: boolean) {
    if (!id) return
    try {
      setCoffee(await setBagFinishedAPI(id, bagID, finished))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update bag')
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
        <div className="section-tag">Rating</div>
        <StarRating value={coffee.rating ?? 0} onChange={saveRating} />
      </section>

      <section className="coffee-section">
        <div className="section-tag">Notes</div>
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
            onClick={saveNotes}
          >
            {saving ? 'Saving…' : 'Save notes'}
          </button>
        </div>
      </section>

      {(() => {
        const open = coffee.bags.filter(b => !b.finished_at)
        const finished = coffee.bags.filter(b => b.finished_at)
        const renderBag = (b: Bag, isOpen: boolean) => (
          <li key={b.bag_id} className="bag-list__item">
            <div className="bag-list__lines">
              <span className="bag-list__roast">
                {b.roast_date ? `Roasted ${b.roast_date}` : 'Roast date unknown'}
              </span>
              <span className="bag-list__dates">
                Opened {formatDate(b.opened_at)}
                {b.finished_at && ` · finished ${formatDate(b.finished_at)}`}
              </span>
            </div>
            <button
              type="button"
              className="bag-list__action"
              onClick={() => setBagFinished(b.bag_id, isOpen)}
            >
              {isOpen ? 'Mark finished' : 'Reopen'}
            </button>
          </li>
        )
        return (
          <>
            <section className="coffee-section">
              <div className="section-tag">Open bags</div>
              {open.length === 0 ? (
                <p className="coffee-section__muted">
                  No open bag right now. Re-scan this coffee when you start a fresh bag — your ratings and notes carry over.
                </p>
              ) : (
                <ul className="bag-list">{open.map(b => renderBag(b, true))}</ul>
              )}
            </section>

            {finished.length > 0 && (
              <section className="coffee-section">
                <div className="section-tag">Finished</div>
                <ul className="bag-list bag-list--finished">{finished.map(b => renderBag(b, false))}</ul>
              </section>
            )}
          </>
        )
      })()}

      {coffee.session_count > 0 && (
        <section className="coffee-section">
          <div className="section-tag">Brew sessions</div>
          <p className="coffee-section__muted">
            {coffee.session_count} session{coffee.session_count === 1 ? '' : 's'} logged.
            Session detail comes in the next iteration.
          </p>
        </section>
      )}

      <section className="coffee-section coffee-section--danger">
        <button
          type="button"
          className="coffee-detail__delete"
          onClick={handleDelete}
          disabled={deleting}
        >
          {deleting ? 'Deleting…' : 'Delete coffee'}
        </button>
      </section>
    </div>
  )
}
