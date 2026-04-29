import { useState, type FormEvent, type CSSProperties } from 'react'
import { useNavigate } from 'react-router-dom'
import { parseBeanAPI } from '../services/api'
import { saveBeanProfile } from '../services/storage'

export default function Home() {
  const [content, setContent] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!content.trim()) return
    setLoading(true)
    setError(null)
    try {
      const bean = await parseBeanAPI(content.trim())
      saveBeanProfile(bean)
      navigate(`/review/${bean.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={styles.page}>
      <header style={styles.header}>
        <h1 style={styles.title}>Brewmaster</h1>
        <p style={styles.subtitle}>Paste your coffee bag info to get dialled-in espresso parameters</p>
      </header>

      <form onSubmit={handleSubmit} style={styles.form}>
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

        <div style={styles.field}>
          <label style={styles.label}>Target drink</label>
          <div style={styles.drinkRow}>
            <button type="button" style={styles.drinkActive}>Espresso</button>
            <button type="button" style={styles.drinkDisabled} disabled title="Coming soon">Filter</button>
            <button type="button" style={styles.drinkDisabled} disabled title="Coming soon">Lungo</button>
          </div>
        </div>

        {error && <p style={styles.errorMsg}>{error}</p>}

        <button type="submit" style={styles.submitBtn} disabled={loading || !content.trim()}>
          {loading ? 'Parsing bean info…' : 'Parse Bean →'}
        </button>
      </form>
    </div>
  )
}

const styles = {
  page: { maxWidth: 600, margin: '0 auto', padding: '2rem 1rem', fontFamily: 'system-ui, sans-serif' } satisfies CSSProperties,
  header: { marginBottom: '2rem', textAlign: 'center' as const } satisfies CSSProperties,
  title: { fontSize: '2rem', fontWeight: 700, margin: 0 } satisfies CSSProperties,
  subtitle: { color: '#555', marginTop: '0.5rem' } satisfies CSSProperties,
  form: { display: 'flex', flexDirection: 'column' as const, gap: '1.5rem' } satisfies CSSProperties,
  field: { display: 'flex', flexDirection: 'column' as const, gap: '0.4rem' } satisfies CSSProperties,
  label: { fontWeight: 600, fontSize: '0.9rem', color: '#333' } satisfies CSSProperties,
  textarea: { padding: '0.75rem', fontSize: '1rem', borderRadius: 8, border: '1.5px solid #ccc', resize: 'vertical' as const, lineHeight: 1.5 } satisfies CSSProperties,
  drinkRow: { display: 'flex', gap: '0.75rem' } satisfies CSSProperties,
  drinkActive: { padding: '0.5rem 1.25rem', borderRadius: 8, border: '2px solid #1a1a1a', background: '#1a1a1a', color: '#fff', cursor: 'pointer', fontWeight: 600 } satisfies CSSProperties,
  drinkDisabled: { padding: '0.5rem 1.25rem', borderRadius: 8, border: '1.5px solid #ddd', background: '#f5f5f5', color: '#aaa', cursor: 'not-allowed' as const } satisfies CSSProperties,
  errorMsg: { color: '#c00', margin: 0, fontSize: '0.9rem' } satisfies CSSProperties,
  submitBtn: { padding: '0.85rem', fontSize: '1rem', fontWeight: 700, borderRadius: 8, border: 'none', background: '#1a1a1a', color: '#fff', cursor: 'pointer' } satisfies CSSProperties,
}
