package sqlite

import (
	"github.com/mr-tron/base58"

	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/queries"
)

// Option is a function that allows configuring the store.
type Option func(*Store) error

// WithEncryptionKey validates and sets the store encryption key.
func WithEncryptionKey(privKeyEnc string) Option {
	return func(s *Store) error {
		privKeyHash, privKeyErr := queries.GetEncryptionPrivKeyHash(s.ctx, s)
		pubKeyEnc, pubKeyErr := queries.GetEncryptionPubKey(s.ctx, s)
		if privKeyErr != nil || !privKeyHash.Valid ||
			pubKeyErr != nil || !pubKeyEnc.Valid {
			return aerrors.NewRuntimeError("missing encryption key", nil,
				"Did you forget to run 'disco init'?")
		}

		privKey, decKeyErr := crypto.DecodeKey(privKeyEnc)
		pubKey, decPubKeyErr := crypto.DecodeKey(pubKeyEnc.V)
		if decKeyErr == nil {
			decKeyErr = decPubKeyErr
		}
		if decKeyErr != nil {
			return aerrors.NewRuntimeError("invalid encryption key", decKeyErr, "")
		}

		inPrivKeyHash := crypto.Hash("encryption key hash", privKey[:])
		inPrivKeyHashEnc := base58.Encode(inPrivKeyHash)
		if privKeyHash.V != inPrivKeyHashEnc {
			return aerrors.NewRuntimeError("invalid encryption key", nil, "")
		}

		s.encPubKey = pubKey
		s.encPrivKey = privKey

		return nil
	}
}
