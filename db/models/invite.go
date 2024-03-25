package models

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/mr-tron/base58"
	"github.com/nrednav/cuid2"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/types"
)

type Invite struct {
	ID        uint64
	UUID      string
	CreatedAt time.Time
	Expires   time.Time
	User      *User
	Token     string

	// Encrypted X25519 private key
	privKeyEnc []byte
}

// NewInvite creates a new invitation for a remote user. A unique token is
// created that must be supplied when authenticating to the server. The token is
// constructed by concatenating random 32 bytes and an ephemeral X25519
// public key, encoded as a base 58 string.
// The encryptionKey is a separate persistent symmetric key used for encrypting
// the X25519 private key.
func NewInvite(user *User, ttl time.Duration, uuidgen func() string, encryptionKey *[32]byte) (*Invite, error) {
	privKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	privKeyR := bytes.NewReader(privKey.Bytes())
	privKeyEnc, err := crypto.EncryptSym(privKeyR, encryptionKey)
	if err != nil {
		return nil, err
	}
	privKeyEncData, err := io.ReadAll(privKeyEnc)
	if err != nil {
		return nil, err
	}

	createdAt := time.Now().UTC()
	expires := createdAt.Add(ttl)
	token := base58.Encode(slices.Concat(b, privKey.PublicKey().Bytes()))

	return &Invite{
		UUID:       uuidgen(),
		CreatedAt:  createdAt,
		Expires:    expires,
		User:       user,
		Token:      token,
		privKeyEnc: privKeyEncData,
	}, nil
}

// Save stores the invite data in the database.
func (inv *Invite) Save(ctx context.Context, d types.Querier, update bool) error {
	var (
		stmt      string
		filterStr string
		op        string
		args      = make([]any, 0)
	)
	if update {
		var filter *types.Filter
		if inv.ID != 0 {
			filter = &types.Filter{Where: "id = ?", Args: []any{inv.ID}}
			filterStr = fmt.Sprintf("ID %d", inv.ID)
		} else if inv.UUID != "" {
			filter = &types.Filter{Where: "uuid = ?", Args: []any{inv.UUID}}
			filterStr = fmt.Sprintf("UUID '%s'", inv.UUID)
		} else {
			return errors.New("must provide either an invite ID or UUID to update")
		}
		stmt = fmt.Sprintf(`UPDATE invites SET expires = ? WHERE %s`, filter.Where)
		args = append(args, inv.Expires)
		args = append(args, filter.Args...)
		op = fmt.Sprintf("updating invite with %s", filterStr)
	} else {
		stmt = `INSERT INTO invites (
				id, uuid, created_at, expires, user_id, token, privkey_enc)
				VALUES (NULL, ?, ?, ?, ?, ?, ?)`
		args = append(args, inv.UUID, inv.CreatedAt, inv.Expires, inv.User.ID, inv.Token, inv.privKeyEnc)
		op = "saving new invite"
	}

	res, err := d.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("failed %s: %w", op, err)
	}

	if update {
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return types.ErrNoResult{Msg: fmt.Sprintf("invite with %s doesn't exist", filterStr)}
		}
	}

	return err
}

// Load the invite record from the database. The invite ID must be set for the
// lookup.
func (inv *Invite) Load(ctx context.Context, d types.Querier) error {
	if inv.ID == 0 {
		return fmt.Errorf("failed loading invite: the invite ID must be set")
	}

	return nil
}

// Delete removes the invite record from the database. Either the invite ID or
// UUID must be set for the lookup. The UUID may be a prefix, as long as it
// matches exactly one record. It returns an error if the invite doesn't exist,
// or if more than one record would be deleted.
func (inv *Invite) Delete(ctx context.Context, d types.Querier) error {
	if inv.ID == 0 && inv.UUID == "" {
		return fmt.Errorf("failed deleting invite: either invite ID or UUID must be set")
	}

	var filter *types.Filter
	var filterStr string
	if inv.ID != 0 {
		filter = &types.Filter{Where: "id = ?", Args: []any{inv.ID}}
		filterStr = fmt.Sprintf("ID %d", inv.ID)
	} else if inv.UUID != "" {
		if !cuid2.IsCuid(inv.UUID) {
			return fmt.Errorf("invalid invite UUID: '%s'", inv.UUID)
		}
		if len(inv.UUID) < 12 {
			filter = &types.Filter{Where: "uuid LIKE ?", Args: []any{fmt.Sprintf("%s%%", inv.UUID)}}
			filterStr = fmt.Sprintf("UUID '%s*'", inv.UUID)
			if err := validateInviteDelete(ctx, d, filter, filterStr); err != nil {
				return err
			}
		} else {
			filter = &types.Filter{Where: "uuid = ?", Args: []any{inv.UUID}}
			filterStr = fmt.Sprintf("UUID '%s'", inv.UUID)
		}
	}

	stmt := fmt.Sprintf(`DELETE FROM invites WHERE %s`, filter.Where)
	res, err := d.ExecContext(ctx, stmt, filter.Args...)
	if err != nil {
		return fmt.Errorf("failed deleting invite with %s: %w", filterStr, err)
	}

	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("invite with %s doesn't exist", filterStr)}
	}

	return nil
}

func validateInviteDelete(ctx context.Context, d types.Querier, filter *types.Filter, filterStr string) error {
	checkQ := fmt.Sprintf(`SELECT COUNT(*) FROM invites WHERE %s`, filter.Where)
	var toDeleteCount int
	err := d.QueryRowContext(ctx, checkQ, filter.Args...).Scan(&toDeleteCount)
	if err != nil {
		return fmt.Errorf("failed validating invite deletion: %w", err)
	}

	if toDeleteCount > 1 {
		return fmt.Errorf("invite filter %s would delete %d invites; make the filter more specific", filterStr, toDeleteCount)
	}

	return nil
}

// Invites returns one or more invites from the database. An optional filter can
// be passed to limit the results.
func Invites(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Invite, error) {
	query := `SELECT inv.id, inv.uuid, inv.created_at, inv.expires, inv.user_id, inv.token, inv.privkey_enc
		FROM invites inv
		%s ORDER BY inv.expires ASC`

	where := "1=1"
	args := []any{}
	if filter != nil {
		where = filter.Where
		args = filter.Args
	}

	query = fmt.Sprintf(query, fmt.Sprintf("WHERE %s", where))

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed loading invites: %w", err)
	}

	invites := []*Invite{}
	users := map[uint64]*User{}
	for rows.Next() {
		inv := Invite{}
		var userID uint64
		err := rows.Scan(&inv.ID, &inv.UUID, &inv.CreatedAt, &inv.Expires, &userID, &inv.Token, &inv.privKeyEnc)
		if err != nil {
			return nil, fmt.Errorf("failed scanning invite data: %w", err)
		}

		// TODO: Load users in the same query for efficiency
		user, ok := users[userID]
		if !ok {
			user = &User{ID: userID}
			if err = user.Load(ctx, d); err != nil {
				return nil, fmt.Errorf("failed loading invite user: %w", err)
			}
			users[userID] = user
		}

		inv.User = user
		invites = append(invites, &inv)
	}

	return invites, nil
}
