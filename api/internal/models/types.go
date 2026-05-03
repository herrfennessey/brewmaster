// Package models defines the core data types for the brewmaster API.
package models

import "time"

// ParsedBean contains the structured coffee bean data extracted from user input.
type ParsedBean struct {
	Producer           *string  `json:"producer"`
	OriginCountry      *string  `json:"origin_country"`
	OriginRegion       *string  `json:"origin_region"`
	AltitudeM          *float64 `json:"altitude_m"`
	AltitudeConfidence *string  `json:"altitude_confidence"`
	Varietal           *string  `json:"varietal"`
	Process            *string  `json:"process"`
	RoastLevel         *string  `json:"roast_level"`
	RoastDate          *string  `json:"roast_date"`
	RoasterName        *string  `json:"roaster_name"`
	LotYear            *int     `json:"lot_year"`
	FlavorNotes        []string `json:"flavor_notes"`
}

// BeanConfidence describes how confident the AI is in its extraction.
type BeanConfidence struct {
	Level string `json:"level"`
	Notes string `json:"notes"`
}

// BeanProfile is the full output of the parse-bean endpoint.
type BeanProfile struct {
	CreatedAt  time.Time      `json:"created_at"`
	Confidence BeanConfidence `json:"confidence"`
	ID         string         `json:"id"`
	SourceType string         `json:"source_type"`
	Parsed     ParsedBean     `json:"parsed"`
}

// ParameterValue holds a recommended value and its acceptable range.
type ParameterValue struct {
	Value float64    `json:"value"`
	Range [2]float64 `json:"range"`
}

// BrewParams holds the individual espresso parameters.
type BrewParams struct {
	Ratio        string         `json:"ratio"`
	DoseG        ParameterValue `json:"dose_g"`
	YieldG       ParameterValue `json:"yield_g"`
	TempC        ParameterValue `json:"temp_c"`
	TimeS        ParameterValue `json:"time_s"`
	PreinfusionS ParameterValue `json:"preinfusion_s"`
}

// BrewConfidence describes how confident the AI is in its recommendations.
type BrewConfidence struct {
	Level  string `json:"level"`
	Reason string `json:"reason"`
}

// DrinkSuitability is the AI's assessment of how well the bean suits the requested drink.
type DrinkSuitability struct {
	Level  string `json:"level"`  // "ideal" | "suitable" | "suboptimal" | "poor"
	Reason string `json:"reason"` // one sentence
}

// BrewParameters is the full output of the generate-parameters endpoint.
type BrewParameters struct {
	Suitability      DrinkSuitability `json:"suitability"`
	Confidence       BrewConfidence   `json:"confidence"`
	BeanID           string           `json:"bean_id"`
	ExtractionMethod string           `json:"extraction_method"`
	DrinkType        string           `json:"drink_type"`
	Reasoning        string           `json:"reasoning"`
	Flags            []string         `json:"flags"`
	Parameters       BrewParams       `json:"parameters"`
	Iteration        int              `json:"iteration"`
}
