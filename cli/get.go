package cli

// The Get command retrieves and prints the value of a key.
type Get struct {
	Key string `arg:"" help:"The unique key associated with the value."`
}
