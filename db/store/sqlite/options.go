package sqlite

import (
	"github.com/mr-tron/base58"

	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/types"
)

// Option is a function that allows configuring the store.
type Option func(*Store) error

// WithEncryptionKey validates and sets the store encryption key.
func WithEncryptionKey(d types.Querier, privKeyEnc string) Option {
	return func(s *Store) error {
		privKeyHash, privKeyErr := queries.GetEncryptionPrivKeyHash(d.NewContext(), d)
		if privKeyErr != nil || !privKeyHash.Valid {
			return aerrors.NewRuntimeError("missing encryption key", privKeyErr,
				"Did you forget to run 'disco init'?")
		}

		privKey, err := crypto.DecodeKey(privKeyEnc)
		if err != nil {
			return aerrors.NewRuntimeError("invalid encryption key", err, "")
		}

		inPrivKeyHash := crypto.Hash("encryption key hash", privKey[:])
		inPrivKeyHashEnc := base58.Encode(inPrivKeyHash)
		if privKeyHash.V != inPrivKeyHashEnc {
			return aerrors.NewRuntimeError("invalid encryption key", nil, "")
		}

		s.privKey = privKey

		return nil
	}
}
