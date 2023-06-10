package discovery

import (
	"context"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/cslq"
	"github.com/cryptopunkscc/astrald/mod/discovery/proto"
	"github.com/cryptopunkscc/astrald/node/services"
)

type Source interface {
	Discover(ctx context.Context, caller id.Identity, origin string) ([]ServiceEntry, error)
}

var _ Source = &ServiceSource{}

type ServiceSource struct {
	services services.Services
	service  string
}

func (src *ServiceSource) String() string {
	return src.service
}

func (src *ServiceSource) Discover(ctx context.Context, caller id.Identity, origin string) ([]ServiceEntry, error) {
	conn, err := src.services.Query(ctx, caller, src.service, nil)
	if err != nil {
		return nil, err
	}

	var list = make([]ServiceEntry, 0)

	for err == nil {
		err = cslq.Invoke(conn, func(msg proto.ServiceEntry) error {
			msg.Identity = conn.RemoteIdentity()
			list = append(list, ServiceEntry(msg))
			return nil
		})
	}

	return list, nil
}
