package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/zpatrick/rbac"

	"go.hackfix.me/disco/db/types"
)

// A Role is a grouping of permissions which guard access to specific resources
// and actions that can be performed upon them.
type Role struct {
	ID          uint64
	Name        string
	Permissions []Permission

	role *rbac.Role
}

// Permission is a mapping of a specific action to a specific target resource.
type Permission struct {
	ActionPattern string
	TargetPattern string
}

// Save the role to the database.
func (r *Role) Save(ctx context.Context, d types.Querier) error {
	res, err := d.ExecContext(ctx,
		`INSERT INTO roles (id, name)
		VALUES (NULL, ?)`, r.Name)
	if err != nil {
		return fmt.Errorf("failed saving role: %w", err)
	}

	rID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed saving role: %w", err)
	}
	r.ID = uint64(rID)

	stmt := `INSERT INTO role_permissions (role_id, action, target) VALUES `
	args := []any{sql.Named("role_id", rID)}
	values := []string{}
	for _, perm := range r.Permissions {
		values = append(values, `(:role_id, ?, ?)`)
		args = append(args, perm.ActionPattern, perm.TargetPattern)
	}
	stmt = fmt.Sprintf("%s %s", stmt, strings.Join(values, ", "))

	_, err = d.ExecContext(ctx, stmt, args...)
	if err != nil {
		return fmt.Errorf("failed saving role: %w", err)
	}

	return nil
}

// Can returns true if the role is allowed to perform the action on the target.
func (r *Role) Can(action, target string) (bool, error) {
	if r.role == nil {
		perms := make([]rbac.Permission, len(r.Permissions))
		for i, perm := range r.Permissions {
			perms[i] = rbac.NewGlobPermission(perm.ActionPattern, perm.TargetPattern)
		}
		r.role = &rbac.Role{RoleID: r.Name, Permissions: perms}
	}

	return r.role.Can(action, target)
}

// Load the role data from the database. Either the role ID or Name must be set
// for the lookup.
func (r *Role) Load(ctx context.Context, d types.Querier) error {
	if r.ID == 0 && r.Name == "" {
		return errors.New("either user ID or Name must be set")
	}

	var (
		filter    *types.Filter
		filterStr string
	)
	if r.ID != 0 {
		filter = types.NewFilter("r.id = ?", []any{r.ID})
		filterStr = fmt.Sprintf("ID %d", r.ID)
	} else if r.Name != "" {
		filter = types.NewFilter("r.name = ?", []any{r.Name})
		filterStr = fmt.Sprintf("name '%s'", r.Name)
	}

	roles, err := Roles(ctx, d, filter)
	if err != nil {
		return err
	}

	if len(roles) == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("role with %s doesn't exist", filterStr)}
	}

	// This is dodgy, but the unique constraint on both users.id and users.name
	// should return only a single result.
	if len(roles) > 1 {
		panic(fmt.Sprintf("roles query returned more than 1 role: %d", len(roles)))
	}
	*r = *roles[0]

	return nil
}

// Delete removes the role data from the database. Either the user ID or Name
// must be set for the lookup. It returns an error if the role doesn't exist.
func (r *Role) Delete(ctx context.Context, d types.Querier) error {
	if r.ID == 0 && r.Name == "" {
		return fmt.Errorf("failed deleting role: either role ID or Name must be set")
	}

	var filter *types.Filter
	var filterStr string
	if r.ID != 0 {
		filter = &types.Filter{Where: "id = ?", Args: []any{r.ID}}
		filterStr = fmt.Sprintf("ID %d", r.ID)
	} else if r.Name != "" {
		filter = &types.Filter{Where: "name = ?", Args: []any{r.Name}}
		filterStr = fmt.Sprintf("name '%s'", r.Name)
	}

	// TODO: Handle FKs and cascade
	stmt := fmt.Sprintf(`DELETE FROM roles WHERE %s`, filter.Where)

	res, err := d.ExecContext(ctx, stmt, filter.Args...)
	if err != nil {
		return fmt.Errorf("failed deleting role with %s: %w", filterStr, err)
	}

	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return types.ErrNoResult{Msg: fmt.Sprintf("role with %s doesn't exist", filterStr)}
	}

	return nil
}

// Roles returns one or more roles from the database. An optional filter can be
// passed to limit the results.
func Roles(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Role, error) {
	query := `SELECT r.id, r.name, rp.action, rp.target
		FROM roles r
		LEFT JOIN role_permissions rp
			ON r.id = rp.role_id
		%s ORDER BY r.name ASC`

	where := "1=1"
	args := []any{}
	if filter != nil {
		where = filter.Where
		args = filter.Args
	}

	query = fmt.Sprintf(query, fmt.Sprintf("WHERE %s", where))

	rows, err := d.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed loading roles: %w", err)
	}

	var role *Role
	roles := []*Role{}
	type row struct {
		ID       uint64
		RoleName string
		Action   sql.Null[string]
		Target   sql.Null[string]
	}
	for rows.Next() {
		r := row{}
		err := rows.Scan(&r.ID, &r.RoleName, &r.Action, &r.Target)
		if err != nil {
			return nil, fmt.Errorf("failed scanning role data: %w", err)
		}

		if role == nil || role.Name != r.RoleName {
			role = &Role{ID: r.ID, Name: r.RoleName}
			roles = append(roles, role)
		}
		perm := Permission{}
		if r.Action.Valid {
			perm.ActionPattern = r.Action.V
		}
		if r.Target.Valid {
			perm.TargetPattern = r.Target.V
		}

		role.Permissions = append(role.Permissions, perm)
	}

	return roles, nil
}
