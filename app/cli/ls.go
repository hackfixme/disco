package cli

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"slices"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/crypto"
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
		err       error
	)

	if c.Remote != "" {
		r := &models.Remote{Name: c.Remote}
		if err = r.Load(appCtx.DB.NewContext(), appCtx.DB); err != nil {
			return err
		}

		tlsConfig := crypto.DefaultTLSConfig()

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(r.TLSCACert))
		tlsConfig.RootCAs = caCertPool

		tlsClientCert, err := r.ClientTLSCert(appCtx.User.PrivateKey)
		if err != nil {
			return err
		}
		tlsConfig.Certificates = []tls.Certificate{*tlsClientCert}

		client := client.New(r.Address, tlsConfig)
		keysPerNS, err = client.StoreList(appCtx.Ctx, c.Namespace, c.KeyPrefix)
	} else {
		keysPerNS, err = appCtx.Store.List(c.Namespace, c.KeyPrefix)
	}

	if err != nil {
		return err
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
