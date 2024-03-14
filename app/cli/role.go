package cli

import (
	"fmt"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
)

// The Role command manages roles.
type Role struct {
	Add struct {
		Name string `arg:"" help:"The unique name of the role."`
	} `kong:"cmd,help='Add a new role.'"`
	Rm struct {
		Name string `arg:"" help:"The unique name of the role."`
	} `kong:"cmd,help='Remove a role.'"`
	Modify struct {
		Name string `arg:"" help:"The unique name of the role."`
	} `kong:"cmd,help='Change the settings of a role.'"`
	Ls struct {
	} `kong:"cmd,help='List roles.'"`
}

// Run the role command.
func (c *Role) Run(kctx *kong.Context, appCtx *actx.Context) error {
	dbCtx := appCtx.DB.NewContext()

	switch kctx.Args[1] {
	case "add":
		// TODO: Add permissions
		role := &models.Role{Name: c.Add.Name}
		if err := role.Save(dbCtx, appCtx.DB); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed adding role '%s'", c.Add.Name), err, "")
		}
	case "rm":
		role := &models.Role{Name: c.Rm.Name}
		if err := role.Delete(dbCtx, appCtx.DB); err != nil {
			return err
		}
	case "modify":
	case "ls":
		roles, err := models.Roles(dbCtx, appCtx.DB, nil)
		if err != nil {
			return aerrors.NewRuntimeError("failed listing roles", err, "")
		}

		for _, role := range roles {
			fmt.Fprintln(appCtx.Stdout, role.Name)
		}
	}

	return nil
}
