package store

type Store interface {
	Close() error
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
}
