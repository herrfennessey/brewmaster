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
  const idx = all.findIndex(p => p.bean_id === params.bean_id && p.iteration === params.iteration)
  if (idx >= 0) {
    all[idx] = params
  } else {
    all.unshift(params)
  }
  localStorage.setItem(BREW_PARAMS_KEY, JSON.stringify(all))
}

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
