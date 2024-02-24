package badger

import (
	badger "github.com/dgraph-io/badger/v4"
	"go.hackfix.me/disco/store"
)

type Badger struct {
	db *badger.DB
}

var _ store.Store = &Badger{}

func Open(path string) (*Badger, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Badger{db: db}, nil
}

func (s *Badger) Close() error {
	return s.db.Close()
}

func (s *Badger) Get(key []byte) ([]byte, error) {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(key)
	if err != nil {
		return nil, err
	}

	val, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (s *Badger) Set(key, value []byte) error {
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	err := txn.Set(key, value)
	if err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Badger) List(prefix []byte) [][]byte {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	// Enable key-only iteration, which is more efficient.
	opts.PrefetchValues = false

	it := txn.NewIterator(opts)
	defer it.Close()

	keys := [][]byte{}
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		keys = append(keys, item.Key())
	}

	return keys
}
