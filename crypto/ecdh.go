package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
)

// ECDHExchange performs ECDH key exchange using the X25519 function, and
// returns the generated shared secret key, and the local public key.
// If privKeyData is nil, it generates a new private key.
func ECDHExchange(remotePubKeyData []byte, privKeyData []byte) (sharedKey []byte, pubKey []byte, err error) {
	remotePubKey, err := ecdh.X25519().NewPublicKey(remotePubKeyData)
	if err != nil {
		return nil, nil, err
	}

	var privKey *ecdh.PrivateKey
	if privKeyData == nil {
		privKey, err = ecdh.X25519().GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
	} else {
		privKey, err = ecdh.X25519().NewPrivateKey(privKeyData)
		if err != nil {
			return nil, nil, err
		}
	}

	sharedKey, err = privKey.ECDH(remotePubKey)
	if err != nil {
		return nil, nil, err
	}

	return sharedKey, privKey.PublicKey().Bytes(), nil
}
