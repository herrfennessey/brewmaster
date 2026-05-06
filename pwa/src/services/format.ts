// Display-formatting helpers shared by every screen. Putting them here keeps
// the rendering rules consistent — e.g. the "null"/"undefined" filter for
// the meta line, which the LLM parser occasionally produces.

const placeholders = new Set(['null', 'undefined', ''])

export function metaJoin(parts: (string | null | undefined)[], sep = ' · '): string {
  return parts
    .filter((p): p is string => typeof p === 'string' && !placeholders.has(p))
    .join(sep)
}

export function formatDate(iso: string): string {
  // Anything we display dates from is either YYYY-MM-DD (roast date) or a full
  // ISO timestamp (opened_at). Coerce both to YYYY-MM-DD so the screen reads
  // consistently regardless of locale.
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const y = d.getUTCFullYear()
  const m = String(d.getUTCMonth() + 1).padStart(2, '0')
  const day = String(d.getUTCDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

export function relativeTime(iso: string): string {
  const ms = Date.now() - new Date(iso).getTime()
  const days = Math.floor(ms / (1000 * 60 * 60 * 24))
  if (days <= 0) return 'today'
  if (days === 1) return 'yesterday'
  if (days < 7) return `${days}d ago`
  if (days < 30) return `${Math.floor(days / 7)}w ago`
  return `${Math.floor(days / 30)}mo ago`
}
