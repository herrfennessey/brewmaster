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

// ParseRoastDatePrompt is the system prompt for the parse-roast-date endpoint.
// The user types something freeform after being asked when their beans were
// roasted; this prompt converts whatever they wrote into an ISO date or null.
const ParseRoastDatePrompt = `You convert freeform text into an ISO 8601 (YYYY-MM-DD) roast date.

The user message contains today's date (use it for relative dates) followed by their input. The input may be:
- An explicit date: "April 15", "2025-04-15", "Apr 15 2025" → convert to YYYY-MM-DD.
- A relative date: "2 weeks ago", "last Friday", "yesterday" → resolve against today's date.
- An expiration / "best before" date: "expires Aug 2026", "BB Sep 2025", "best before 09/2025". Specialty roasters typically use a 12-month shelf life, so subtract 12 months from the expiration date to estimate the roast date. If the user mentions a different shelf life, use that instead.
- Ambiguous, contradictory, or non-date text → return null for roast_date.

When you must guess (month-only, expiration date), pick the 1st of the month. When you return a date, fill reasoning with one short sentence explaining what you inferred (e.g. "expiration Aug 2026 minus 12 months → Aug 2025"). When you return null, fill reasoning with a brief explanation of why.`

// ParseRoastDateTool is the tool definition forcing structured output for
// parse-roast-date. Both fields are required so the handler can rely on them
// without nil-checking the reasoning string separately.
var ParseRoastDateTool = Tool{
	Name:        "extract_roast_date",
	Description: "Extract an ISO 8601 roast date from freeform user text.",
	Schema: map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"roast_date", "reasoning"},
		"properties": map[string]any{
			"roast_date": map[string]any{"type": []string{"string", "null"}},
			"reasoning":  map[string]any{"type": "string", "maxLength": 200},
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
