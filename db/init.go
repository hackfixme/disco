package db

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"

	"github.com/mr-tron/base58"
	"go.hackfix.me/disco/crypto"
	"go.hackfix.me/disco/db/migrator"
	"go.hackfix.me/disco/db/models"
	"go.hackfix.me/disco/db/types"
	"golang.org/x/crypto/nacl/box"
)

// Init creates the database schema and initial records.
func (d *DB) Init(
	appVersion string, serverTLSCert, serverTLSKey []byte, serverTLSSAN string,
	logger *slog.Logger,
) (localUser *models.User, err error) {
	err = migrator.RunMigrations(d, d.migrations, migrator.MigrationUp, "all", logger)
	if err != nil {
		return nil, err
	}

	dbCtx := d.NewContext()
	roles, err := createRoles(dbCtx, d)
	if err != nil {
		return nil, err
	}

	localUser, err = createLocalUser(dbCtx, d, roles["admin"])
	if err != nil {
		return nil, err
	}

	serverTLSKeyEnc, err := crypto.EncryptSymInMemory(serverTLSKey, localUser.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed encrypting TLS private key: %w", err)
	}

	_, err = d.ExecContext(dbCtx,
		`INSERT INTO _meta (version, server_tls_cert, server_tls_key_enc, server_tls_san)
		VALUES (?, ?, ?, ?)`, appVersion, serverTLSCert, serverTLSKeyEnc, serverTLSSAN)
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
				{
					Namespaces: map[string]struct{}{"*": {}},
					Actions:    map[models.Action]struct{}{models.ActionAny: {}},
					Target:     models.PermissionTarget{Resource: models.ResourceAny},
				},
			},
		},
		{
			Name: "node",
			Permissions: []models.Permission{
				{
					Namespaces: map[string]struct{}{"*": {}},
					Actions:    map[models.Action]struct{}{models.ActionRead: {}},
					Target: models.PermissionTarget{
						Resource: models.ResourceStore,
						Patterns: []string{"*"},
					},
				},
			},
		},
		{
			Name: "user",
			Permissions: []models.Permission{
				{
					Namespaces: map[string]struct{}{"*": {}},
					Actions:    map[models.Action]struct{}{models.ActionAny: {}},
					Target: models.PermissionTarget{
						Resource: models.ResourceStore,
						Patterns: []string{"*"},
					},
				},
			},
		},
	}
	rolesMap := map[string]*models.Role{}
	for _, role := range roles {
		if err := role.Save(ctx, d, false); err != nil {
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

	pkHash := crypto.Hash("", pubKey[:])
	user := &models.User{
		Name:       base58.Encode(pkHash[:8]),
		Type:       models.UserTypeLocal,
		PublicKey:  pubKey,
		PrivateKey: privKey,
		Roles:      []*models.Role{role},
	}
	if err := user.Save(ctx, d, false); err != nil {
		return nil, err
	}

	return user, nil
}
