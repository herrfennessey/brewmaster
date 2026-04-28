import { Link, useParams } from 'react-router-dom'
import { getBrewParamsForBean } from '../services/storage'
import type { ParameterValue } from '../types'
import ConfidenceBadge from '../components/ConfidenceBadge'

const s = {
  page: { maxWidth: 640, margin: '0 auto', padding: '2rem 1rem', fontFamily: 'system-ui, sans-serif' } satisfies React.CSSProperties,
  back: { color: '#555', textDecoration: 'none', fontSize: '0.9rem' } satisfies React.CSSProperties,
  heading: { margin: '1rem 0 0.25rem' } satisfies React.CSSProperties,
  reason: { color: '#555', fontSize: '0.9rem', margin: '0.5rem 0 1rem' } satisfies React.CSSProperties,

  grid: { display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem', margin: '1.25rem 0' } satisfies React.CSSProperties,
  section: { marginBottom: '1.25rem' } satisfies React.CSSProperties,
  sectionTitle: { fontSize: '0.9rem', fontWeight: 700, textTransform: 'uppercase' as const, letterSpacing: '0.05em', color: '#555', margin: '0 0 0.5rem' } satisfies React.CSSProperties,
  reasoning: { color: '#333', lineHeight: 1.6, margin: 0 } satisfies React.CSSProperties,
  flagRow: { display: 'flex', flexWrap: 'wrap' as const, gap: '0.5rem' } satisfies React.CSSProperties,
  flag: { background: '#fff3cd', color: '#7a5a00', border: '1px solid #f0d080', borderRadius: 6, padding: '0.3rem 0.75rem', fontSize: '0.85rem' } satisfies React.CSSProperties,
  disabledBtn: { width: '100%', padding: '0.85rem', fontSize: '0.95rem', borderRadius: 8, border: '1.5px solid #ccc', background: '#f5f5f5', color: '#aaa', cursor: 'not-allowed', marginTop: '0.5rem' } satisfies React.CSSProperties,
}

const card = {
  wrap: { background: '#f8f8f8', borderRadius: 12, padding: '1rem 1.25rem', display: 'flex', flexDirection: 'column' as const, gap: '0.25rem' } satisfies React.CSSProperties,
  label: { fontSize: '0.75rem', fontWeight: 600, textTransform: 'uppercase' as const, letterSpacing: '0.06em', color: '#888' } satisfies React.CSSProperties,
  value: { fontSize: '1.75rem', fontWeight: 700, color: '#1a1a1a' } satisfies React.CSSProperties,
  range: { fontSize: '0.85rem', color: '#888' } satisfies React.CSSProperties,
}

export default function BrewParameters() {
  const { beanId } = useParams<{ beanId: string }>()
  const params = getBrewParamsForBean(beanId ?? '')

  if (!params) {
    return <div style={s.page}><p>No parameters found. <Link to="/">Start over</Link></p></div>
  }

  const p = params.parameters

  return (
    <div style={s.page}>
      <Link to={`/review/${params.bean_id}`} style={s.back}>← Back to bean review</Link>
      <h2 style={s.heading}>Brew Parameters</h2>
      <ConfidenceBadge level={params.confidence.level} />
      {params.confidence.reason && <p style={s.reason}>{params.confidence.reason}</p>}

      <div style={s.grid}>
        <ParamCard label="Dose" param={p.dose_g} unit="g" />
        <ParamCard label="Yield" param={p.yield_g} unit="g" />
        <ParamCard label="Ratio" value={p.ratio} />
        <ParamCard label="Temperature" param={p.temp_c} unit="°C" />
        <ParamCard label="Time" param={p.time_s} unit="s" />
        <ParamCard label="Preinfusion" param={p.preinfusion_s} unit="s" />
      </div>

      {params.reasoning && (
        <section style={s.section}>
          <h3 style={s.sectionTitle}>Reasoning</h3>
          <p style={s.reasoning}>{params.reasoning}</p>
        </section>
      )}

      {params.flags && params.flags.length > 0 && (
        <section style={s.section}>
          <h3 style={s.sectionTitle}>Notes</h3>
          <div style={s.flagRow}>
            {params.flags.map(flag => <span key={flag} style={s.flag}>{flag}</span>)}
          </div>
        </section>
      )}

      <button style={s.disabledBtn} disabled title="Coming in Phase 3">
        Log shot feedback (coming in Phase 3)
      </button>
    </div>
  )
}

function ParamCard({ label, param, unit, value }: { label: string; param?: ParameterValue; unit?: string; value?: string }) {
  if (value !== undefined) {
    return (
      <div style={card.wrap}>
        <div style={card.label}>{label}</div>
        <div style={card.value}>{value}</div>
      </div>
    )
  }
  if (!param) return null
  return (
    <div style={card.wrap}>
      <div style={card.label}>{label}</div>
      <div style={card.value}>{param.value}{unit}</div>
      <div style={card.range}>{param.range[0]}–{param.range[1]}{unit}</div>
    </div>
  )
}
