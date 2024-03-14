package cli

import (
	"fmt"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
)

// The User command manages users.
type User struct {
	Add struct {
		Name  string   `arg:"" help:"The unique name of the user."`
		Roles []string `help:"The roles to associate with this user."`
	} `kong:"cmd,help='Add a new user.'"`
	Rm struct {
		Name string `arg:"" help:"The unique name of the user."`
	} `kong:"cmd,help='Remove a user.'"`
	Modify struct {
		Name  string   `arg:"" help:"The unique name of the user."`
		Roles []string `help:"The roles to associate with this user."`
	} `kong:"cmd,help='Change the settings of a user.'"`
	Invite struct {
		Name string `arg:"" help:"The unique name of the user."`
	} `kong:"cmd,help='Create an invitation token for a user.'"`
	Ls struct {
	} `kong:"cmd,help='List users.'"`
}

// Run the user command.
func (c *User) Run(kctx *kong.Context, appCtx *actx.Context) error {
	dbCtx := appCtx.DB.NewContext()

	switch kctx.Args[1] {
	case "add":
		// TODO: Hook up roles.
		user := &models.User{Name: c.Add.Name}
		if err := user.Save(dbCtx, appCtx.DB); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed adding user '%s'", c.Add.Name), err, "")
		}
	case "rm":
		user := &models.User{Name: c.Rm.Name}
		err := user.Delete(dbCtx, appCtx.DB)
		if err != nil {
			return err
		}
	case "modify":
	case "invite":
	case "ls":
		users, err := models.Users(dbCtx, appCtx.DB, nil)
		if err != nil {
			return aerrors.NewRuntimeError("failed listing users", err, "")
		}

		for _, user := range users {
			fmt.Fprintln(appCtx.Stdout, user.Name)
		}
	}

	return nil
}
