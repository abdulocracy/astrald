package sdp

import (
	"context"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/cslq"
	"github.com/cryptopunkscc/astrald/log"
	"github.com/cryptopunkscc/astrald/mod/admin/api"
	"github.com/cryptopunkscc/astrald/mod/router/api"
	. "github.com/cryptopunkscc/astrald/mod/sdp/api"
	"github.com/cryptopunkscc/astrald/mod/sdp/proto"
	"github.com/cryptopunkscc/astrald/net"
	"github.com/cryptopunkscc/astrald/node"
	"github.com/cryptopunkscc/astrald/node/assets"
	"github.com/cryptopunkscc/astrald/node/events"
	"github.com/cryptopunkscc/astrald/tasks"
	"sync"
)

type Module struct {
	node      node.Node
	events    events.Queue
	config    Config
	assets    assets.Store
	log       *log.Logger
	sources   map[Source]struct{}
	sourcesMu sync.Mutex
	cache     map[string][]ServiceEntry
	router    router.API
	cacheMu   sync.Mutex
	ctx       context.Context
}

func (m *Module) Run(ctx context.Context) error {
	m.ctx = ctx

	m.router, _ = m.node.Modules().Find("router").(router.API)

	// inject admin command
	if adm, _ := m.node.Modules().Find("admin").(admin.API); adm != nil {
		adm.AddCommand(ModuleName, NewAdmin(m))
	}

	return tasks.Group(
		&DiscoveryService{Module: m},
		&EventHandler{Module: m},
	).Run(ctx)
}

func (m *Module) AddSource(source Source) {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()

	m.sources[source] = struct{}{}
}

func (m *Module) RemoveSource(source Source) {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()

	_, found := m.sources[source]
	if !found {
		return
	}

	delete(m.sources, source)
}

func (m *Module) QueryLocal(ctx context.Context, caller id.Identity, origin string) ([]ServiceEntry, error) {
	var list = make([]ServiceEntry, 0)

	var wg sync.WaitGroup

	for source := range m.sources {
		source := source

		wg.Add(1)
		go func() {
			defer wg.Done()

			slist, err := source.Discover(ctx, caller, origin)
			if err != nil {
				return
			}

			list = append(list, slist...)
		}()
	}

	wg.Wait()

	return list, nil
}

func (m *Module) QueryRemoteAs(ctx context.Context, remoteID id.Identity, callerID id.Identity) ([]ServiceEntry, error) {
	if callerID.IsZero() {
		callerID = m.node.Identity()
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if callerID.PrivateKey() == nil {
		keyStore, err := m.assets.KeyStore()
		if err != nil {
			return nil, err
		}

		callerID, err = keyStore.Find(callerID)
		if err != nil {
			return nil, err
		}
	}

	q, err := net.Route(ctx,
		m.node.Router(),
		net.NewQuery(callerID, remoteID, DiscoverServiceName),
	)
	if err != nil {
		return nil, err
	}

	var list = make([]ServiceEntry, 0)

	go func() {
		<-ctx.Done()
		q.Close()
	}()
	for err == nil {
		err = cslq.Invoke(q, func(msg proto.ServiceEntry) error {
			list = append(list, ServiceEntry(msg))
			if !msg.Identity.IsEqual(remoteID) {
				if m.router != nil {
					m.router.SetRouter(msg.Identity, remoteID)
				}
			}
			return nil
		})
	}

	return list, nil
}

func (m *Module) setCache(identity id.Identity, list []ServiceEntry) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	m.cache[identity.String()] = list
}

func (m *Module) getCache(identity id.Identity) []ServiceEntry {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	return m.cache[identity.String()]
}
