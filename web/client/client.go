package client

import (
	"crypto/tls"
	"net/http"
	"time"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/web/server/types"
)

type Client struct {
	*http.Client
	address string
}

func New(address string, tlsConfig *tls.Config) *Client {
	if tlsConfig == nil {
		tlsConfig = crypto.DefaultTLSConfig()
	}

	tlsConfig.ServerName = types.ServerName

	return &Client{
		Client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DisableCompression: false,
				TLSClientConfig:    tlsConfig,
			},
		},
		address: address,
	}
}
