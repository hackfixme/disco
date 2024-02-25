package badger

import (
	"bytes"
	"fmt"
	"slices"
	"time"

	badger "github.com/dgraph-io/badger/v4"
	"go.hackfix.me/disco/store"
)

type Badger struct {
	db *badger.DB
}

var _ store.Store = &Badger{}

func Open(path string, encryptionKey []byte) (*Badger, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil

	if len(encryptionKey) > 0 {
		opts.EncryptionKey = encryptionKey
		opts.EncryptionKeyRotationDuration = 24 * time.Hour
		// Should be set only if using encryption
		opts.IndexCacheSize = 10 << 20 // 10MB
	}

	if path == "" {
		opts.InMemory = true
	}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Badger{db: db}, nil
}

func (s *Badger) Close() error {
	return s.db.Close()
}

func (s *Badger) Get(namespace string, key []byte) ([]byte, error) {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	key = namespaceKey(namespace, key)

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

func (s *Badger) Set(namespace string, key, value []byte) error {
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	key = namespaceKey(namespace, key)

	err := txn.Set(key, value)
	if err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Badger) List(namespace string, prefix []byte) map[string][][]byte {
	txn := s.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	// Enable key-only iteration, which is more efficient.
	opts.PrefetchValues = false

	it := txn.NewIterator(opts)
	defer it.Close()

	keys := map[string][][]byte{}
	if namespace == "*" {
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			ns, key, _ := bytes.Cut(item.Key(), []byte{namespaceSep})
			// Cut returns slices of the original slice, so a copy is needed.
			keyCopy := slices.Clone(key)
			if bytes.HasPrefix(keyCopy, prefix) {
				keys[string(ns)] = append(keys[string(ns)], keyCopy)
			}
		}
	} else {
		prefix = namespaceKey(namespace, prefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			keyClean, _ := bytes.CutPrefix(item.Key(), namespaceKey(namespace, nil))
			// CutPrefix returns a slice of the original slice, so a copy is needed.
			keyCopy := slices.Clone(keyClean)
			keys[namespace] = append(keys[namespace], keyCopy)
		}
	}

	return keys
}

const namespaceSep = '\x00'

// namespaceKey returns a composite key used for lookup and storage for a
// given namespace and key.
func namespaceKey(namespace string, key []byte) []byte {
	prefix := []byte(fmt.Sprintf("%s%c", namespace, namespaceSep))
	return append(prefix, key...)
}
