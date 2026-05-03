import type { BeanProfile, BrewParameters } from '../types'

const BEANS_KEY = 'brewmaster:beans'
const BREW_PARAMS_KEY = 'brewmaster:brew_params'

export function saveBeanProfile(bean: BeanProfile): void {
  const beans = getBeans()
  const idx = beans.findIndex(b => b.id === bean.id)
  if (idx >= 0) {
    beans[idx] = bean
  } else {
    beans.unshift(bean)
  }
  localStorage.setItem(BEANS_KEY, JSON.stringify(beans))
}

export function getBeans(): BeanProfile[] {
  return parseStored<BeanProfile[]>(BEANS_KEY) ?? []
}

export function getBeanById(id: string): BeanProfile | null {
  return getBeans().find(b => b.id === id) ?? null
}

export function saveBrewParameters(params: BrewParameters): void {
  const all = getAllBrewParams()
  // Key by variant (method + drink) so generating pourover for the same bean
  // doesn't clobber an earlier espresso analysis.
  const idx = all.findIndex(p =>
    p.bean_id === params.bean_id &&
    p.extraction_method === params.extraction_method &&
    p.drink_type === params.drink_type &&
    p.iteration === params.iteration
  )
  if (idx >= 0) {
    all[idx] = params
  } else {
    all.unshift(params)
  }
  localStorage.setItem(BREW_PARAMS_KEY, JSON.stringify(all))
}

// getBrewParamsForBean returns the most recently saved variant for the bean.
// Variants are stored newest-first via unshift, so `find` yields the latest one.
export function getBrewParamsForBean(beanId: string): BrewParameters | null {
  return getAllBrewParams().find(p => p.bean_id === beanId) ?? null
}

function getAllBrewParams(): BrewParameters[] {
  return parseStored<BrewParameters[]>(BREW_PARAMS_KEY) ?? []
}

function parseStored<T>(key: string): T | null {
  const raw = localStorage.getItem(key)
  if (!raw) return null
  try {
    return JSON.parse(raw) as T
  } catch {
    localStorage.removeItem(key)
    return null
  }
}
