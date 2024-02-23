package cli

// The Set command stores the value of a key.
type Set struct {
	Key   string `arg:"" help:"The unique key that identifies the value."`
	Value string `arg:"" help:"The value."`
}

// Run the set command.
func (c *Set) Run() error {
	return nil
}
