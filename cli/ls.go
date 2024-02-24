package cli

import (
	"fmt"
	"slices"
)

// The LS command prints keys.
type LS struct {
	KeyPrefix string `arg:"" optional:"" help:"An optional key prefix."`
	Namespace string `default:"default" help:"The namespace to retrieve the keys from.\n If '*' is specified, keys in all namespaces are listed. "`
}

// Run the ls command.
func (c *LS) Run(appCtx *AppContext) error {
	keysPerNS := appCtx.Store.List(c.Namespace, []byte(c.KeyPrefix))
	if len(keysPerNS) == 0 {
		return nil
	}

	if c.Namespace == "*" {
		namespaces := []string{}
		for ns := range keysPerNS {
			namespaces = append(namespaces, ns)
		}
		slices.Sort(namespaces)

		for _, ns := range namespaces {
			for _, key := range keysPerNS[ns] {
				fmt.Fprintf(appCtx.Stdout, "%s:%s\n", ns, key)
			}
		}
	} else {
		for ns := range keysPerNS {
			for _, key := range keysPerNS[ns] {
				fmt.Fprintf(appCtx.Stdout, "%s\n", key)
			}
		}
	}

	return nil
}
