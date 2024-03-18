package models

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/mr-tron/base58"

	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/types"
)

type UserType uint8

const (
	UserTypeLocal UserType = iota + 1
	UserTypeRemote
)

type User struct {
	ID                uint64
	Name              string
	Type              UserType
	Roles             []*Role
	PublicKey         *[32]byte
	PrivateKey        *[32]byte
	PrivateKeyHashEnc sql.Null[string]
}

// Save stores the user data in the database.
func (u *User) Save(ctx context.Context, d types.Querier, update bool) error {
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

	insertStmt := `INSERT %s INTO users
		(id, name, type, public_key, private_key_hash)
		VALUES (NULL, ?, ?, ?, ?)`
	replace := ""
	if update {
		replace = "OR REPLACE"
	}
	res, err := d.ExecContext(ctx, fmt.Sprintf(insertStmt, replace),
		u.Name, u.Type, pubKeyEnc, privKeyHashEnc)
	if err != nil {
		return fmt.Errorf("failed saving user: %w", err)
	}

	uID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed saving user: %w", err)
	}
	u.ID = uint64(uID)

	args := []any{sql.Named("user_id", u.ID)}
	if update {
		delRoles := `DELETE FROM users_roles WHERE user_id = :user_id`
		_, err = d.ExecContext(ctx, delRoles, args...)
		if err != nil {
			return fmt.Errorf("failed deleting existing user roles: %w", err)
		}
	}

	if len(u.Roles) > 0 {
		stmt := `INSERT INTO users_roles (user_id, role_id) VALUES`

		values := []string{}
		for _, role := range u.Roles {
			values = append(values, `(:user_id, ?)`)
			args = append(args, role.ID)
		}
		stmt = fmt.Sprintf("%s %s", stmt, strings.Join(values, ", "))

		_, err = d.ExecContext(ctx, stmt, args...)
		if err != nil {
			return fmt.Errorf("failed saving user roles: %w", err)
		}
	}

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
	*u = *users[0]

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

// Can returns true if the user is allowed to perform the action on the target.
func (u *User) Can(action, target string) (bool, error) {
	if len(u.Roles) == 0 {
		return false, nil
	}
	for _, role := range u.Roles {
		can, err := role.Can(action, target)
		if err != nil {
			return false, err
		}
		if can {
			return true, nil
		}
	}

	return false, nil
}

// Users returns one or more users from the database. An optional filter can be
// passed to limit the results.
func Users(ctx context.Context, d types.Querier, filter *types.Filter) ([]*User, error) {
	query := `SELECT u.id, u.name, u.type, u.public_key, u.private_key_hash,
		(SELECT group_concat(r.id)
		FROM roles r
		INNER JOIN users_roles ur
			ON ur.role_id = r.id
			AND ur.user_id = u.id
		ORDER BY r.name ASC) role_ids
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
	users := []*User{}
	roles := map[string]*Role{}
	type row struct {
		ID             uint64
		UserName       string
		UserType       UserType
		PubKeyEnc      sql.Null[string]
		PrivKeyHashEnc sql.Null[string]
		RoleIDsConcat  sql.Null[string]
	}
	for rows.Next() {
		r := row{}
		err := rows.Scan(&r.ID, &r.UserName, &r.UserType, &r.PubKeyEnc,
			&r.PrivKeyHashEnc, &r.RoleIDsConcat)
		if err != nil {
			return nil, fmt.Errorf("failed scanning user data: %w", err)
		}

		if user == nil || user.Name != r.UserName {
			user = &User{ID: r.ID, Name: r.UserName, Type: r.UserType}
			if r.PubKeyEnc.Valid {
				if user.PublicKey, err = crypto.DecodeKey(r.PubKeyEnc.V); err != nil {
					return nil, fmt.Errorf("failed decoding public key of user ID %d: %w", r.ID, err)
				}
			}
			if r.PrivKeyHashEnc.Valid {
				user.PrivateKeyHashEnc = r.PrivKeyHashEnc
			}

			users = append(users, user)
		}

		if !r.RoleIDsConcat.Valid {
			continue
		}
		roleIDs := strings.Split(r.RoleIDsConcat.V, ",")
		for _, rIDStr := range roleIDs {
			if r, ok := roles[rIDStr]; ok {
				user.Roles = append(user.Roles, r)
				continue
			}
			rID, err := strconv.Atoi(rIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed converting role ID %s: %w", rIDStr, err)
			}
			role := &Role{ID: uint64(rID)}
			if err := role.Load(ctx, d); err != nil {
				return nil, fmt.Errorf("failed loading role ID %d: %w", rID, err)
			}
			user.Roles = append(user.Roles, role)
			roles[rIDStr] = role
		}
	}

	return users, nil
}
