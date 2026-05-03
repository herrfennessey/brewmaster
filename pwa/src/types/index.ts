export interface ParsedBean {
  producer: string | null
  origin_country: string | null
  origin_region: string | null
  altitude_m: number | null
  altitude_confidence: 'exact' | 'range' | 'estimated' | null
  varietal: string | null
  process: 'natural' | 'washed' | 'honey' | 'wet-hulled' | 'anaerobic' | 'other' | null
  roast_level: 'light' | 'medium' | 'dark' | null
  roast_date: string | null
  roaster_name: string | null
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
