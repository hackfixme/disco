package store

import (
	"io"
	"log/slog"
)

// Store defines the operations data stores must implement to store and retrieve
// data.
type Store interface {
	Init(appVersion string, logger *slog.Logger) error
	Close() error
	Get(namespace, key string) (ok bool, value io.Reader, err error)
	Set(namespace, key string, value io.Reader) error
	Delete(namespace, key string) error
	List(namespace, keyPrefix string) (map[string][]string, error)
}
