package user

import (
	"context"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/mod/admin"
	"github.com/cryptopunkscc/astrald/mod/user"
	"github.com/cryptopunkscc/astrald/node/modules"
)

func (mod *Module) Prepare(ctx context.Context) error {
	if adm, err := modules.Load[admin.Module](mod.node, admin.ModuleName); err == nil {
		adm.AddCommand(user.ModuleName, NewAdmin(mod))
	}

	for _, u := range mod.config.Identities {
		userID, err := mod.node.Resolver().Resolve(u)
		if err != nil {
			mod.log.Error("config: cannot resolve identity '%v': %v", u, err)
			continue
		}

		mod.db.Create(&dbIdentity{Identity: userID.PublicKeyHex()})
	}

	var rows []dbIdentity

	var tx = mod.db.Find(&rows)
	if tx.Error != nil {
		return tx.Error
	}

	for _, row := range rows {
		userID, err := id.ParsePublicKeyHex(row.Identity)
		if err != nil {
			mod.log.Error("db: invalid identity '%v': %v", row.Identity, err)
			continue
		}

		err = mod.addIdentity(userID)
		if err != nil {
			mod.log.Error("cannot add identity %v: %v", userID, err)
			continue
		}

	}

	// look for user profiles in discovered services
	go mod.discoverUsers(ctx)

	return nil
}
