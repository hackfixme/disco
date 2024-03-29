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
	PublicKey string

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

	return &Invite{
		UUID:       uuidgen(),
		CreatedAt:  createdAt,
		Expires:    expires,
		User:       user,
		Token:      base58.Encode(b),
		PublicKey:  base58.Encode(privKey.PublicKey().Bytes()),
		privKeyEnc: privKeyEncData,
	}, nil
}

// Save stores the invite data in the database. If update is true, either the
// invite ID or UUID must be set for the lookup. The UUID may be a prefix, as
// long as it matches exactly one record. It returns an error if the invite
// doesn't exist, or if more than one record would be updated.
func (inv *Invite) Save(ctx context.Context, d types.Querier, update bool) error {
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
		filter, filterStr, err = inv.createFilter(ctx, d, 1)
		if err != nil {
			return fmt.Errorf("failed updating invite: %w", err)
		}
		stmt = fmt.Sprintf(`UPDATE invites SET expires = ? WHERE %s`, filter.Where)
		args = append(args, inv.Expires)
		args = append(args, filter.Args...)
		op = fmt.Sprintf("updating invite with %s", filterStr)
	} else {
		stmt = `INSERT INTO invites (
				id, uuid, created_at, expires, user_id, token, public_key, privkey_enc)
				VALUES (NULL, ?, ?, ?, ?, ?, ?, ?)`
		args = append(args, inv.UUID, inv.CreatedAt, inv.Expires, inv.User.ID, inv.Token, inv.PublicKey, inv.privKeyEnc)
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
	filter, filterStr, err := inv.createFilter(ctx, d, 1)
	if err != nil {
		return fmt.Errorf("failed deleting invite: %w", err)
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

// TokenComposite generates the final token by concatenating the random token
// with the X25519 public key.
func (inv *Invite) TokenComposite() (string, error) {
	tokenDec, err := base58.Decode(inv.Token)
	if err != nil {
		return "", err
	}
	pubKeyDec, err := base58.Decode(inv.PublicKey)
	if err != nil {
		return "", err
	}

	return base58.Encode(slices.Concat(tokenDec, pubKeyDec)), nil
}

// PrivateKey returns the decrypted X25519 private key.
func (inv *Invite) PrivateKey(encryptionKey *[32]byte) (*ecdh.PrivateKey, error) {
	privKeyDataR, err := crypto.DecryptSym(bytes.NewReader(inv.privKeyEnc), encryptionKey)
	if err != nil {
		return nil, err
	}
	privKeyData, err := io.ReadAll(privKeyDataR)
	if err != nil {
		return nil, err
	}
	privKey, err := ecdh.X25519().NewPrivateKey(privKeyData)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

func (inv *Invite) createFilter(ctx context.Context, d types.Querier, limit int) (*types.Filter, string, error) {
	var filter *types.Filter
	var filterStr string
	if inv.ID != 0 {
		filter = types.NewFilter("id = ?", []any{inv.ID})
		filterStr = fmt.Sprintf("ID %d", inv.ID)
	} else if inv.UUID != "" {
		if !cuid2.IsCuid(inv.UUID) {
			return nil, "", fmt.Errorf("invalid invite UUID: '%s'", inv.UUID)
		}
		if len(inv.UUID) < 12 {
			filter = types.NewFilter("uuid LIKE ?", []any{fmt.Sprintf("%s%%", inv.UUID)})
			filterStr = fmt.Sprintf("UUID '%s*'", inv.UUID)
		} else {
			filter = types.NewFilter("uuid = ?", []any{inv.UUID})
			filterStr = fmt.Sprintf("UUID '%s'", inv.UUID)
		}
	} else if inv.Token != "" {
		filter = types.NewFilter("token = ?", []any{inv.Token}).
			And(types.NewFilter("expires > ?", []any{time.Now().UTC()}))
		filterStr = fmt.Sprintf("token '%s'", inv.Token)
	} else {
		return nil, "", errors.New("must provide either an invite ID, UUID or token")
	}

	if count, err := filterCount(ctx, d, "invites", filter); err != nil {
		return nil, "", err
	} else if count > limit {
		return nil, "", fmt.Errorf("filter %s returns %d results; make the filter more specific", filterStr, count)
	}

	filter.Limit = limit

	return filter, filterStr, nil
}

// Invites returns one or more invites from the database. An optional filter can
// be passed to limit the results.
func Invites(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Invite, error) {
	queryFmt := `SELECT inv.id, inv.uuid, inv.created_at, inv.expires, inv.user_id, inv.token, inv.public_key, inv.privkey_enc
		FROM invites inv
		%s ORDER BY inv.expires ASC %s`

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
		return nil, fmt.Errorf("failed loading invites: %w", err)
	}

	invites := []*Invite{}
	users := map[uint64]*User{}
	for rows.Next() {
		inv := Invite{}
		var userID uint64
		err := rows.Scan(&inv.ID, &inv.UUID, &inv.CreatedAt, &inv.Expires, &userID, &inv.Token, &inv.PublicKey, &inv.privKeyEnc)
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
