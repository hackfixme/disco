package models

import (
	"bytes"
	"context"
	"database/sql"
	"encoding"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/zpatrick/rbac"

	"go.hackfix.me/disco/db/types"
)

// Actions are activities that can be performed on resources.
type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionDelete Action = "delete"
	ActionAny    Action = "*"
)

// ActionFromString returns a valid Action from a string value.
func ActionFromString(act string) (Action, error) {
	switch act {
	case "read":
		return ActionRead, nil
	case "write":
		return ActionWrite, nil
	case "delete":
		return ActionDelete, nil
	case "*":
		return ActionAny, nil
	default:
		return "", fmt.Errorf("invalid action '%s'", act)
	}
}

// Resources are object types that can be acted upon.
type Resource string

const (
	ResourceStore Resource = "store"
	ResourceUser  Resource = "user"
	ResourceRole  Resource = "role"
	ResourceAny   Resource = "*"
)

// ResourceFromString returns a valid Resource from a string value.
func ResourceFromString(res string) (Resource, error) {
	switch res {
	case "store":
		return ResourceStore, nil
	case "user":
		return ResourceUser, nil
	case "role":
		return ResourceRole, nil
	case "*":
		return ResourceAny, nil
	default:
		return "", fmt.Errorf("invalid resource '%s'", res)
	}
}

// A Role is a grouping of permissions which guard access to specific resources
// and actions that can be performed upon them.
type Role struct {
	ID          uint64
	Name        string
	Permissions []Permission

	role *rbac.Role
}

// Permission is a combination of access rules. It declares the actions allowed
// for a specific target in one or more namespaces.
// Namespaces are arbitrary and can be created at runtime by the user.
// The target can either be a static resource name, or a pattern that includes
// wildcards, e.g. 'store:myapp/*'. Namespaces and actions can also be a
// wildcard, to allow any action in any namespace (e.g. for admin roles).
type Permission struct {
	Namespaces map[string]struct{}
	Actions    map[Action]struct{}
	Target     PermissionTarget
}

type PermissionTarget struct {
	Resource Resource
	Patterns []string
}

// Save the role to the database.
func (r *Role) Save(ctx context.Context, d types.Querier, update bool) error {
	if update {
		// The roles table doesn't need to be updated, just the role
		// permissions. We won't allow role renaming. So just load the role to
		// get its ID, but preserve the passed permissions.
		perms := r.Permissions
		if err := r.Load(ctx, d); err != nil {
			return err
		}
		r.Permissions = perms
	} else {
		insertStmt := `INSERT INTO roles (id, name) VALUES (NULL, ?)`
		res, err := d.ExecContext(ctx, insertStmt, r.Name)
		if err != nil {
			return err
		}

		rID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		r.ID = uint64(rID)
	}

	args := []any{sql.Named("role_id", r.ID)}
	if update {
		delPerms := `DELETE FROM role_permissions WHERE role_id = :role_id`
		_, err := d.ExecContext(ctx, delPerms, args...)
		if err != nil {
			return fmt.Errorf("failed deleting existing role permissions: %w", err)
		}
	}

	stmt := `INSERT INTO role_permissions (role_id, namespaces, actions, target) VALUES `
	values := []string{}
	for _, perm := range r.Permissions {
		namespaces := make([]string, 0, len(perm.Namespaces))
		for ns := range perm.Namespaces {
			namespaces = append(namespaces, ns)
		}
		slices.Sort(namespaces)

		actions := make([]byte, 0, len(perm.Actions))
		for action := range perm.Actions {
			_, ok := actionMap[rune(action[0])]
			if !ok {
				return fmt.Errorf("invalid action '%s'", action)
			}
			actions = append(actions, action[0])
		}
		slices.Sort(actions)

		values = append(values, `(:role_id, ?, ?, ?)`)
		var target string
		if perm.Target.Resource == ResourceAny {
			target = "*"
		} else {
			target = fmt.Sprintf("%s:%s", perm.Target.Resource, strings.Join(perm.Target.Patterns, ","))
		}
		args = append(args, strings.Join(namespaces, ","), string(actions), target)
	}

	stmt = fmt.Sprintf("%s %s", stmt, strings.Join(values, ", "))

	_, err := d.ExecContext(ctx, stmt, args...)
	if err != nil {
		return err
	}

	return nil
}

// Can returns true if the role is allowed to perform the action on the target.
func (r *Role) Can(action, target string) (bool, error) {
	if r.role == nil {
		perms := []rbac.Permission{}
		for _, perm := range r.Permissions {
			for act := range perm.Actions {
				for ns := range perm.Namespaces {
					for _, pat := range perm.Target.Patterns {
						perms = append(perms,
							rbac.NewGlobPermission(string(act),
								fmt.Sprintf("%s:%s:%s", ns, perm.Target.Resource, pat)))
					}
				}
			}
		}
		r.role = &rbac.Role{RoleID: r.Name, Permissions: perms}
	}

	return r.role.Can(action, target)
}

// Load the role data from the database. Either the role ID or Name must be set
// for the lookup.
func (r *Role) Load(ctx context.Context, d types.Querier) error {
	if r.ID == 0 && r.Name == "" {
		return errors.New("either role ID or Name must be set")
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
// If force is true, it will remove the role even if it's currently assigned to
// existing users.
func (r *Role) Delete(ctx context.Context, d types.Querier, force bool) error {
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

	if !force {
		origFilterWhere := filter.Where
		filter.Where = fmt.Sprintf("r.%s", filter.Where)
		usersWithRoleCount, err := usersWithRole(ctx, d, filter)
		if err != nil {
			return err
		}

		if usersWithRoleCount > 0 {
			causeMsg := "%d users have this role"
			if usersWithRoleCount == 1 {
				causeMsg = "%d user has this role"
			}
			return &types.ErrReference{
				Msg:   fmt.Sprintf("failed deleting role with %s", filterStr),
				Cause: fmt.Errorf(causeMsg, usersWithRoleCount),
			}
		}
		filter.Where = origFilterWhere
	}

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

func usersWithRole(ctx context.Context, d types.Querier, filter *types.Filter) (int, error) {
	stmt := fmt.Sprintf(
		`SELECT COUNT(*)
		FROM roles r
		INNER JOIN users_roles ur
			ON ur.role_id = r.id
		INNER JOIN users u
			ON u.id = ur.user_id
		WHERE %s`, filter.Where)

	var count int
	err := d.QueryRowContext(ctx, stmt, filter.Args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed getting role count: %w", err)
	}

	return count, nil
}

// Roles returns one or more roles from the database. An optional filter can be
// passed to limit the results.
func Roles(ctx context.Context, d types.Querier, filter *types.Filter) ([]*Role, error) {
	query := `SELECT r.id, r.name, rp.namespaces, rp.actions, rp.target
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
		ID         uint64
		RoleName   string
		Namespaces sql.Null[string]
		Actions    sql.Null[string]
		Target     sql.Null[string]
	}
	for rows.Next() {
		r := row{}
		err := rows.Scan(&r.ID, &r.RoleName, &r.Namespaces, &r.Actions, &r.Target)
		if err != nil {
			return nil, fmt.Errorf("failed scanning role data: %w", err)
		}

		if role == nil || role.Name != r.RoleName {
			role = &Role{ID: r.ID, Name: r.RoleName}
			roles = append(roles, role)
		}

		namespaces := map[string]struct{}{}
		if r.Namespaces.Valid {
			for _, ns := range strings.Split(r.Namespaces.V, ",") {
				namespaces[ns] = struct{}{}
			}
		}

		actions := map[Action]struct{}{}
		if r.Actions.Valid {
			for _, actRune := range r.Actions.V {
				act, ok := actionMap[actRune]
				if !ok {
					return nil, fmt.Errorf("invalid action: %s", string(actRune))
				}
				actions[act] = struct{}{}
			}
		}

		perm := Permission{Namespaces: namespaces, Actions: actions}
		if r.Target.Valid {
			if r.Target.V == "*" {
				perm.Target = PermissionTarget{Resource: ResourceAny, Patterns: []string{"*"}}
			} else {
				targetPatterns := strings.Split(r.Target.V, ":")
				if len(targetPatterns) != 2 {
					return nil, fmt.Errorf("invalid target: %s", r.Target.V)
				}
				resource, err := ResourceFromString(targetPatterns[0])
				if err != nil {
					return nil, err
				}
				perm.Target = PermissionTarget{Resource: resource}
				for _, pat := range strings.Split(targetPatterns[1], ",") {
					perm.Target.Patterns = append(perm.Target.Patterns, pat)
				}
			}
		}

		role.Permissions = append(role.Permissions, perm)
	}

	return roles, nil
}

// actionMap maps short action names to their valid values.
var actionMap = map[rune]Action{
	'r': ActionRead,
	'w': ActionWrite,
	'd': ActionDelete,
	'*': ActionAny,
}

// MarshalText implements the encoding.TextMarshaler interface for Permission.
func (p Permission) MarshalText() ([]byte, error) {
	var buf bytes.Buffer

	actions := make([]string, 0, len(p.Actions))
	for action := range p.Actions {
		actions = append(actions, string(action))
	}
	slices.Sort(actions)
	for _, action := range actions {
		act, _ := utf8.DecodeRuneInString(action)
		if _, ok := actionMap[act]; !ok {
			return nil, fmt.Errorf("invalid action '%s'", action)
		}
		buf.WriteRune(act)
	}

	buf.WriteByte(':')
	namespaces := make([]string, 0, len(p.Namespaces))
	for ns := range p.Namespaces {
		namespaces = append(namespaces, ns)
	}
	slices.Sort(namespaces)
	buf.WriteString(strings.Join(namespaces, ","))

	buf.WriteByte(':')
	buf.WriteString(string(p.Target.Resource))
	buf.WriteByte(':')
	buf.WriteString(strings.Join(p.Target.Patterns, ","))

	return buf.Bytes(), nil
}

var _ encoding.TextMarshaler = &Permission{}

// UnmarshalText implements the encoding.TextUnmarshaler interface for Permission.
func (p *Permission) UnmarshalText(text []byte) error {
	parts := bytes.Split(text, []byte(":"))
	if len(parts) < 3 || len(parts) > 4 {
		return errors.New("invalid permission format: must have 3 or 4 components")
	}

	actions := map[Action]struct{}{}
	for _, char := range string(parts[0]) {
		action, ok := actionMap[char]
		if !ok {
			return fmt.Errorf("invalid action '%s'", string(char))
		}
		actions[action] = struct{}{}
	}

	namespaces := map[string]struct{}{}
	for _, ns := range strings.Split(string(parts[1]), ",") {
		namespaces[ns] = struct{}{}
	}

	targetPattern := string(parts[2])
	if len(parts) == 3 && targetPattern != "*" {
		return errors.New("invalid permission: with 3 components, the third must be a wildcard")
	}

	var (
		resource       Resource
		targetPatterns []string
	)
	if len(parts) == 4 {
		var err error
		resource, err = ResourceFromString(string(parts[2]))
		if err != nil {
			return err
		}

		if string(parts[3]) == "" {
			return errors.New("invalid permission: the fourth component must not be empty")
		}

		if resource == ResourceAny && string(parts[3]) != "*" {
			// It doesn't make sense to target specific objects if the resource
			// is a wildcard.
			return errors.New("invalid permission: the fourth component must be a wildcard if the resource is a wildcard")
		}

		for _, t := range strings.Split(string(parts[3]), ",") {
			targetPatterns = append(targetPatterns, t)
		}
	}

	p.Actions = actions
	p.Namespaces = namespaces
	p.Target = PermissionTarget{Resource: resource, Patterns: targetPatterns}

	return nil
}

var _ encoding.TextUnmarshaler = &Permission{}
