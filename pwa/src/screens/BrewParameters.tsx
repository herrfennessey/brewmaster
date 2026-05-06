import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { getBrewParamsForBean, getBeanById } from '../services/storage'
import { upsertCoffeeAPI } from '../services/api'
import type { DrinkSuitability, DrinkType, ParameterValue } from '../types'
import { DRINK_LABELS } from '../types'
import ConfidenceBadge from '../components/ConfidenceBadge'

function truncate(s: string, max: number) {
  return s.length > max ? s.slice(0, max).trimEnd() + '…' : s
}

function getParamLabels(method: string) {
  if (method === 'pourover') {
    return { yield: 'Water', preinfusion: 'Bloom', time: 'Total Time' }
  }
  return { yield: 'Yield', preinfusion: 'Preinfusion', time: 'Time' }
}

function SuitabilityBanner({ s }: { s: DrinkSuitability | undefined }) {
  if (!s || s.level === 'ideal' || s.level === 'suitable') return null
  return (
    <div className={`suitability-banner suitability-banner--${s.level}`}>
      <span className="suitability-banner__label">
        {s.level === 'poor' ? 'Poor pairing' : 'Not recommended'}
      </span>
      <span className="suitability-banner__reason">{s.reason}</span>
    </div>
  )
}

function SuitabilityChip({ s }: { s: DrinkSuitability | undefined }) {
  if (!s || s.level === 'suboptimal' || s.level === 'poor') return null
  return (
    <span className={`suitability-chip suitability-chip--${s.level}`} title={s.reason}>
      {s.level === 'ideal' ? 'Ideal pairing' : 'Suitable pairing'}
    </span>
  )
}

export default function BrewParameters() {
  const { beanId } = useParams<{ beanId: string }>()
  const params = getBrewParamsForBean(beanId ?? '')
  const bean = getBeanById(beanId ?? '')
  const [showReasoning, setShowReasoning] = useState(false)
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')
  const [saveMsg, setSaveMsg] = useState<string | null>(null)
  const navigate = useNavigate()
  const navTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => () => {
    if (navTimer.current) clearTimeout(navTimer.current)
  }, [])

  async function saveToMyCoffees() {
    if (!bean) return
    setSaveState('saving')
    setSaveMsg(null)
    try {
      const res = await upsertCoffeeAPI(bean)
      setSaveState('saved')
      setSaveMsg(res.is_new ? 'Saved to my coffees.' : 'Updated existing coffee.')
      navTimer.current = setTimeout(() => {
        navigate(`/coffees/${encodeURIComponent(res.canonical_key)}`)
      }, 600)
    } catch (err) {
      setSaveState('error')
      setSaveMsg(err instanceof Error ? err.message : 'Failed to save')
    }
  }

  if (!params) {
    return (
      <div className="screen results-screen">
        <p style={{ color: 'var(--text-2)' }}>No parameters found. <Link to="/">Start over</Link></p>
      </div>
    )
  }

  const p = params.parameters
  const parsed = bean?.parsed
  const labels = getParamLabels(params.extraction_method ?? 'espresso')

  const roaster   = parsed?.roaster_name   ?? null
  const region    = parsed?.origin_region  ?? null
  const country   = parsed?.origin_country ?? null
  const process   = parsed?.process        ?? null
  const roastLevel = parsed?.roast_level   ?? null
  const varietal  = parsed?.varietal       ?? null

  const locationTitle = [region, country].filter(Boolean).join(', ')
  const title = locationTitle || truncate(roaster ?? 'Brew Parameters', 36)
  const beanMeta = [varietal, process, roastLevel].filter(Boolean).join(' · ')

  const methodLabel = params.extraction_method === 'pourover' ? 'Pourover' : 'Espresso'
  const drinkLabel = params.drink_type ? (DRINK_LABELS[params.drink_type as DrinkType] ?? params.drink_type) : null

  return (
    <div className="screen results-screen">
      <Link to="/" className="results-back">← New beans</Link>

      {(methodLabel || drinkLabel) && (
        <div className="results-brew-context">
          {methodLabel}{drinkLabel ? ` · ${drinkLabel}` : ''}
        </div>
      )}

      <div className="results-bean">
        {roaster && <div className="results-roaster">{truncate(roaster, 42)}</div>}
        <div className="results-title">{title}</div>
        {beanMeta && <div className="results-meta">{beanMeta}</div>}
      </div>

      <div className="results-badge-row">
        <ConfidenceBadge level={params.confidence.level} />
        <SuitabilityChip s={params.suitability} />
      </div>
      {params.confidence.reason && (
        <div className="results-reason">{params.confidence.reason}</div>
      )}

      <SuitabilityBanner s={params.suitability} />

      <div className="param-grid">
        <ParamCell label="Dose"            param={p.dose_g}        unit="g"  />
        <ParamCell label={labels.yield}    param={p.yield_g}       unit="g"  />
        <ParamCell label="Ratio"           value={p.ratio}                    />
        <ParamCell label="Temperature"     param={p.temp_c}        unit="°C" />
        <ParamCell label={labels.time}     param={p.time_s}        unit="s"  />
        <ParamCell label={labels.preinfusion} param={p.preinfusion_s} unit="s"  />
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
        <button
          type="button"
          className="action-btn"
          disabled={!bean?.canonical_key || saveState === 'saving' || saveState === 'saved'}
          onClick={saveToMyCoffees}
          title={bean?.canonical_key ? '' : 'Need roaster + bean to save'}
        >
          {saveState === 'saving' ? 'Saving…' : saveState === 'saved' ? 'Saved' : 'Save to my coffees'}
        </button>
        <Link to="/" className="action-btn action-btn--primary">
          New analysis →
        </Link>
      </div>
      {saveMsg && (
        <p className={saveState === 'error' ? 'foot-error' : 'foot-hint'} style={{ marginTop: '0.5rem' }}>
          {saveMsg}
        </p>
      )}
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
