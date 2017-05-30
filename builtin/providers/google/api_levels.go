package google

type ApiLevel uint8

const (
	// PRODUCTION is the v1 API.
	PRODUCTION ApiLevel = iota

	// BETA is the v0.beta API.
	BETA
)
