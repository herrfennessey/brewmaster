import { useState, useRef, useEffect, type DragEvent, type ClipboardEvent, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { parseBeanAPI, parseImageAPI, parseURLAPI, generateParametersAPI } from '../services/api'
import { saveBeanProfile, saveBrewParameters } from '../services/storage'

type Tab = 'text' | 'image' | 'url'
type Phase = 'parsing' | 'brewing'

function CupIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 32 30" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <rect x="4" y="10" width="18" height="14" rx="2.5" stroke="currentColor" strokeWidth="1.5"/>
      <path d="M22 13.5 Q29 13.5 29 18 Q29 22.5 22 22.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" fill="none"/>
      <path d="M2.5 24 Q11.5 28 24 24" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
      <path d="M10 7 Q9 4.5 11 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" opacity="0.55"/>
      <path d="M16 7 Q15 4.5 17 2" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" opacity="0.55"/>
    </svg>
  )
}

const HINTS: Record<Tab, string> = {
  text:  'More detail = better parameters',
  image: 'Vision AI reads the bag label',
  url:   'Fetches & parses the product page',
}

const PHASE_LABELS: Record<Phase, string> = {
  parsing: 'Reading your beans…',
  brewing: 'Dialling in parameters…',
}

export default function Home() {
  const [activeTab, setActiveTab] = useState<Tab>('text')
  const [content, setContent] = useState('')
  const [url, setUrl] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [previewSrc, setPreviewSrc] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [phase, setPhase] = useState<Phase | null>(null)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()

  useEffect(() => {
    return () => { if (previewSrc) URL.revokeObjectURL(previewSrc) }
  }, [previewSrc])

  function pickFile(f: File) {
    setFile(f)
    setPreviewSrc(URL.createObjectURL(f))
    setError(null)
  }

  function handleDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragOver(false)
    const f = e.dataTransfer.files[0]
    if (f) pickFile(f)
  }

  function handlePaste(e: ClipboardEvent<HTMLDivElement>) {
    const item = Array.from(e.clipboardData.items).find(i => i.type.startsWith('image/'))
    if (!item) return
    const f = item.getAsFile()
    if (f) pickFile(f)
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    setPhase('parsing')

    try {
      let bean
      if (activeTab === 'text') {
        bean = await parseBeanAPI(content.trim())
      } else if (activeTab === 'image') {
        if (!file) {
          setError('Please select a photo first')
          setPhase(null)
          return
        }
        bean = await parseImageAPI(file)
      } else {
        bean = await parseURLAPI(url.trim())
      }

      saveBeanProfile(bean)
      setPhase('brewing')

      const params = await generateParametersAPI(bean)
      saveBrewParameters(params)
      navigate(`/brew/${bean.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
      setPhase(null)
    }
  }

  const canSubmit = phase === null && (
    (activeTab === 'text' && content.trim().length > 0) ||
    (activeTab === 'image' && file !== null) ||
    (activeTab === 'url' && url.trim().length > 0)
  )

  if (phase !== null) {
    return (
      <div className="load-screen">
        <CupIcon className="load-cup" />
        <div className="load-text">
          <div className="load-phase">{PHASE_LABELS[phase]}</div>
          <div className="load-dots"><span /><span /><span /></div>
        </div>
      </div>
    )
  }

  return (
    <div className="screen home-screen">
      <div className="logo">
        <CupIcon className="logo__icon" />
        <span className="logo__name">Brewmaster</span>
      </div>

      <p className="home-tagline">Precision espresso from your coffee bag.</p>

      <form onSubmit={handleSubmit} className="input-card">
        <div className="input-card__tabs">
          {(['text', 'image', 'url'] as Tab[]).map(tab => (
            <button
              key={tab}
              type="button"
              className={`tab-btn${activeTab === tab ? ' tab-btn--active' : ''}`}
              onClick={() => { setActiveTab(tab); setError(null) }}
            >
              {tab === 'text' ? 'Text' : tab === 'image' ? 'Photo' : 'URL'}
            </button>
          ))}
        </div>

        <div className="input-card__body">
          {activeTab === 'text' && (
            <textarea
              className="bean-textarea"
              value={content}
              onChange={e => setContent(e.target.value)}
              placeholder="Paste the bag label, origin notes, or anything you know about these beans…"
              rows={7}
              autoFocus
            />
          )}

          {activeTab === 'image' && (
            <>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                style={{ display: 'none' }}
                onChange={e => { if (e.target.files?.[0]) pickFile(e.target.files[0]) }}
              />
              <div
                className={`drop-zone${dragOver ? ' drop-zone--over' : ''}`}
                onClick={() => fileInputRef.current?.click()}
                onDragOver={e => { e.preventDefault(); setDragOver(true) }}
                onDragLeave={() => setDragOver(false)}
                onDrop={handleDrop}
                onPaste={handlePaste}
                tabIndex={0}
                role="button"
                aria-label="Upload bag photo"
                onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); fileInputRef.current?.click() } }}
              >
                {previewSrc ? (
                  <img src={previewSrc} alt="Preview" className="drop-zone__preview" />
                ) : (
                  <div className="drop-zone__prompt">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                      <rect x="3" y="3" width="18" height="18" rx="3"/>
                      <circle cx="8.5" cy="8.5" r="1.5"/>
                      <polyline points="21 15 16 10 5 21"/>
                    </svg>
                    <span>Drop, paste or click to browse</span>
                    <small>JPEG · PNG · WEBP</small>
                  </div>
                )}
              </div>
            </>
          )}

          {activeTab === 'url' && (
            <input
              type="url"
              className="bean-url-input"
              value={url}
              onChange={e => setUrl(e.target.value)}
              placeholder="https://roaster.com/product/ethiopia-yirgacheffe"
              autoFocus
            />
          )}
        </div>

        <div className="input-card__foot">
          {error
            ? <span className="foot-error">{error}</span>
            : <span className="foot-hint">{HINTS[activeTab]}</span>
          }
          <button type="submit" className="parse-btn" disabled={!canSubmit}>
            Analyse →
          </button>
        </div>
      </form>

      <div className="drink-row">
        <span className="drink-label">Brew type:</span>
        <button type="button" className="drink-chip drink-chip--active">Espresso</button>
        <button type="button" className="drink-chip" disabled title="Coming soon">Filter</button>
        <button type="button" className="drink-chip" disabled title="Coming soon">Lungo</button>
      </div>
    </div>
  )
}
