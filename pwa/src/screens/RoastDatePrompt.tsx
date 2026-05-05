import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { generateParametersAPI, parseRoastDateAPI } from '../services/api'
import { getBeanById, saveBeanProfile, saveBrewParameters } from '../services/storage'
import type { DrinkType, ExtractionMethod } from '../types'

type LocationState = { method?: ExtractionMethod; drink?: DrinkType } | null

export default function RoastDatePrompt() {
  const { beanId } = useParams<{ beanId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const handoff = (location.state as LocationState) ?? null
  const method: ExtractionMethod = handoff?.method ?? 'espresso'
  const drink: DrinkType = handoff?.drink ?? (method === 'pourover' ? 'black' : 'espresso')

  const bean = getBeanById(beanId ?? '')
  const [text, setText] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<'parsing' | 'brewing' | null>(null)

  if (!bean) {
    return (
      <div className="screen roast-date-screen">
        <p style={{ color: 'var(--text-2)' }}>Bean not found. <Link to="/">Start over</Link></p>
      </div>
    )
  }

  async function brewAndGo(roastDate: string | null) {
    if (!bean) return
    const updated = roastDate
      ? { ...bean, parsed: { ...bean.parsed, roast_date: roastDate } }
      : bean
    if (roastDate) saveBeanProfile(updated)
    setBusy('brewing')
    const params = await generateParametersAPI(updated, method, drink)
    saveBrewParameters(params)
    navigate(`/brew/${updated.id}`)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!text.trim() || busy) return
    setError(null)
    setBusy('parsing')
    try {
      const { roast_date, reasoning } = await parseRoastDateAPI(text.trim())
      if (!roast_date) {
        setError(reasoning || 'Couldn\'t read a date from that — try again or skip.')
        setBusy(null)
        return
      }
      await brewAndGo(roast_date)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setBusy(null)
    }
  }

  async function handleSkip() {
    if (busy) return
    setError(null)
    try {
      await brewAndGo(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setBusy(null)
    }
  }

  if (busy === 'brewing') {
    return (
      <div className="load-screen">
        <div className="load-text">
          <div className="load-phase">Dialling in parameters…</div>
          <div className="load-dots"><span /><span /><span /></div>
        </div>
      </div>
    )
  }

  return (
    <div className="screen roast-date-screen">
      <Link to="/" className="roast-date-back">← Start over</Link>

      <div>
        <h2 className="roast-date-heading">When were these beans roasted?</h2>
        <p className="roast-date-sub">Roast date is a major factor — fresh beans need extra preinfusion to degas, older beans benefit from a small temperature bump.</p>
      </div>

      <form onSubmit={handleSubmit} className="roast-date-form">
        <input
          className="roast-date-input"
          value={text}
          onChange={e => setText(e.target.value)}
          placeholder="e.g. April 15, 2 weeks ago, expires Aug 2026"
          autoFocus
          disabled={busy !== null}
        />
        <button type="submit" className="roast-date-submit" disabled={!text.trim() || busy !== null}>
          {busy === 'parsing' ? 'Reading…' : 'Use this date'}
        </button>
      </form>

      <div className="roast-date-tip">
        <strong>Don't see a roast date?</strong> Many roasters print only a "best before" or expiration date — typically 12 months after roast. Subtract that, or check the roaster's website. You can also skip this step.
      </div>

      {error && <p className="roast-date-error">{error}</p>}

      <button
        type="button"
        className="roast-date-skip"
        onClick={handleSkip}
        disabled={busy !== null}
      >
        Skip — get general advice
      </button>
    </div>
  )
}
