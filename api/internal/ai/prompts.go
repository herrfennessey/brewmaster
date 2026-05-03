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
const BrewParamPrompt = `You are a specialty coffee expert. Generate brew parameters for the requested extraction method and drink based on the bean profile provided.

## Suitability assessment
Evaluate honestly before generating parameters. The default when no rule matches is "suitable" — not "ideal". Reserve "ideal" for genuinely synergistic matches. Apply rules in order and stop at the first match.

Milk volume reference (espresso drinks): latte = high milk; cappuccino/flat white = medium milk; cortado/macchiato = low milk; espresso/americano = no milk.

POOR — any single match here overrides everything:

Varietal (non-negotiable regardless of roast, process, or milk volume):
- Gesha/Geisha + any milk drink: jasmine, bergamot, and tea-like complexity disappear in milk. Universally considered wasteful by specialty professionals. Always poor.
- Eugenioides + medium or high milk: too delicate and rare for milk.

Process:
- Anaerobic + high milk (latte): fermented funky aromatics clash with or are buried by milk. The character you paid for is gone.
- Anaerobic + medium milk (cappuccino, flat white): same problem, slightly less severe — still a waste of the fermentation character.
- Carbonic maceration + medium or high milk: wine-like fermented fruit gets muddied by milk.

Roast:
- Light roast + latte: delicate aromatics and brightness overwhelmed completely. The coffee disappears into the milk regardless of origin.
- Dark roast + pourover (black or cafe au lait): bitterness amplified without espresso pressure to tame it; heavy and harsh.

SUBOPTIMAL — apply when no poor rule matched and any of these fit:
- Natural process + latte: fruity sweetness partly survives but complex berry/stone-fruit notes that define good naturals are lost.
- Light roast + cappuccino or flat white: some character survives at medium milk but significantly diminished; acidity can also clash against milk proteins.
- Ethiopia, Kenya, Rwanda, or Burundi origin + latte: bright floral, acidic, fruity complexity defines these beans — milk buries the value proposition.
- SL28 varietal + medium or high milk: SL28's signature intense sharp acidity does not integrate with milk; tastes sour in a latte.
- Natural or anaerobic + cortado or macchiato: lower milk lets more character survive, but still loses nuance vs. drinking it straight.
- Dark roast + cafe au lait: drinkable but heavy and flat.

SUITABLE — pairing works without notable concern:
- Washed process + any milk drink (clean flavors integrate well)
- Honey process + any milk drink (balanced body and sweetness complement milk)
- Medium roast + any milk drink
- Brazil, Colombia, Guatemala, El Salvador, Honduras, or Nicaragua origin + any milk drink
- Indonesia/Sumatra (wet-hulled, Giling Basah) + any milk drink (heavy body and low acidity built for milk)
- Washed or natural + pourover black
- Any bean + americano (water dilution is forgiving; origin matters less)

IDEAL — genuinely synergistic, not just absence of problems:
- Brazil, Colombia, Guatemala, El Salvador, or Indonesia/Sumatra + medium-dark or dark roast + cappuccino, flat white, or latte: the classic milk espresso. Chocolate, caramel, and heavy body cut through milk perfectly.
- Honey process + medium roast + any milk drink: body and sweetness synergise with milk.
- Wet-hulled Sumatra + cappuccino or latte: earthy chocolate and very low acidity are made for milk.
- Light or medium-light roast + washed + pourover black or straight espresso: origin clarity and complexity shine without interference.
- Washed Ethiopian or Kenyan + straight espresso or americano: bright fruit, florals, and acidity are the point — these show what specialty coffee is.

## Espresso parameters (when extraction_method = espresso)
SCA standards apply:
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
- Roast level adjustments: light → higher temp, longer time, higher ratio; dark → lower temp, shorter time, lower ratio
- Preinfusion: 4–8s standard, lean lower for darker roasts
- Drink adjustments within espresso:
  - milk drinks (latte, cappuccino, flat white, cortado): slightly higher dose and ratio tolerated; suitability depends on whether the bean has enough body to cut through milk
  - straight espresso or macchiato: dial for clarity and sweetness, tighter ratio
  - americano: standard espresso parameters; yield will be diluted externally

## Pourover parameters (when extraction_method = pourover)
Fields are reused with pourover semantics:
- dose_g: ground coffee. Typical 15–20g for a 250–350ml cup
- yield_g: TOTAL water used (not liquid yield). Typical 225–350g
- ratio: water-to-coffee as "1:15" to "1:17". 1:15 = stronger, 1:17 = lighter
- temp_c: 88–96°C depending on roast (light → higher, dark → lower)
- time_s: total brew time including bloom. Typical 180–240s
- preinfusion_s: bloom time. Typical 30–45s using ~2× coffee weight in water
- Roast adjustments: light → 93–96°C, 1:16–1:17; medium → 90–93°C, 1:15–1:16; dark → 87–90°C, 1:14–1:15
- café au lait: generate parameters for the coffee portion; note milk ratio in reasoning

## Output format rules — follow exactly
- ratio: ONLY "1:X" or "1:X.X" (e.g. "1:2.5"). No other text. MUST equal yield_g.value ÷ dose_g.value rounded to one decimal. Choose dose and yield first, then derive ratio — never pick all three independently.
- reasoning: 2–3 sentences maximum. Justify key parameter choices concisely.
- confidence.reason: one sentence only.
- suitability.reason: one sentence only.
- flags: short labels under 60 characters. Use only for genuinely unusual factors. Omit entirely if nothing unusual applies.`

// BrewParamTool is the tool definition used to force structured output from the generate-parameters call.
var BrewParamTool = Tool{
	Name:        "generate_brew_parameters",
	Description: "Generate brew parameters for the requested extraction method and drink.",
	Schema: map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"parameters", "confidence", "suitability", "reasoning", "iteration"},
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
			"suitability": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"level", "reason"},
				"properties": map[string]any{
					"level":  map[string]any{"type": "string", "enum": []string{"ideal", "suitable", "suboptimal", "poor"}},
					"reason": map[string]any{"type": "string", "maxLength": 160, "description": "One sentence describing the bean-drink pairing"},
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
