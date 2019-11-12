package apitypes

// SignalBody encapsulates the payload for a call to a plugin's Signal function
type SignalBody struct {
	// Name of the signal that's to be sent
	Signal string `json:"signal"`
}
