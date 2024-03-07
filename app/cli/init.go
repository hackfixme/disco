package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/nacl/box"

	actx "go.hackfix.me/disco/app/context"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/queries"
	"go.hackfix.me/disco/db/store/sqlite"
)

// The Init command initializes the Disco data stores and generates a new
// encryption key.
type Init struct{}

// Run the init command.
func (c *Init) Run(appCtx *actx.Context) error {
	dbMigrations := appCtx.DB.Migrations()
	err := migrator.RunMigrations(appCtx.DB, dbMigrations, migrator.MigrationUp, "all")
	if err != nil {
		return err
	}

	version, err := queries.Version(appCtx.Ctx, appCtx.DB)
	if version.Valid {
		// TODO: Add --force option?
		return fmt.Errorf("Disco is already initialized with version %s", version.V)
	}

	_, err = appCtx.DB.ExecContext(appCtx.Ctx,
		`INSERT INTO _meta (version) VALUES (?)`,
		appCtx.Version)
	if err != nil {
		return err
	}

	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed generating encryption keypair: %w", err)
	}
	privKeyHex := hex.EncodeToString(privKey[:])

	if sqlStore, ok := appCtx.Store.(*sqlite.Store); ok {
		storeMigrations := sqlStore.Migrations()
		err = migrator.RunMigrations(sqlStore, storeMigrations, migrator.MigrationUp, "all")
		if err != nil {
			return err
		}

		pubKeyHex := hex.EncodeToString(pubKey[:])
		privKeyHash := crypto.Hash("encryption key hash", privKey[:])
		privKeyHashHex := hex.EncodeToString(privKeyHash)
		_, err = sqlStore.ExecContext(appCtx.Ctx,
			`INSERT INTO _meta (version, public_key, private_key_hash)
			VALUES (?, ?, ?)`,
			appCtx.Version, pubKeyHex, privKeyHashHex)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(appCtx.Stdout, `New encryption key: %s

Make sure to store this key in a secure location, such as a password manager.

You won't be able to access the data on this node without it!
	`, privKeyHex)

	return nil
}
