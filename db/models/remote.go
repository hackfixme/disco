package models

import (
	"context"
	"time"

	"go.hackfix.me/disco/db/types"
)

type Remote struct {
	ID        uint64
	Name      string
	Address   string
	CreatedAt time.Time

	// Encrypted TLS client certificate
	tlsClientCertEnc []byte
}

// NewRemote creates a new remote object.
func NewRemote(name, address string, tlsClientCertEnc []byte) *Remote {
	return &Remote{
		Name:             name,
		Address:          address,
		tlsClientCertEnc: tlsClientCertEnc,
	}
}

// Save stores the remote data in the database. If update is true, either the
// remote ID or name must be set for the lookup.
func (r *Remote) Save(ctx context.Context, d types.Querier, update bool) error {
	return nil
}

// Load the remote record from the database. The remote ID or name must be set
// for the lookup.
func (r *Remote) Load(ctx context.Context, d types.Querier) error {
	return nil
}

// Delete removes the remote record from the database. Either the remote ID or
// name must be set for the lookup.
func (r *Remote) Delete(ctx context.Context, d types.Querier) error {
	return nil
}

// Remotes returns one or more remotes from the database. An optional filter can
// be passed to limit the results.
func Remotes(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Remote, error) {
	return nil, nil
}
