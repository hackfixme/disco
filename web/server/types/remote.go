package types

type RemoteJoinResponse struct {
	*Response
	TLSCACert        string `json:"tls_ca_cert"`
	TLSClientCertEnc string `json:"tls_client_cert_enc"`
	TLSClientKeyEnc  string `json:"tls_client_key_enc"`
}
