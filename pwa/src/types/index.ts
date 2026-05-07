export interface ParsedBean {
  producer: string | null
  origin_country: string | null
  origin_region: string | null
  altitude_m: number | null
  altitude_confidence: 'exact' | 'range' | 'estimated' | null
  varietal: string | null
  process: 'natural' | 'washed' | 'honey' | 'wet-hulled' | 'anaerobic' | 'other' | null
  roast_level: 'light' | 'medium-light' | 'medium' | 'dark' | null
  roast_date: string | null
  roaster_name: string | null
  intended_use: 'filter' | 'espresso' | 'omni' | null
  flavor_notes: string[]
  lot_year: number | null
}

export type ConfidenceLevel = 'high' | 'medium' | 'low'

export interface BeanConfidence {
  level: ConfidenceLevel
  notes: string
}

export interface BeanProfile {
  id: string
  source_type: string
  parsed: ParsedBean
  confidence: BeanConfidence
  created_at: string
  canonical_key?: string
}

export interface Bag {
  bag_id: string
  opened_at: string
  finished_at?: string
  roast_date?: string
  source_type: string
}

export interface Coffee {
  canonical_key: string
  bean_profile: BeanProfile
  bags: Bag[]
  rating?: number
  notes?: string
  first_seen_at: string
  last_seen_at: string
  session_count: number
  best_session_id?: string
}

export interface BeanCard {
  roaster_name: string
  producer: string
  origin_country: string
  origin_region: string
  process: string
  roast_level: string
  varietal: string
}

export interface BagSummary {
  bag_id: string
  opened_at: string
  roast_date?: string
}

export interface CoffeeSummary {
  coffee_id: string
  bean_card: BeanCard
  rating?: number
  last_seen_at: string
  session_count: number
  bag_count: number
  open_bag?: BagSummary
}

export interface ParameterValue {
  value: number
  range: [number, number]
}

export interface BrewParams {
  dose_g: ParameterValue
  yield_g: ParameterValue
  ratio: string
  temp_c: ParameterValue
  time_s: ParameterValue
  preinfusion_s: ParameterValue
}

export interface BrewConfidence {
  level: ConfidenceLevel
  reason: string
}

export type ExtractionMethod = 'espresso' | 'pourover'

export type DrinkType =
  | 'espresso' | 'americano' | 'macchiato' | 'cortado' | 'cappuccino' | 'flat white' | 'latte'
  | 'black' | 'cafe au lait'

export type SuitabilityLevel = 'ideal' | 'suitable' | 'suboptimal' | 'poor'

export interface DrinkSuitability {
  level: SuitabilityLevel
  reason: string
}

export const DRINK_LABELS: Record<DrinkType, string> = {
  espresso: 'Espresso',
  americano: 'Americano',
  macchiato: 'Macchiato',
  cortado: 'Cortado',
  cappuccino: 'Cappuccino',
  'flat white': 'Flat White',
  latte: 'Latte',
  black: 'Black',
  'cafe au lait': 'Café au lait',
}

export interface BrewParameters {
  bean_id: string
  extraction_method: ExtractionMethod
  drink_type: DrinkType
  parameters: BrewParams
  confidence: BrewConfidence
  suitability: DrinkSuitability
  reasoning: string
  flags: string[]
  iteration: number
}
