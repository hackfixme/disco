package client

import (
	"crypto/tls"
	"net/http"
	"time"

	"go.hackfix.me/disco/crypto"
)

type Client struct {
	*http.Client
	address string
}

func New(address string, tlsConfig *tls.Config) *Client {
	if tlsConfig == nil {
		tlsConfig = crypto.DefaultTLSConfig()
	}

	return &Client{
		Client: &http.Client{
			Timeout: time.Minute,
			Transport: &http.Transport{
				DisableCompression: false,
				TLSClientConfig:    tlsConfig,
			},
		},
		address: address,
	}
}
