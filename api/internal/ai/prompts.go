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
					"roast_level":         map[string]any{"type": []string{"string", "null"}, "enum": []any{"light", "medium-light", "medium", "dark", nil}},
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

// BrewAnnotatePrompt is the system prompt for the annotation-only LLM call in
// the generate-parameters endpoint. All numeric parameters, suitability, and
// confidence are already computed deterministically by the rules engine before
// this call is made.
const BrewAnnotatePrompt = `You are a specialty coffee expert. Brew parameters have already been computed by a rules engine.
Your only job: write 2–3 sentences explaining why these parameters suit this bean, and list any flags for genuinely unusual factors (rare varietal, unusual origin+process combo, very old roast date, etc).
Do not suggest different numbers.`

// BrewAnnotateTool is the tool definition for the annotation-only LLM call.
// Both fields are optional — if the call fails entirely the handler falls back
// to a deterministic reasoning string.
var BrewAnnotateTool = Tool{
	Name:        "annotate_brew",
	Description: "Provide reasoning and flags for the pre-computed brew parameters.",
	Schema: map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"reasoning": map[string]any{"type": "string", "maxLength": 400},
			"flags":     map[string]any{"type": "array", "items": map[string]any{"type": "string", "maxLength": 60}},
		},
	},
}
