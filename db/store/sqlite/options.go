package sqlite

import (
	"encoding/hex"

	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/queries"
)

// Option is a function that allows configuring the store.
type Option func(*Store) error

// WithEncryptionKey validates and sets the store encryption key.
func WithEncryptionKey(key string) Option {
	return func(s *Store) error {
		existingKeyHash, err := queries.GetEncryptionKeyHash(s.ctx, s)
		if err != nil || !existingKeyHash.Valid {
			return aerrors.NewRuntimeError("missing encryption key", nil,
				"Did you forget to run 'disco init'?")
		}

		encKey, decodeErr := crypto.DecodeHexKey(key)
		if decodeErr != nil {
			return aerrors.NewRuntimeError("invalid encryption key", decodeErr, "")
		}

		keyHash := crypto.Hash("encryption key hash", encKey[:])
		keyHashHex := hex.EncodeToString(keyHash)
		if existingKeyHash.V != keyHashHex {
			return aerrors.NewRuntimeError("invalid encryption key", nil, "")
		}

		s.encKey = encKey

		return nil
	}
}
