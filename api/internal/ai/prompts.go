package ai

// ParseBeanPrompt is the system prompt for the parse-bean endpoint.
const ParseBeanPrompt = `You are a specialty coffee expert. Extract structured data from the provided coffee bean text.

Rules:
- Use null for any field not mentioned or clearly inferrable from the text
- flavor_notes should be an empty array if none are mentioned
- altitude_confidence: "exact" if a single number given, "range" if a range, "estimated" if inferred from region
- roast_date must be ISO8601 format (YYYY-MM-DD) or null
- Do not invent data not present in the text
- confidence.level reflects how complete and unambiguous the source text was`

// ParseBeanTool is the tool definition used to force structured output from the parse-bean call.
var ParseBeanTool = Tool{
	Name:        "extract_bean_profile",
	Description: "Extract structured coffee bean data from the provided text.",
	Schema: map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"parsed", "confidence"},
		"properties": map[string]any{
			"parsed": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required": []string{
					"producer", "origin_country", "origin_region", "altitude_m",
					"altitude_confidence", "varietal", "process", "roast_level",
					"roast_date", "roaster_name", "flavor_notes", "lot_year",
				},
				"properties": map[string]any{
					"producer":            map[string]any{"type": []string{"string", "null"}},
					"origin_country":      map[string]any{"type": []string{"string", "null"}},
					"origin_region":       map[string]any{"type": []string{"string", "null"}},
					"altitude_m":          map[string]any{"type": []string{"number", "null"}},
					"altitude_confidence": map[string]any{"type": []string{"string", "null"}, "enum": []any{"exact", "range", "estimated", nil}},
					"varietal":            map[string]any{"type": []string{"string", "null"}},
					"process":             map[string]any{"type": []string{"string", "null"}, "enum": []any{"natural", "washed", "honey", "wet-hulled", "anaerobic", "other", nil}},
					"roast_level":         map[string]any{"type": []string{"string", "null"}, "enum": []any{"light", "medium", "dark", nil}},
					"roast_date":          map[string]any{"type": []string{"string", "null"}},
					"roaster_name":        map[string]any{"type": []string{"string", "null"}},
					"flavor_notes":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"lot_year":            map[string]any{"type": []string{"integer", "null"}},
				},
			},
			"confidence": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"level", "notes"},
				"properties": map[string]any{
					"level": map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}},
					"notes": map[string]any{"type": "string"},
				},
			},
		},
	},
}

// BrewParamPrompt is the system prompt for the generate-parameters endpoint.
const BrewParamPrompt = `You are a specialty coffee expert with deep knowledge of SCA espresso standards.
Generate espresso brew parameters for the provided bean profile.

Espresso standards and logic to apply:
- Target EY 18–22%, TDS 8–12%
- Standard dose: 18–20g (double basket). Yield typically 36–48g
- Process → extraction tendency:
  - natural: needs less extraction (lower temp, shorter time, or lower ratio)
  - washed: standard to more extraction
  - honey: between natural and washed
  - anaerobic: reduce aggressiveness, can be volatile
  - wet-hulled: often heavier body, lower temp can help clarity
- Altitude → density → brew temperature:
  - below 1000m: 88–90°C
  - 1000–1400m: 90–92°C
  - 1400–1800m: 92–94°C
  - above 1800m: 94–96°C
  - If altitude unknown: default to 92–93°C
- Roast level adjustments:
  - light: higher temp, longer time, higher ratio
  - medium: standard
  - dark: lower temp, shorter time, lower ratio
- Widen confidence ranges proportionally to missing fields
- Add a flag for any unusual varietals (Gesha, Eugenioides, Robusta, etc.)
- Preinfusion: 4–8s standard, lean toward lower for darker roasts

Output format rules — follow exactly:
- ratio: the string must be ONLY "1:X" or "1:X.X" (e.g. "1:2.5"). No other text, no parentheses, no explanation.
- reasoning: 2–3 sentences maximum. Justify the key parameter choices concisely.
- confidence.reason: one sentence only.
- flags: each flag is a short label under 60 characters. Use flags only for genuinely unusual factors. Omit flags entirely if nothing unusual applies.`

// BrewParamTool is the tool definition used to force structured output from the generate-parameters call.
var BrewParamTool = Tool{
	Name:        "generate_brew_parameters",
	Description: "Generate espresso brew parameters for the provided bean profile.",
	Schema: map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"parameters", "confidence", "reasoning", "iteration"},
		"$defs": map[string]any{
			"param_value": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"value", "range"},
				"properties": map[string]any{
					"value": map[string]any{"type": "number"},
					"range": map[string]any{
						"type":     "array",
						"items":    map[string]any{"type": "number"},
						"minItems": 2,
						"maxItems": 2,
					},
				},
			},
		},
		"properties": map[string]any{
			"parameters": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"dose_g", "yield_g", "ratio", "temp_c", "time_s", "preinfusion_s"},
				"properties": map[string]any{
					"dose_g":        map[string]any{"$ref": "#/$defs/param_value"},
					"yield_g":       map[string]any{"$ref": "#/$defs/param_value"},
					"ratio":         map[string]any{"type": "string", "pattern": `^1:\d+(\.\d+)?$`, "description": "Brew ratio as '1:X' or '1:X.X' only — no other text"},
					"temp_c":        map[string]any{"$ref": "#/$defs/param_value"},
					"time_s":        map[string]any{"$ref": "#/$defs/param_value"},
					"preinfusion_s": map[string]any{"$ref": "#/$defs/param_value"},
				},
			},
			"confidence": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"level", "reason"},
				"properties": map[string]any{
					"level":  map[string]any{"type": "string", "enum": []string{"high", "medium", "low"}},
					"reason": map[string]any{"type": "string", "maxLength": 160, "description": "One sentence summarizing confidence level"},
				},
			},
			"reasoning": map[string]any{"type": "string", "maxLength": 400, "description": "2–3 sentences justifying the key parameter choices"},
			"flags":     map[string]any{"type": "array", "items": map[string]any{"type": "string", "maxLength": 60}, "description": "Short labels for unusual factors only"},
			"iteration": map[string]any{"type": "integer"},
		},
	},
}
