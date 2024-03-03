package db

// Store defines the operations data stores must implement to store and retrieve
// data.
type Store interface {
	Close() error
	Get(namespace, key string) (value []byte, err error)
	Set(namespace, key string, value []byte) error
	List(namespace, prefix string) map[string][][]byte
}

var _ Store = &DB{}

func (db *DB) Get(namespace, key string) (value []byte, err error) {
	return nil, nil
}

func (db *DB) Set(namespace, key string, value []byte) error {
	return nil
}

func (db *DB) List(namespace, key string) map[string][][]byte {
	return nil
}
