package types

const (
	// ServerName is used in TLS certificates for verification.
	// TODO: Use a randomized name per server.
	ServerName = "hackfix.me/disco"
	// ConnTLSUserKey is the key used to reference the Disco user extracted from
	// the client TLS certificate and stored in the HTTP request context.
	ConnTLSUserKey = "connTLSUser"
)
