package cli

import (
	"fmt"
	"strings"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
)

// The User command manages users.
type User struct {
	Add struct {
		Name  string   `arg:"" help:"The unique name of the user."`
		Roles []string `help:"Names of roles to assign to this user."`
	} `kong:"cmd,help='Add a new user.'"`
	Rm struct {
		Name string `arg:"" help:"The unique name of the user."`
	} `kong:"cmd,help='Remove a user.'"`
	Update struct {
		Name  string   `arg:"" help:"The unique name of the user."`
		Roles []string `help:"Names of roles to assign to this user. \n Any existing roles will be removed and replaced with this set."`
	} `kong:"cmd,help='Update the configuration of a user.'"`
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
		var roles []*models.Role
		for _, roleName := range c.Add.Roles {
			role := &models.Role{Name: roleName}
			if err := role.Load(dbCtx, appCtx.DB); err != nil {
				return err
			}
			roles = append(roles, role)
		}

		if len(roles) == 0 {
			appCtx.Logger.Warn(fmt.Sprintf(
				"user '%s' has no associated roles and won't be able to "+
					"access any resources", c.Add.Name))
		}

		user := &models.User{Name: c.Add.Name, Roles: roles}
		if err := user.Save(dbCtx, appCtx.DB, false); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed adding user '%s'", c.Add.Name), err, "")
		}
	case "rm":
		user := &models.User{Name: c.Rm.Name}
		err := user.Delete(dbCtx, appCtx.DB)
		if err != nil {
			return err
		}
	case "update":
		var roles []*models.Role
		for _, roleName := range c.Update.Roles {
			role := &models.Role{Name: roleName}
			if err := role.Load(dbCtx, appCtx.DB); err != nil {
				return err
			}
			roles = append(roles, role)
		}

		if len(roles) == 0 {
			appCtx.Logger.Warn(fmt.Sprintf(
				"user '%s' has no associated roles and won't be able to "+
					"access any resources", c.Update.Name))
		}

		user := &models.User{Name: c.Update.Name, Roles: roles}
		if err := user.Save(dbCtx, appCtx.DB, true); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed adding user '%s'", c.Update.Name), err, "")
		}
	case "invite":
	case "ls":
		users, err := models.Users(dbCtx, appCtx.DB, nil)
		if err != nil {
			return aerrors.NewRuntimeError("failed listing users", err, "")
		}

		data := make([][]string, len(users))
		for i, user := range users {
			roles := make([]string, len(user.Roles))
			for ri, role := range user.Roles {
				roles[ri] = role.Name
			}
			data[i] = []string{user.Name, strings.Join(roles, ",")}
		}

		header := []string{"Name", "Roles"}
		newTable(header, data, appCtx.Stdout).Render()
	}

	return nil
}
