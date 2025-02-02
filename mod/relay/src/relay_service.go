package relay

import (
	"context"
	"github.com/cryptopunkscc/astrald/mod/relay"
	"github.com/cryptopunkscc/astrald/mod/relay/proto"
	"github.com/cryptopunkscc/astrald/net"
	"time"
)

var _ net.Router = &RelayService{}

type RelayService struct {
	*Module
}

func (srv *RelayService) Run(ctx context.Context) error {
	err := srv.node.LocalRouter().AddRoute(relay.RelayServiceName, srv)
	if err != nil {
		return err
	}
	defer srv.node.LocalRouter().RemoveRoute(relay.RelayServiceName)

	<-ctx.Done()

	return nil
}

func (srv *RelayService) RouteQuery(ctx context.Context, query net.Query, caller net.SecureWriteCloser, hints net.Hints) (net.SecureWriteCloser, error) {
	return net.Accept(query, caller, func(conn net.SecureConn) {
		if err := srv.serve(ctx, conn); err != nil {
			srv.log.Errorv(2, "error serving %s: %s", query.Caller(), err)
		}
	})
}

func (srv *RelayService) serve(ctx context.Context, conn net.SecureConn) error {
	defer conn.Close()

	var err error
	var callerIM = NewIdentityMachine(conn.RemoteIdentity())
	var session = proto.New(conn)

	// get query params
	var params proto.QueryParams
	if err = session.Decode(&params); err != nil {
		return err
	}

	// apply caller certificate
	if len(params.Cert) > 0 {
		if err = callerIM.Apply(params.Cert); err != nil {
			session.EncodeErr(proto.ErrCertificateRejected)
			return err
		}
	}

	var response proto.QueryResponse

	// attach target certificate if necessary
	if !params.Target.IsEqual(srv.node.Identity()) {
		// get an inbound relay certificate
		response.Cert, err = srv.ReadCert(&relay.FindOpts{
			TargetID:  params.Target,
			RelayID:   srv.node.Identity(),
			Direction: relay.Inbound,
		})
		if err != nil {
			srv.log.Errorv(1, "error getting target certificate: %v", err)
			_ = session.EncodeErr(proto.ErrRouteNotFound)
			return err
		}
	}

	// create a proxy service
	redirectCtx, _ := context.WithTimeout(ctx, time.Minute)
	var realQuery = net.NewQueryNonce(callerIM.identity, params.Target, params.Query, net.Nonce(params.Nonce))

	redirect, err := NewRedirect(redirectCtx, realQuery, conn.RemoteIdentity(), srv.node)
	if err != nil {
		session.EncodeErr(proto.ErrInternalError)
		return err
	}

	response.ProxyService = redirect.ServiceName

	// send response
	_ = session.EncodeErr(nil)
	return session.Encode(response)
}
