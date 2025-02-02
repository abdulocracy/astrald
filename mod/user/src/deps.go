package user

import (
	"github.com/cryptopunkscc/astrald/mod/admin"
	"github.com/cryptopunkscc/astrald/mod/apphost"
	"github.com/cryptopunkscc/astrald/mod/data"
	"github.com/cryptopunkscc/astrald/mod/discovery"
	"github.com/cryptopunkscc/astrald/mod/keys"
	"github.com/cryptopunkscc/astrald/mod/relay"
	"github.com/cryptopunkscc/astrald/mod/storage"
	"github.com/cryptopunkscc/astrald/node/modules"
)

func (mod *Module) LoadDependencies() error {
	var err error

	// load required dependencies
	mod.storage, err = modules.Load[storage.Module](mod.node, storage.ModuleName)
	if err != nil {
		return err
	}

	mod.relay, err = modules.Load[relay.Module](mod.node, relay.ModuleName)
	if err != nil {
		return err
	}

	// load optional dependencies
	mod.data, _ = modules.Load[data.Module](mod.node, data.ModuleName)
	mod.sdp, _ = modules.Load[discovery.Module](mod.node, discovery.ModuleName)
	mod.keys, _ = modules.Load[keys.Module](mod.node, keys.ModuleName)
	mod.admin, _ = modules.Load[admin.Module](mod.node, admin.ModuleName)
	mod.apphost, _ = modules.Load[apphost.Module](mod.node, apphost.ModuleName)

	if mod.sdp != nil {
		mod.sdp.AddServiceDiscoverer(mod)
		mod.sdp.AddDataDiscoverer(mod)
	}

	if mod.storage != nil {
		mod.storage.Access().AddAccessVerifier(mod)
	}

	return nil
}
