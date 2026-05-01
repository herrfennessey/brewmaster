import type { BeanProfile, BrewParameters } from '../types'

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error((err as { error?: string }).error ?? res.statusText)
  }
  return res.json() as Promise<T>
}

export function parseBeanAPI(content: string): Promise<BeanProfile> {
  return postJSON<BeanProfile>('/api/parse-bean', { input_type: 'text', content })
}

export function parseImageAPI(file: File): Promise<BeanProfile> {
  const form = new FormData()
  form.append('input_type', 'image')
  form.append('file', file)
  return fetch('/api/parse-bean', { method: 'POST', body: form }).then(async res => {
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error((err as { error?: string }).error ?? res.statusText)
    }
    return res.json() as Promise<BeanProfile>
  })
}

export function parseURLAPI(url: string): Promise<BeanProfile> {
  return postJSON<BeanProfile>('/api/parse-bean', { input_type: 'url', content: url })
}

export function generateParametersAPI(bean: BeanProfile, targetDrink = 'espresso'): Promise<BrewParameters> {
  return postJSON<BrewParameters>('/api/generate-parameters', {
    bean_profile: bean,
    target_drink: targetDrink,
  })
}
