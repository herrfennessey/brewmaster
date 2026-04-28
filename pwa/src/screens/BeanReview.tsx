import { useState } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { generateParametersAPI } from '../services/api'
import { getBeanById, saveBeanProfile, saveBrewParameters } from '../services/storage'
import type { ParsedBean } from '../types'
import ConfidenceBadge from '../components/ConfidenceBadge'

const s = {
  page: { maxWidth: 640, margin: '0 auto', padding: '2rem 1rem', fontFamily: 'system-ui, sans-serif' } satisfies React.CSSProperties,
  back: { color: '#555', textDecoration: 'none', fontSize: '0.9rem' } satisfies React.CSSProperties,
  heading: { margin: '1rem 0 0.5rem' } satisfies React.CSSProperties,
  confidenceNote: { color: '#555', fontSize: '0.85rem', margin: '0.35rem 0 1.5rem' } satisfies React.CSSProperties,
  grid: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', marginBottom: '1.5rem' } satisfies React.CSSProperties,
  fieldBlock: { display: 'flex', flexDirection: 'column' as const, gap: '0.3rem' } satisfies React.CSSProperties,
  label: { fontSize: '0.8rem', fontWeight: 600, color: '#555', textTransform: 'uppercase' as const, letterSpacing: '0.05em' } satisfies React.CSSProperties,
  input: { padding: '0.5rem 0.75rem', borderRadius: 6, border: '1.5px solid #ccc', fontSize: '0.95rem' } satisfies React.CSSProperties,
  errorMsg: { color: '#c00', margin: '0 0 1rem', fontSize: '0.9rem' } satisfies React.CSSProperties,
  btn: { width: '100%', padding: '0.85rem', fontSize: '1rem', fontWeight: 700, borderRadius: 8, border: 'none', background: '#1a1a1a', color: '#fff', cursor: 'pointer' } satisfies React.CSSProperties,
}

const numericFields = new Set<keyof ParsedBean>(['altitude_m', 'lot_year'])

export default function BeanReview() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const original = getBeanById(id ?? '')

  const [parsed, setParsed] = useState<ParsedBean | null>(original?.parsed ?? null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!original || !parsed) {
    return <div style={s.page}><p>Bean not found. <Link to="/">Start over</Link></p></div>
  }

  function updateField(key: keyof ParsedBean, value: string) {
    setParsed(prev => {
      if (!prev) return prev
      if (value === '') return { ...prev, [key]: null }
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
      const params = await generateParametersAPI(updated)
      saveBrewParameters(params)
      navigate(`/brew/${updated.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={s.page}>
      <Link to="/" style={s.back}>← Back</Link>
      <h2 style={s.heading}>Review Bean Info</h2>
      <ConfidenceBadge level={original.confidence.level} />
      <p style={s.confidenceNote}>{original.confidence.notes}</p>

      <div style={s.grid}>
        {fields.map(({ key, label }) => (
          <div key={key} style={s.fieldBlock}>
            <label htmlFor={key} style={s.label}>{label}</label>
            <input
              id={key}
              style={s.input}
              value={getFieldValue(parsed, key)}
              onChange={e => updateField(key, e.target.value)}
              placeholder="unknown"
            />
          </div>
        ))}
        <div style={s.fieldBlock}>
          <label htmlFor="flavor_notes" style={s.label}>Flavor notes</label>
          <input
            id="flavor_notes"
            style={s.input}
            value={parsed.flavor_notes?.join(', ') ?? ''}
            onChange={e => setParsed(p => p ? { ...p, flavor_notes: e.target.value ? e.target.value.split(',').map(t => t.trim()) : [] } : p)}
            placeholder="e.g. caramel, citrus, chocolate"
          />
        </div>
      </div>

      {error && <p style={s.errorMsg}>{error}</p>}

      <button style={s.btn} onClick={handleConfirm} disabled={loading}>
        {loading ? 'Generating parameters…' : 'Generate Brew Parameters →'}
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
  { key: 'producer', label: 'Producer' },
  { key: 'roaster_name', label: 'Roaster' },
  { key: 'origin_country', label: 'Country' },
  { key: 'origin_region', label: 'Region' },
  { key: 'altitude_m', label: 'Altitude (m)' },
  { key: 'varietal', label: 'Varietal' },
  { key: 'process', label: 'Process' },
  { key: 'roast_level', label: 'Roast level' },
  { key: 'roast_date', label: 'Roast date' },
  { key: 'lot_year', label: 'Lot year' },
]
