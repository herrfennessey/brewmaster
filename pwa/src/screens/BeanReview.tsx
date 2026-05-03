import { useState } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { generateParametersAPI } from '../services/api'
import { getBeanById, getBrewParamsForBean, saveBeanProfile, saveBrewParameters } from '../services/storage'
import type { DrinkType, ExtractionMethod, ParsedBean } from '../types'
import ConfidenceBadge from '../components/ConfidenceBadge'

const numericFields = new Set<keyof ParsedBean>(['altitude_m'])
const integerFields = new Set<keyof ParsedBean>(['lot_year'])

export default function BeanReview() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const original = getBeanById(id ?? '')
  const existingParams = getBrewParamsForBean(id ?? '')

  const [parsed, setParsed] = useState<ParsedBean | null>(original?.parsed ?? null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!original || !parsed) {
    return (
      <div className="screen review-screen">
        <p style={{ color: 'var(--text-2)' }}>Bean not found. <Link to="/">Start over</Link></p>
      </div>
    )
  }

  function updateField(key: keyof ParsedBean, value: string) {
    setParsed(prev => {
      if (!prev) return prev
      if (value === '') return { ...prev, [key]: null }
      if (integerFields.has(key)) {
        const n = parseInt(value, 10)
        return { ...prev, [key]: Number.isFinite(n) ? n : null }
      }
      if (numericFields.has(key)) {
        const n = Number(value)
        return { ...prev, [key]: Number.isFinite(n) ? n : null }
      }
      return { ...prev, [key]: value }
    })
  }

  async function handleConfirm() {
    const updated = { ...original!, parsed: parsed! }
    saveBeanProfile(updated)
    setLoading(true)
    setError(null)
    try {
      const method = (existingParams?.extraction_method ?? 'espresso') as ExtractionMethod
      const drink = (existingParams?.drink_type ?? 'espresso') as DrinkType
      const params = await generateParametersAPI(updated, method, drink)
      saveBrewParameters(params)
      navigate(`/brew/${updated.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="screen review-screen">
      <Link to={`/brew/${original.id}`} className="review-back">← Back to parameters</Link>

      <div>
        <h2 className="review-heading">Edit Bean Details</h2>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 6 }}>
          <ConfidenceBadge level={original.confidence.level} />
          {original.source_type === 'image+web' && (
            <span className="review-enriched">enriched from roaster site</span>
          )}
        </div>
        <p className="review-note" style={{ marginTop: 8 }}>{original.confidence.notes}</p>
      </div>

      <div className="review-grid">
        {fields.map(({ key, label }) => (
          <div key={key} className="review-field">
            <label htmlFor={key}>{label}</label>
            <input
              id={key}
              value={getFieldValue(parsed, key)}
              onChange={e => updateField(key, e.target.value)}
              placeholder="unknown"
            />
          </div>
        ))}
        <div className="review-field">
          <label htmlFor="flavor_notes">Flavor notes</label>
          <input
            id="flavor_notes"
            value={parsed.flavor_notes?.join(', ') ?? ''}
            onChange={e => setParsed(p => p ? {
              ...p,
              flavor_notes: e.target.value ? e.target.value.split(',').map(t => t.trim()) : []
            } : p)}
            placeholder="e.g. caramel, citrus, chocolate"
          />
        </div>
      </div>

      {error && <p className="review-error">{error}</p>}

      <button className="review-submit" onClick={handleConfirm} disabled={loading}>
        {loading ? 'Regenerating…' : 'Regenerate Parameters →'}
      </button>
    </div>
  )
}

function getFieldValue(parsed: ParsedBean, key: keyof ParsedBean): string {
  const v = parsed[key]
  if (v === null || v === undefined) return ''
  if (Array.isArray(v)) return v.join(', ')
  return String(v)
}

const fields: { key: keyof ParsedBean; label: string }[] = [
  { key: 'producer',       label: 'Producer' },
  { key: 'roaster_name',   label: 'Roaster' },
  { key: 'origin_country', label: 'Country' },
  { key: 'origin_region',  label: 'Region' },
  { key: 'altitude_m',     label: 'Altitude (m)' },
  { key: 'varietal',       label: 'Varietal' },
  { key: 'process',        label: 'Process' },
  { key: 'roast_level',    label: 'Roast level' },
  { key: 'roast_date',     label: 'Roast date' },
  { key: 'lot_year',       label: 'Lot year' },
]
