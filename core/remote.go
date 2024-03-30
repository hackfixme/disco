package core

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/mr-tron/base58"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/web/client"
)

// RemoteAuth attempts to connect to a remote Disco server, and authenticate
// with the given invitation token. The token is a concatentation of 32 bytes of
// random data and the public X25519 key of the remote node, as generated by the
// `invite user` command, and transmitted out-of-band by the user to the client
// node. If the authentication is successful, it returns the TLS client
// certificate generated by the server.
// See the inline comments for details about the process.
func RemoteAuth(ctx context.Context, address, token string) ([]byte, error) {
	// 1. Extract the random token data, and the remote X25519 public key from
	// the composite token.
	tokenData, remotePubKeyData, err := decodeToken(token)
	if err != nil {
		return nil, err
	}

	// 2. Generate an ephemeral X25519 key pair, and perform ECDH key exchange
	// in order to generate a shared secret key.
	sharedKey, pubKeyData, err := crypto.ECDHExchange(remotePubKeyData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed performing ECDH key exchange: %w", err)
	}

	// 3. Send a join request to the remote node, providing the random token and
	// the local X25519 public key. If the token is valid and not expired, the
	// remote node will generate a TLS client certificate, encrypt it with the
	// shared key, and send it in the response.
	c := client.New(address)
	joinResp, err := c.Join(ctx, base58.Encode(tokenData), base58.Encode(pubKeyData))
	if err != nil {
		return nil, err
	}
	tlsClientCertDec, err := base58.Decode(joinResp.TLSClientCertEnc)
	if err != nil {
		return nil, fmt.Errorf("failed decoding TLS client certificate: %w", err)
	}

	// 4. Decrypt the client certificate with the shared key.
	var sharedKeyArr [32]byte
	copy(sharedKeyArr[:], sharedKey)
	tlsClientCertR, err := crypto.DecryptSym(bytes.NewBuffer(tlsClientCertDec), &sharedKeyArr)
	if err != nil {
		return nil, fmt.Errorf("failed decrypting TLS client certificate: %w", err)
	}
	tlsClientCert, err := io.ReadAll(tlsClientCertR)
	if err != nil {
		return nil, fmt.Errorf("failed reading TLS client certificate: %w", err)
	}

	return tlsClientCert, nil
}

func decodeToken(token string) ([]byte, []byte, error) {
	tokenDec, err := base58.Decode(token)
	if err != nil {
		return nil, nil, fmt.Errorf("failed decoding token: %w", err)
	}
	if len(tokenDec) != 64 {
		return nil, nil, ErrInvalidToken
	}

	return tokenDec[:32], tokenDec[32:], nil
}
