package store

// Store defines the operations data stores must implement to store and retrieve
// data.
type Store interface {
	Close() error
	Get(namespace, key string) (ok bool, value []byte, err error)
	Set(namespace, key string, value []byte) error
	List(namespace, keyPrefix string) (map[string][]string, error)
}
