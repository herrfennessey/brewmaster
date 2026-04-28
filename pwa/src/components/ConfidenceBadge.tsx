const COLORS: Record<string, string> = { high: '#2a7a2a', medium: '#7a5a00', low: '#c00' }

interface Props {
  level: string
}

export default function ConfidenceBadge({ level }: Props) {
  const style: React.CSSProperties = {
    display: 'inline-block',
    padding: '0.25rem 0.65rem',
    borderRadius: 6,
    fontSize: '0.8rem',
    fontWeight: 600,
    color: '#fff',
    background: COLORS[level] ?? '#888',
  }
  return <span style={style}>Confidence: {level}</span>
}
