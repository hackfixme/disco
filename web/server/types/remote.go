package types

type RemoteJoinResponse struct {
	*Response
	// Encrypted payload in JSON format, encoded in base58
	Data string `json:"data"`
}

type RemoteJoinResponsePayload struct {
	TLSCACert     string `json:"tls_ca_cert"`
	TLSServerSAN  string `json:"tls_server_san"`
	TLSClientCert []byte `json:"tls_client_cert"`
	TLSClientKey  []byte `json:"tls_client_key"`
}
