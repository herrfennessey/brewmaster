import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getBrewParamsForBean, getBeanById } from '../services/storage'
import type { ParameterValue } from '../types'
import ConfidenceBadge from '../components/ConfidenceBadge'

function truncate(s: string, max: number) {
  return s.length > max ? s.slice(0, max).trimEnd() + '…' : s
}

export default function BrewParameters() {
  const { beanId } = useParams<{ beanId: string }>()
  const params = getBrewParamsForBean(beanId ?? '')
  const bean = getBeanById(beanId ?? '')
  const [showReasoning, setShowReasoning] = useState(false)

  if (!params) {
    return (
      <div className="screen results-screen">
        <p style={{ color: 'var(--text-2)' }}>No parameters found. <Link to="/">Start over</Link></p>
      </div>
    )
  }

  const p = params.parameters
  const parsed = bean?.parsed

  const roaster   = parsed?.roaster_name   ?? null
  const region    = parsed?.origin_region  ?? null
  const country   = parsed?.origin_country ?? null
  const process   = parsed?.process        ?? null
  const roastLevel = parsed?.roast_level   ?? null
  const varietal  = parsed?.varietal       ?? null

  // Origin makes a better page title than a company's legal name
  const locationTitle = [region, country].filter(Boolean).join(', ')
  const title = locationTitle || truncate(roaster ?? 'Brew Parameters', 36)
  const beanMeta = [varietal, process, roastLevel].filter(Boolean).join(' · ')

  return (
    <div className="screen results-screen">
      <Link to="/" className="results-back">← New beans</Link>

      <div className="results-bean">
        {roaster && <div className="results-roaster">{truncate(roaster, 42)}</div>}
        <div className="results-title">{title}</div>
        {beanMeta && <div className="results-meta">{beanMeta}</div>}
      </div>

      <div className="results-badge-row">
        <ConfidenceBadge level={params.confidence.level} />
      </div>

      <div className="param-grid">
        <ParamCell label="Dose"        param={p.dose_g}        unit="g"  />
        <ParamCell label="Yield"       param={p.yield_g}       unit="g"  />
        <ParamCell label="Ratio"       value={p.ratio}                    />
        <ParamCell label="Temperature" param={p.temp_c}        unit="°C" />
        <ParamCell label="Time"        param={p.time_s}        unit="s"  />
        <ParamCell label="Preinfusion" param={p.preinfusion_s} unit="s"  />
      </div>

      {params.flags && params.flags.length > 0 && (
        <section className="results-section">
          <div className="results-section-title">Notes</div>
          <div className="flag-row">
            {params.flags.map(flag => <span key={flag} className="flag-chip">{flag}</span>)}
          </div>
        </section>
      )}

      {params.reasoning && (
        <section className="results-section">
          <p className={`results-reasoning${showReasoning ? '' : ' results-reasoning--collapsed'}`}>
            {params.reasoning}
          </p>
          <button className="reasoning-toggle" onClick={() => setShowReasoning(v => !v)}>
            {showReasoning ? 'Show less' : 'Show reasoning'}
          </button>
        </section>
      )}

      <hr className="results-divider" />

      <div className="results-actions">
        <Link to={`/review/${params.bean_id}`} className="action-btn">
          Edit bean details
        </Link>
        <Link to="/" className="action-btn action-btn--primary">
          New analysis →
        </Link>
      </div>
    </div>
  )
}

function ParamCell({ label, param, unit, value }: { label: string; param?: ParameterValue; unit?: string; value?: string }) {
  if (value !== undefined) {
    return (
      <div className="param-cell">
        <div className="param-cell__label">{label}</div>
        <div className="param-cell__value">{value}</div>
      </div>
    )
  }
  if (!param) return null
  return (
    <div className="param-cell">
      <div className="param-cell__label">{label}</div>
      <div className="param-cell__value">
        {param.value}
        {unit && <span className="param-cell__unit">{unit}</span>}
      </div>
      <div className="param-cell__range">{param.range[0]}–{param.range[1]}{unit}</div>
    </div>
  )
}
