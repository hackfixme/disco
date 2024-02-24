package store

type Store interface {
	Close() error
	Get(namespace string, key []byte) (value []byte, err error)
	Set(namespace string, key, value []byte) error
	List(namespace string, prefix []byte) map[string][][]byte
}
