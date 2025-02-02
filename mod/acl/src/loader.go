package acl

import (
	"github.com/cryptopunkscc/astrald/log"
	"github.com/cryptopunkscc/astrald/mod/acl"
	"github.com/cryptopunkscc/astrald/node/assets"
	"github.com/cryptopunkscc/astrald/node/modules"
)

type Loader struct{}

func (Loader) Load(node modules.Node, assets assets.Assets, log *log.Logger) (modules.Module, error) {
	var err error
	var mod = &Module{
		node:   node,
		log:    log,
		assets: assets,
	}

	_ = assets.LoadYAML(acl.ModuleName, &mod.config)

	mod.db, err = assets.OpenDB(acl.ModuleName)
	if err != nil {
		return nil, err
	}
	if err = mod.db.AutoMigrate(&dbPerm{}); err != nil {
		return nil, err
	}

	return mod, nil
}

func init() {
	if err := modules.RegisterModule(acl.ModuleName, Loader{}); err != nil {
		panic(err)
	}
}
