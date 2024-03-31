package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/web/client"
)

// The Get command retrieves and prints the value of a key.
type Get struct {
	Key string `arg:"" help:"The unique key associated with the value."`

	Namespace string `default:"default" help:"The namespace to retrieve the value from."`
	Remote    string `help:"The remote Disco node to retrieve the value from."`
}

// Run the get command.
func (c *Get) Run(appCtx *actx.Context) error {
	if c.Namespace == "*" {
		// TODO: Think about how the wildcard namespace could work for the get
		// command. Output values for the given key in all namespaces, separated
		// by \0?
		return errors.New("namespace '*' is not supported for the get command")
	}

	var (
		value io.Reader
		ok    bool
		err   error
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
		ok, value, err = client.StoreGet(appCtx.Ctx, c.Namespace, c.Key)
		if err != nil {
			return err
		}
	} else {
		ok, value, err = appCtx.Store.Get(c.Namespace, c.Key)
		if err != nil {
			return err
		}
	}

	if ok {
		fmt.Fprintf(appCtx.Stdout, "%s", value)
	}

	return nil
}
