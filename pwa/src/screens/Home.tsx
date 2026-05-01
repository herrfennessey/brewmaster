import { useState, useRef, type CSSProperties, type DragEvent, type ClipboardEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { parseBeanAPI, parseImageAPI, parseURLAPI } from '../services/api'
import { saveBeanProfile } from '../services/storage'

type Tab = 'text' | 'image' | 'url'

export default function Home() {
  const [activeTab, setActiveTab] = useState<Tab>('text')
  const [content, setContent] = useState('')
  const [url, setUrl] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [previewSrc, setPreviewSrc] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()

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

  async function handleSubmit() {
    setLoading(true)
    setError(null)
    try {
      let bean
      if (activeTab === 'text') {
        bean = await parseBeanAPI(content.trim())
      } else if (activeTab === 'image') {
        bean = await parseImageAPI(file!)
      } else {
        bean = await parseURLAPI(url.trim())
      }
      saveBeanProfile(bean)
      navigate(`/review/${bean.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  const canSubmit = !loading && (
    (activeTab === 'text' && content.trim().length > 0) ||
    (activeTab === 'image' && file !== null) ||
    (activeTab === 'url' && url.trim().length > 0)
  )

  return (
    <div style={styles.page}>
      <header style={styles.header}>
        <h1 style={styles.title}>Brewmaster</h1>
        <p style={styles.subtitle}>Get dialled-in espresso parameters from your coffee bag info</p>
      </header>

      <div style={styles.form}>
        <div style={styles.tabBar}>
          {(['text', 'image', 'url'] as Tab[]).map(tab => (
            <button
              key={tab}
              type="button"
              onClick={() => { setActiveTab(tab); setError(null) }}
              style={activeTab === tab ? styles.tabActive : styles.tab}
            >
              {tab === 'text' ? 'Text' : tab === 'image' ? 'Image' : 'URL'}
            </button>
          ))}
        </div>

        {activeTab === 'text' && (
          <div style={styles.field}>
            <label style={styles.label} htmlFor="bean-input">Coffee bean info</label>
            <textarea
              id="bean-input"
              style={styles.textarea}
              value={content}
              onChange={e => setContent(e.target.value)}
              placeholder="Paste anything: bag label text, tasting notes, origin info..."
              rows={8}
              disabled={loading}
            />
          </div>
        )}

        {activeTab === 'image' && (
          <div style={styles.field}>
            <label style={styles.label}>Bag label photo</label>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/webp"
              style={{ display: 'none' }}
              onChange={e => { if (e.target.files?.[0]) pickFile(e.target.files[0]) }}
            />
            <div
              style={{ ...styles.dropZone, ...(dragOver ? styles.dropZoneOver : {}) }}
              onClick={() => fileInputRef.current?.click()}
              onDragOver={e => { e.preventDefault(); setDragOver(true) }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleDrop}
              onPaste={handlePaste}
              tabIndex={0}
            >
              {previewSrc
                ? <img src={previewSrc} alt="Preview" style={styles.preview} />
                : <span style={styles.dropHint}>Drop, paste, or click to browse<br /><small>JPEG, PNG, WEBP</small></span>
              }
            </div>
          </div>
        )}

        {activeTab === 'url' && (
          <div style={styles.field}>
            <label style={styles.label} htmlFor="url-input">Roaster product page URL</label>
            <input
              id="url-input"
              type="url"
              style={styles.urlInput}
              value={url}
              onChange={e => setUrl(e.target.value)}
              placeholder="https://example.com/product/kenya-aa"
              disabled={loading}
            />
          </div>
        )}

        <div style={styles.field}>
          <label style={styles.label}>Target drink</label>
          <div style={styles.drinkRow}>
            <button type="button" style={styles.drinkActive}>Espresso</button>
            <button type="button" style={styles.drinkDisabled} disabled title="Coming soon">Filter</button>
            <button type="button" style={styles.drinkDisabled} disabled title="Coming soon">Lungo</button>
          </div>
        </div>

        {error && <p style={styles.errorMsg}>{error}</p>}

        <button
          type="button"
          style={{ ...styles.submitBtn, ...(canSubmit ? {} : styles.submitBtnDisabled) }}
          disabled={!canSubmit}
          onClick={handleSubmit}
        >
          {loading
            ? (activeTab === 'url' ? 'Fetching page…' : 'Parsing bean info…')
            : 'Parse Bean →'}
        </button>
      </div>
    </div>
  )
}

const styles = {
  page: { maxWidth: 600, margin: '0 auto', padding: '2rem 1rem', fontFamily: 'system-ui, sans-serif' } satisfies CSSProperties,
  header: { marginBottom: '2rem', textAlign: 'center' as const } satisfies CSSProperties,
  title: { fontSize: '2rem', fontWeight: 700, margin: 0 } satisfies CSSProperties,
  subtitle: { color: '#555', marginTop: '0.5rem' } satisfies CSSProperties,
  form: { display: 'flex', flexDirection: 'column' as const, gap: '1.5rem' } satisfies CSSProperties,
  tabBar: { display: 'flex', gap: '0.5rem' } satisfies CSSProperties,
  tab: { padding: '0.4rem 1rem', borderRadius: 6, border: '1.5px solid #ccc', background: '#f5f5f5', color: '#555', cursor: 'pointer', fontWeight: 500 } satisfies CSSProperties,
  tabActive: { padding: '0.4rem 1rem', borderRadius: 6, border: '2px solid #1a1a1a', background: '#1a1a1a', color: '#fff', cursor: 'pointer', fontWeight: 600 } satisfies CSSProperties,
  field: { display: 'flex', flexDirection: 'column' as const, gap: '0.4rem' } satisfies CSSProperties,
  label: { fontWeight: 600, fontSize: '0.9rem', color: '#333' } satisfies CSSProperties,
  textarea: { padding: '0.75rem', fontSize: '1rem', borderRadius: 8, border: '1.5px solid #ccc', resize: 'vertical' as const, lineHeight: 1.5 } satisfies CSSProperties,
  urlInput: { padding: '0.75rem', fontSize: '1rem', borderRadius: 8, border: '1.5px solid #ccc' } satisfies CSSProperties,
  dropZone: { padding: '2rem', borderRadius: 8, border: '2px dashed #ccc', background: '#fafafa', cursor: 'pointer', textAlign: 'center' as const, minHeight: 140, display: 'flex', alignItems: 'center', justifyContent: 'center' } satisfies CSSProperties,
  dropZoneOver: { borderColor: '#1a1a1a', background: '#f0f0f0' } satisfies CSSProperties,
  dropHint: { color: '#888', lineHeight: 1.8 } satisfies CSSProperties,
  preview: { maxHeight: 200, maxWidth: '100%', borderRadius: 6, objectFit: 'contain' as const } satisfies CSSProperties,
  drinkRow: { display: 'flex', gap: '0.75rem' } satisfies CSSProperties,
  drinkActive: { padding: '0.5rem 1.25rem', borderRadius: 8, border: '2px solid #1a1a1a', background: '#1a1a1a', color: '#fff', cursor: 'pointer', fontWeight: 600 } satisfies CSSProperties,
  drinkDisabled: { padding: '0.5rem 1.25rem', borderRadius: 8, border: '1.5px solid #ddd', background: '#f5f5f5', color: '#aaa', cursor: 'not-allowed' as const } satisfies CSSProperties,
  errorMsg: { color: '#c00', margin: 0, fontSize: '0.9rem' } satisfies CSSProperties,
  submitBtn: { padding: '0.85rem', fontSize: '1rem', fontWeight: 700, borderRadius: 8, border: 'none', background: '#1a1a1a', color: '#fff', cursor: 'pointer' } satisfies CSSProperties,
  submitBtnDisabled: { background: '#999', cursor: 'not-allowed' as const } satisfies CSSProperties,
}
