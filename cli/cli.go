package cli

// CLI is the command line interface of disco.
type CLI struct {
	Get Get `kong:"cmd,help='Get the value of a key.'"`
	Set Set `kong:"cmd,help='Set the value of a key.'"`
	LS  LS  `kong:"cmd,help='List all keys.'"`
}
