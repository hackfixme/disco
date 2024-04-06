package models

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/types"
)

type Remote struct {
	ID           uint64
	CreatedAt    time.Time
	Name         string
	Address      string
	TLSCACert    string
	TLSServerSAN string

	tlsClientCertEnc []byte
	tlsClientKeyEnc  []byte
}

// NewRemote creates a new remote object.
func NewRemote(
	name, address, tlsCACert, tlsServerSAN string, tlsClientCertEnc,
	tlsClientKeyEnc []byte,
) *Remote {
	return &Remote{
		CreatedAt:        time.Now(),
		Name:             name,
		Address:          address,
		TLSCACert:        tlsCACert,
		TLSServerSAN:     tlsServerSAN,
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
				id, created_at, name, address, tls_ca_cert, tls_server_san, tls_client_cert_enc, tls_client_key_enc)
				VALUES (NULL, ?, ?, ?, ?, ?, ?, ?)`
		args = append(args, r.CreatedAt, r.Name, r.Address, r.TLSCACert,
			r.TLSServerSAN, r.tlsClientCertEnc, r.tlsClientKeyEnc)
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
	filter, filterStr, err := r.createFilter(ctx, d, 1)
	if err != nil {
		return fmt.Errorf("failed loading remote: %w", err)
	}

	remotes, err := Remotes(ctx, d, filter)
	if err != nil {
		return err
	}

	if len(remotes) == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("remote with %s doesn't exist", filterStr)}
	}

	*r = *remotes[0]

	return nil
}

// Delete removes the remote record from the database. Either the remote ID or
// name must be set for the lookup.
func (r *Remote) Delete(ctx context.Context, d types.Querier) error {
	return nil
}

// ClientTLSConfig returns the TLS client configuration.
func (r *Remote) ClientTLSConfig(encKey *[32]byte) (*tls.Config, error) {
	tlsConfig := crypto.DefaultTLSConfig()

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(r.TLSCACert))
	tlsConfig.RootCAs = caCertPool

	tlsClientCert, err := r.clientTLSCert(encKey)
	if err != nil {
		return nil, err
	}
	tlsConfig.Certificates = []tls.Certificate{*tlsClientCert}
	tlsConfig.ServerName = r.TLSServerSAN

	return tlsConfig, nil
}

// clientTLSCert returns the unencrypted TLS client certificate and private key
// pair.
func (r *Remote) clientTLSCert(encKey *[32]byte) (*tls.Certificate, error) {
	tlsClientCert, err := crypto.DecryptSymInMemory(r.tlsClientCertEnc, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed decrypting TLS client certificate: %w", err)
	}

	tlsClientKey, err := crypto.DecryptSymInMemory(r.tlsClientKeyEnc, encKey)
	if err != nil {
		return nil, fmt.Errorf("failed decrypting TLS client private key: %w", err)
	}

	certPair, err := tls.X509KeyPair(tlsClientCert, tlsClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed parsing PEM encoded TLS client certificate: %w", err)
	}

	return &certPair, nil
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

// Remotes returns one or more remotes from the database. An optional filter can
// be passed to limit the results.
func Remotes(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Remote, error) {
	queryFmt := `SELECT r.id, r.created_at, r.name, r.address,
					r.tls_ca_cert, r.tls_server_san, r.tls_client_cert_enc,
					r.tls_client_key_enc
				FROM remotes r
				%s ORDER BY r.name ASC %s`

	where := "1=1"
	var limit string
	args := []any{}
	if filter != nil {
		where = filter.Where
		args = filter.Args
		if filter.Limit > 0 {
			limit = fmt.Sprintf("LIMIT %d", filter.Limit)
		}
	}

	query := fmt.Sprintf(queryFmt, fmt.Sprintf("WHERE %s", where), limit)

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed loading remotes: %w", err)
	}

	remotes := []*Remote{}
	for rows.Next() {
		r := Remote{}
		err := rows.Scan(&r.ID, &r.CreatedAt, &r.Name, &r.Address, &r.TLSCACert,
			&r.TLSServerSAN, &r.tlsClientCertEnc, &r.tlsClientKeyEnc)
		if err != nil {
			return nil, fmt.Errorf("failed scanning remote data: %w", err)
		}

		remotes = append(remotes, &r)
	}

	return remotes, nil
}
