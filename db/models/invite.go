package models

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/mr-tron/base58"
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
func (inv *Invite) Save(ctx context.Context, d types.Querier) error {
	stmt := `INSERT INTO invites (id, uuid, created_at, expires, user_id, token, privkey_enc)
			VALUES (NULL, ?, ?, ?, ?, ?, ?)`
	_, err := d.ExecContext(ctx, stmt, inv.UUID, inv.CreatedAt, inv.Expires, inv.User.ID, inv.Token, inv.privKeyEnc)

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

// Delete removes the invite record from the database. The invite ID must be set
// for the lookup. It returns an error if the invite doesn't exist.
func (inv *Invite) Delete(ctx context.Context, d types.Querier) error {
	if inv.ID == 0 {
		return fmt.Errorf("failed loading invite: the invite ID must be set")
	}

	return nil
}

// Invites returns one or more invites from the database. An optional filter can
// be passed to limit the results.
func Invites(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Invite, error) {
	invites := []*Invite{}

	return invites, nil
}
