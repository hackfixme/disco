package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.hackfix.me/disco/db/types"
)

type Remote struct {
	ID        uint64
	CreatedAt time.Time
	Name      string
	Address   string
	TLSCACert string

	tlsClientCertEnc []byte
	tlsClientKeyEnc  []byte
}

// NewRemote creates a new remote object.
func NewRemote(name, address string, tlsCACert string, tlsClientCertEnc, tlsClientKeyEnc []byte) *Remote {
	return &Remote{
		CreatedAt:        time.Now(),
		Name:             name,
		Address:          address,
		TLSCACert:        tlsCACert,
		tlsClientCertEnc: tlsClientCertEnc,
		tlsClientKeyEnc:  tlsClientKeyEnc,
	}
}

// Save stores the remote data in the database. If update is true, either the
// remote ID or name must be set for the lookup.
func (r *Remote) Save(ctx context.Context, d types.Querier, update bool) error {
	var (
		stmt      string
		filterStr string
		op        string
		args      = []any{}
	)
	if update {
		var (
			filter *types.Filter
			err    error
		)
		filter, filterStr, err = r.createFilter(ctx, d, 1)
		if err != nil {
			return fmt.Errorf("failed creating remotes filter: %w", err)
		}
		stmt = fmt.Sprintf(`UPDATE invites SET name = ?, address = ?
							WHERE %s`, filter.Where)
		args = append(args, r.Name, r.Address)
		args = append(args, filter.Args...)
		op = fmt.Sprintf("updating remote with %s", filterStr)
	} else {
		stmt = `INSERT INTO remotes (
				id, created_at, name, address, tls_ca_cert, tls_client_cert_enc, tls_client_key_enc)
				VALUES (NULL, ?, ?, ?, ?, ?, ?)`
		args = append(args, r.CreatedAt, r.Name, r.Address, r.TLSCACert, r.tlsClientCertEnc, r.tlsClientKeyEnc)
		op = "saving new remote"
	}

	res, err := d.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("failed %s: %w", op, err)
	}

	if update {
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return types.ErrNoResult{Msg: fmt.Sprintf("remote with %s doesn't exist", filterStr)}
		}
	} else {
		rID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		r.ID = uint64(rID)
	}

	return err
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

func (r *Remote) createFilter(ctx context.Context, d types.Querier, limit int) (*types.Filter, string, error) {
	var filter *types.Filter
	var filterStr string
	if r.ID != 0 {
		filter = types.NewFilter("id = ?", []any{r.ID})
		filterStr = fmt.Sprintf("ID %d", r.ID)
	} else if r.Name != "" {
		filter = types.NewFilter("name = ?", []any{r.Name})
		filterStr = fmt.Sprintf("name '%s'", r.Name)
	} else {
		return nil, "", errors.New("must provide either an remote ID or name")
	}

	if limit > 0 {
		if count, err := filterCount(ctx, d, "remotes", filter); err != nil {
			return nil, "", err
		} else if count > limit {
			return nil, "", fmt.Errorf("filter %s returns %d results; make the filter more specific", filterStr, count)
		}

		filter.Limit = limit
	}

	return filter, filterStr, nil
}
