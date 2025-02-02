package data

import (
	"github.com/cryptopunkscc/astrald/log"
	"github.com/cryptopunkscc/astrald/mod/data"
	"github.com/cryptopunkscc/astrald/node/assets"
	"github.com/cryptopunkscc/astrald/node/modules"
)

type Loader struct{}

func (Loader) Load(node modules.Node, assets assets.Assets, log *log.Logger) (modules.Module, error) {
	mod := &Module{
		node:  node,
		log:   log,
		ready: make(chan struct{}),
	}

	var err error

	_ = assets.LoadYAML(data.ModuleName, &mod.config)

	mod.events.SetParent(node.Events())

	mod.db, err = assets.OpenDB(data.ModuleName)
	if err != nil {
		return nil, err
	}

	if err := mod.db.AutoMigrate(&dbDataType{}); err != nil {
		return nil, err
	}
	if err := mod.db.AutoMigrate(&dbLabel{}); err != nil {
		return nil, err
	}

	return mod, nil
}

func init() {
	if err := modules.RegisterModule(data.ModuleName, Loader{}); err != nil {
		panic(err)
	}
}
