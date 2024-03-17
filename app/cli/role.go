package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
)

// The Role command manages roles.
type Role struct {
	Add struct {
		Name        string              `arg:"" help:"The unique name of the role."`
		Permissions []models.Permission `arg:"" help:"Permissions to assign to the role. \n Permission format: \"<actions>:<namespaces>:<resource>:<target>\" \n Example: \"rwd:dev,prod:store:myapp/*\""`
	} `kong:"cmd,help='Add a new role.'"`
	Rm struct {
		Name  string `arg:"" help:"The unique name of the role."`
		Force bool   `help:"Remove role even if it's assigned to existing users."`
	} `kong:"cmd,help='Remove a role.'"`
	Modify struct {
		Name        string              `arg:"" help:"The unique name of the role."`
		Permissions []models.Permission `arg:"" help:"Permissions to assign to the role. \n Permission format: \"<actions>:<namespaces>:<resource>:<target>\" \n Example: \"rwd:dev,prod:store:myapp/*\""`
	} `kong:"cmd,help='Change the settings of a role.'"`
	Ls struct {
	} `kong:"cmd,help='List roles.'"`
}

// Run the role command.
func (c *Role) Run(kctx *kong.Context, appCtx *actx.Context) error {
	dbCtx := appCtx.DB.NewContext()

	switch kctx.Args[1] {
	case "add":
		role := &models.Role{Name: c.Add.Name, Permissions: c.Add.Permissions}
		if err := role.Save(dbCtx, appCtx.DB); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed adding role '%s'", c.Add.Name), err, "")
		}
	case "rm":
		role := &models.Role{Name: c.Rm.Name}
		err := role.Delete(dbCtx, appCtx.DB, c.Rm.Force)

		var errRef *types.ErrReference
		if errors.As(err, &errRef) {
			return aerrors.NewRuntimeError(err.Error(), errRef.Cause,
				"remove all assignments first or pass --force to delete anyway")
		}

		return err
	case "modify":
	case "ls":
		roles, err := models.Roles(dbCtx, appCtx.DB, nil)
		if err != nil {
			return aerrors.NewRuntimeError("failed listing roles", err, "")
		}

		data := [][]string{}
		for _, role := range roles {
			for i, perm := range role.Permissions {
				namespaces := make([]string, 0, len(perm.Namespaces))
				for ns := range perm.Namespaces {
					namespaces = append(namespaces, ns)
				}
				slices.Sort(namespaces)
				nsJoined := strings.Join(namespaces, ",")

				actions := make([]string, 0, len(perm.Actions))
				for action := range perm.Actions {
					actions = append(actions, string(action))
				}
				slices.Sort(actions)
				actsJoined := strings.Join(actions, ",")

				row := []string{role.Name, nsJoined, actsJoined, perm.TargetPattern}
				if i > 0 {
					row[0] = ""
				}
				data = append(data, row)
			}
		}

		header := []string{"Name", "Namespaces", "Actions", "Target"}
		newTable(header, data, appCtx.Stdout).Render()
	}

	return nil
}
