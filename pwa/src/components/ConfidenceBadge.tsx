import type { ConfidenceLevel } from '../types'

interface Props {
  level: ConfidenceLevel
}

export default function ConfidenceBadge({ level }: Props) {
  return (
    <span className={`confidence-badge confidence-badge--${level}`}>
      {level} confidence
    </span>
  )
}
