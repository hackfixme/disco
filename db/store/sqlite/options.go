package sqlite

// Option is a function that allows configuring the store.
type Option func(*Store) error

// WithEncryptionKey sets the store encryption key.
func WithEncryptionKey(encKey *[32]byte) Option {
	return func(s *Store) error {
		s.encKey = encKey
		return nil
	}
}
