package db

import (
	"context"
	"crypto/rand"
	"fmt"

	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
	"golang.org/x/crypto/nacl/box"
)

// Init creates the database schema and initial records.
func (d *DB) Init(appVersion string) (localUser *models.User, err error) {
	err = migrator.RunMigrations(d, d.migrations, migrator.MigrationUp, "all")
	if err != nil {
		return nil, err
	}

	dbCtx := d.NewContext()
	_, err = d.ExecContext(dbCtx,
		`INSERT INTO _meta (version) VALUES (?)`, appVersion)
	if err != nil {
		return nil, err
	}

	roles, err := createRoles(dbCtx, d)
	if err != nil {
		return nil, err
	}

	localUser, err = createLocalUser(dbCtx, d, roles["admin"])
	if err != nil {
		return nil, err
	}

	return localUser, nil
}

func createRoles(ctx context.Context, d types.Querier) (map[string]*models.Role, error) {
	roles := []*models.Role{
		{
			Name: "admin",
			Permissions: []models.Permission{
				{ActionPattern: "*", TargetPattern: "*"},
			},
		},
		{
			Name: "node",
			Permissions: []models.Permission{
				{ActionPattern: "read", TargetPattern: "store:*"},
			},
		},
		{
			Name: "user",
			Permissions: []models.Permission{
				{ActionPattern: "read", TargetPattern: "store:*"},
				{ActionPattern: "write", TargetPattern: "store:*"},
				{ActionPattern: "create", TargetPattern: "store:*"},
				{ActionPattern: "delete", TargetPattern: "store:*"},
			},
		},
	}
	rolesMap := map[string]*models.Role{}
	for _, role := range roles {
		if err := role.Save(ctx, d); err != nil {
			return nil, err
		}
		rolesMap[role.Name] = role
	}

	return rolesMap, nil
}

func createLocalUser(ctx context.Context, d types.Querier, role *models.Role) (*models.User, error) {
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed generating encryption key pair: %w", err)
	}

	user := &models.User{
		Name:       "local",
		PublicKey:  pubKey,
		PrivateKey: privKey,
		Roles:      []*models.Role{role},
	}
	if err := user.Save(ctx, d); err != nil {
		return nil, err
	}

	return user, nil
}
