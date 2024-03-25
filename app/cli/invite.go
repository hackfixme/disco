package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/alecthomas/kong"

	actx "go.hackfix.me/disco/app/context"
	aerrors "go.hackfix.me/disco/app/errors"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
)

// The Invite command manages invitations for remote users.
type Invite struct {
	User struct {
		Name string        `arg:"" help:"The name of the user to invite."`
		TTL  time.Duration `default:"1h" help:"Time duration the invite is valid for."`
	} `kong:"cmd,help='Create a new invitation token for an existing user to access this Disco node remotely.'"`
	Ls struct {
	} `kong:"cmd,help='List invites.'"`
	Rm struct {
		UUID string `arg:"" help:"The unique invite ID. A short prefix can be specified as long as it's unique."`
	} `kong:"cmd,help='Delete an unredeemed invite.'"`
	Update struct {
		UUID string         `arg:"" help:"The unique invite ID."`
		TTL  *time.Duration `help:"Time duration the invite is valid for."`
	} `kong:"cmd,help='Update an invite to extend its validity period.'"`
}

// Run the invite command.
func (c *Invite) Run(kctx *kong.Context, appCtx *actx.Context) error {
	dbCtx := appCtx.DB.NewContext()

	switch kctx.Args[1] {
	case "user":
		user := &models.User{Name: c.User.Name}
		if err := user.Load(dbCtx, appCtx.DB); err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed loading user '%s'", c.User.Name), err, "")
		}
		inv, err := models.NewInvite(user, c.User.TTL, appCtx.UUIDGen, appCtx.User.PrivateKey)
		if err != nil {
			return aerrors.NewRuntimeError(
				fmt.Sprintf("failed creating invite for user '%s'", c.User.Name), err, "")
		}

		if err := inv.Save(dbCtx, appCtx.DB, false); err != nil {
			return aerrors.NewRuntimeError(
				"failed saving invite to the database", err, "")
		}

		timeLeft := inv.Expires.Sub(time.Now().UTC())
		expFmt := fmt.Sprintf("%s (%s)",
			inv.Expires.Local().Format(time.DateTime),
			timeLeft.Round(time.Second))
		fmt.Fprintf(appCtx.Stdout, `Token: %s
Expires: %s
	`, inv.Token, expFmt)

	case "ls":
		now := time.Now().UTC()
		invites, err := models.Invites(dbCtx, appCtx.DB,
			types.NewFilter("inv.expires > ?", []any{now}))
		if err != nil {
			return aerrors.NewRuntimeError("failed listing invites", err, "")
		}

		data := make([][]string, len(invites))
		for i, inv := range invites {
			timeLeft := inv.Expires.Sub(now)
			expFmt := fmt.Sprintf("%s (%s)",
				inv.Expires.Local().Format(time.DateTime),
				timeLeft.Round(time.Second))
			data[i] = []string{inv.UUID, inv.User.Name, inv.Token, expFmt}
		}

		if len(data) > 0 {
			header := []string{"UUID", "User", "Token", "Expiration"}
			newTable(header, data, appCtx.Stdout).Render()
		}

	case "rm":
		inv := &models.Invite{UUID: c.Rm.UUID}
		if err := inv.Delete(dbCtx, appCtx.DB); err != nil {
			return err
		}

	case "update":
		if c.Update.TTL == nil {
			return errors.New("must set a valid TTL")
		}

		newExpiration := time.Now().UTC().Add(*c.Update.TTL)
		inv := &models.Invite{UUID: c.Update.UUID, Expires: newExpiration}
		if err := inv.Save(dbCtx, appCtx.DB, true); err != nil {
			return err
		}
	}

	return nil
}
