package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/mr-tron/base58"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/types"
)

type User struct {
	ID                uint64
	Name              string
	PublicKey         *[32]byte
	PrivateKey        *[32]byte
	PrivateKeyHashEnc sql.Null[string]
}

// Save stores the user data in the database.
func (u *User) Save(ctx context.Context, d types.Querier) error {
	var pubKeyEnc sql.Null[string]
	if u.PublicKey != nil {
		pubKeyEnc.V = base58.Encode(u.PublicKey[:])
		pubKeyEnc.Valid = true
	}
	var privKeyHashEnc sql.Null[string]
	if u.PrivateKey != nil {
		privKeyHash := crypto.Hash("encryption key hash", u.PrivateKey[:])
		privKeyHashEnc.V = base58.Encode(privKeyHash)
		privKeyHashEnc.Valid = true
		u.PrivateKeyHashEnc = privKeyHashEnc
	}
	res, err := d.ExecContext(ctx,
		`INSERT INTO users (id, name, public_key, private_key_hash)
		VALUES (NULL, ?, ?, ?)`,
		u.Name, pubKeyEnc, privKeyHashEnc)
	if err != nil {
		return fmt.Errorf("failed saving user: %w", err)
	}

	uID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed saving role: %w", err)
	}
	u.ID = uint64(uID)

	return nil
}

// Load the user data from the database. Either the user ID or Name must be set
// for the lookup.
func (u *User) Load(ctx context.Context, d types.Querier) error {
	if u.ID == 0 && u.Name == "" {
		return fmt.Errorf("failed loading user: either user ID or Name must be set")
	}

	var filter *types.Filter
	var filterStr string
	if u.ID != 0 {
		filter = &types.Filter{Where: "u.id = ?", Args: []any{u.ID}}
		filterStr = fmt.Sprintf("ID %d", u.ID)
	} else if u.Name != "" {
		filter = &types.Filter{Where: "u.name = ?", Args: []any{u.Name}}
		filterStr = fmt.Sprintf("name '%s'", u.Name)
	}

	users, err := Users(ctx, d, filter)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("user with %s doesn't exist", filterStr)}
	}

	// This is dodgy, but the unique constraint on both users.id and users.name
	// should return only a single result.
	if len(users) > 1 {
		panic(fmt.Sprintf("users query returned more than 1 user: %d", len(users)))
	}
	for _, user := range users {
		*u = *user
	}

	return nil
}

// Delete removes the user data from the database. Either the user ID or Name
// must be set for the lookup. It returns an error if the user doesn't exist.
func (u *User) Delete(ctx context.Context, d types.Querier) error {
	if u.ID == 0 && u.Name == "" {
		return fmt.Errorf("failed deleting user: either user ID or Name must be set")
	}

	var filter *types.Filter
	var filterStr string
	if u.ID != 0 {
		filter = &types.Filter{Where: "id = ?", Args: []any{u.ID}}
		filterStr = fmt.Sprintf("ID %d", u.ID)
	} else if u.Name != "" {
		filter = &types.Filter{Where: "name = ?", Args: []any{u.Name}}
		filterStr = fmt.Sprintf("name '%s'", u.Name)
	}

	// TODO: Handle FKs and cascade
	stmt := fmt.Sprintf(`DELETE FROM users WHERE %s`, filter.Where)

	res, err := d.ExecContext(ctx, stmt, filter.Args...)
	if err != nil {
		return fmt.Errorf("failed deleting user with %s: %w", filterStr, err)
	}

	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("user with %s doesn't exist", filterStr)}
	}

	return nil
}

// Users returns one or more users from the database, indexed by their name. An
// optional filter can be passed to limit the results.
func Users(ctx context.Context, d types.Querier, filter *types.Filter) (map[string]*User, error) {
	query := `SELECT u.id, u.name, u.public_key, u.private_key_hash
		FROM users u %s
		ORDER BY u.name ASC`

	where := "1=1"
	args := []any{}
	if filter != nil {
		where = filter.Where
		args = filter.Args
	}

	query = fmt.Sprintf(query, fmt.Sprintf("WHERE %s", where))

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed loading users: %w", err)
	}

	var user *User
	users := map[string]*User{}
	type row struct {
		ID             uint64
		UserName       string
		PubKeyEnc      sql.Null[string]
		PrivKeyHashEnc sql.Null[string]
	}
	for rows.Next() {
		r := row{}
		err := rows.Scan(&r.ID, &r.UserName, &r.PubKeyEnc, &r.PrivKeyHashEnc)
		if err != nil {
			return nil, fmt.Errorf("failed scanning user data: %w", err)
		}

		if user == nil || user.Name != r.UserName {
			user = &User{ID: r.ID, Name: r.UserName}
			if r.PubKeyEnc.Valid {
				if user.PublicKey, err = crypto.DecodeKey(r.PubKeyEnc.V); err != nil {
					return nil, fmt.Errorf("failed decoding public key of user ID %d: %w", r.ID, err)
				}
			}
			if r.PrivKeyHashEnc.Valid {
				user.PrivateKeyHashEnc = r.PrivKeyHashEnc
			}

			users[user.Name] = user
		}
	}

	return users, nil
}
