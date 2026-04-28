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

export interface BeanConfidence {
  level: 'high' | 'medium' | 'low'
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
  level: 'high' | 'medium' | 'low'
  reason: string
}

export interface BrewParameters {
  bean_id: string
  parameters: BrewParams
  confidence: BrewConfidence
  reasoning: string
  flags: string[]
  iteration: number
}
