import type {
  BeanProfile, BrewParameters, Coffee, CoffeeSummary, DrinkType, ExtractionMethod,
} from '../types'

// Token getter is wired up by App once Firebase is initialized. Decoupling it
// from this module keeps api.ts independent of React.
let getIdToken: () => Promise<string | null> = async () => null

export function setIdTokenGetter(fn: () => Promise<string | null>) {
  getIdToken = fn
}

// Called when the server rejects an authed request with 401. The auth layer
// installs a recovery handler that signs out the cached (likely revoked)
// Firebase user and mints a fresh anonymous one so the app self-heals.
let onAuthFailure: () => Promise<void> = async () => {}

export function setAuthFailureHandler(fn: () => Promise<void>) {
  onAuthFailure = fn
}

async function authedHeaders(extra?: Record<string, string>): Promise<Record<string, string>> {
  const token = await getIdToken()
  const headers: Record<string, string> = { ...(extra ?? {}) }
  if (token) headers.Authorization = `Bearer ${token}`
  return headers
}

async function unwrap<T>(res: Response): Promise<T> {
  if (res.status === 401) {
    await onAuthFailure()
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((err as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

async function postJSON<T>(path: string, body: unknown, authed = false): Promise<T> {
  const headers = authed
    ? await authedHeaders({ 'Content-Type': 'application/json' })
    : { 'Content-Type': 'application/json' }
  const res = await fetch(path, { method: 'POST', headers, body: JSON.stringify(body) })
  return unwrap<T>(res)
}

async function patchJSON<T>(path: string, body: unknown): Promise<T> {
  const headers = await authedHeaders({ 'Content-Type': 'application/json' })
  const res = await fetch(path, { method: 'PATCH', headers, body: JSON.stringify(body) })
  return unwrap<T>(res)
}

async function getJSON<T>(path: string, authed = false): Promise<T> {
  const headers = authed ? await authedHeaders() : undefined
  const res = await fetch(path, { headers })
  return unwrap<T>(res)
}

export function parseBeanAPI(content: string): Promise<BeanProfile> {
  return postJSON<BeanProfile>('/api/parse-bean', { input_type: 'text', content })
}

export function parseImageAPI(file: File): Promise<BeanProfile> {
  const form = new FormData()
  form.append('input_type', 'image')
  form.append('file', file)
  return fetch('/api/parse-bean', { method: 'POST', body: form }).then(res => unwrap<BeanProfile>(res))
}

export function parseURLAPI(url: string): Promise<BeanProfile> {
  return postJSON<BeanProfile>('/api/parse-bean', { input_type: 'url', content: url })
}

export interface ParseRoastDateResult {
  roast_date: string | null
  reasoning: string
}

export function parseRoastDateAPI(text: string): Promise<ParseRoastDateResult> {
  return postJSON<ParseRoastDateResult>('/api/parse-roast-date', { text })
}

export function generateParametersAPI(
  bean: BeanProfile,
  extractionMethod: ExtractionMethod = 'espresso',
  drinkType?: DrinkType,
): Promise<BrewParameters> {
  const drink: DrinkType = drinkType ?? (extractionMethod === 'pourover' ? 'black' : 'espresso')
  return postJSON<BrewParameters>('/api/generate-parameters', {
    bean_profile: bean,
    extraction_method: extractionMethod,
    drink_type: drink,
  })
}

// =============================================================================
// User-scoped coffees (require auth)
// =============================================================================

export interface UpsertCoffeeResponse {
  canonical_key: string
  is_new: boolean
  coffee: Coffee
}

export function upsertCoffeeAPI(
  bean: BeanProfile,
  opts?: { rating?: number; notes?: string; roastDate?: string },
): Promise<UpsertCoffeeResponse> {
  return postJSON<UpsertCoffeeResponse>(
    '/api/coffees/upsert',
    {
      bean_profile: bean,
      bag: { roast_date: opts?.roastDate ?? bean.parsed.roast_date ?? '' },
      rating: opts?.rating,
      notes: opts?.notes,
    },
    true,
  )
}

export function listCoffeesAPI(): Promise<{ coffees: CoffeeSummary[] }> {
  return getJSON<{ coffees: CoffeeSummary[] }>('/api/coffees', true)
}

export function getCoffeeAPI(id: string): Promise<Coffee> {
  return getJSON<Coffee>(`/api/coffees/${encodeURIComponent(id)}`, true)
}

export function patchCoffeeAPI(
  id: string,
  patch: { rating?: number; notes?: string; clear?: ('rating' | 'notes')[] },
): Promise<Coffee> {
  return patchJSON<Coffee>(`/api/coffees/${encodeURIComponent(id)}`, patch)
}

export function lookupCoffeeAPI(canonicalKey: string): Promise<{ coffee: CoffeeSummary | null }> {
  return getJSON<{ coffee: CoffeeSummary | null }>(
    `/api/coffees/lookup?key=${encodeURIComponent(canonicalKey)}`,
    true,
  )
}
