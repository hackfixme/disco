package cli

import (
	"fmt"
	"slices"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/web/client"
)

// The Ls command prints keys.
type Ls struct {
	KeyPrefix string `arg:"" optional:"" help:"An optional key prefix."`

	Namespace string `default:"default" help:"The namespace to retrieve the keys from.\n If '*' is specified, keys in all namespaces are listed. "`
	Remote    string `help:"The remote Disco node to retrieve key data from."`
}

// Run the ls command.
func (c *Ls) Run(appCtx *actx.Context) error {
	var (
		keysPerNS map[string][]string
		listErr   error
	)

	if c.Remote != "" {
		r := &models.Remote{Name: c.Remote}
		if err := r.Load(appCtx.DB.NewContext(), appCtx.DB); err != nil {
			return err
		}

		tlsConfig, err := r.ClientTLSConfig(appCtx.User.PrivateKey)
		if err != nil {
			return err
		}

		client := client.New(r.Address, tlsConfig)
		keysPerNS, listErr = client.StoreList(appCtx.Ctx, c.Namespace, c.KeyPrefix)
	} else {
		keysPerNS, listErr = appCtx.Store.List(c.Namespace, c.KeyPrefix)
	}

	if listErr != nil {
		return listErr
	}
	if len(keysPerNS) == 0 {
		return nil
	}

	if c.Namespace == "*" {
		namespaces := []string{}
		for ns := range keysPerNS {
			namespaces = append(namespaces, ns)
		}
		slices.Sort(namespaces)

		data := make([][]string, 0)
		for _, ns := range namespaces {
			for i, key := range keysPerNS[ns] {
				row := []string{ns, key}
				if i > 0 {
					row[0] = ""
				}
				data = append(data, row)
			}
		}

		header := []string{"Namespace", "Key"}
		newTable(header, data, appCtx.Stdout).Render()
	} else {
		for ns := range keysPerNS {
			for _, key := range keysPerNS[ns] {
				fmt.Fprintf(appCtx.Stdout, "%s\n", key)
			}
		}
	}

	return nil
}
