package gateway

import (
	"context"
	"errors"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/cslq"
	"github.com/cryptopunkscc/astrald/hub"
	"github.com/cryptopunkscc/astrald/infra/gw"
	"github.com/cryptopunkscc/astrald/node"
	"github.com/cryptopunkscc/astrald/node/link"
	"github.com/cryptopunkscc/astrald/streams"
	"log"
)

const queryConnect = "connect"

type Gateway struct {
	node *node.Node
}

func (mod *Gateway) Run(ctx context.Context) error {
	port, err := mod.node.Ports.RegisterContext(ctx, gw.PortName)
	if err != nil {
		return err
	}

	for req := range port.Queries() {
		conn, err := req.Accept()
		if err != nil {
			continue
		}

		go func() {
			if err := mod.handleConn(ctx, conn); err != nil {
				cslq.Encode(conn, "c", false)
				log.Println("[gateway] error serving client:", err)
			}
			defer conn.Close()
		}()
	}

	return nil
}

func (mod *Gateway) handleConn(ctx context.Context, conn *hub.Conn) error {
	c := cslq.NewEndec(conn)

	var cookie string

	err := c.Decode("[c]c", &cookie)
	if err != nil {
		return err
	}

	nodeID, err := id.ParsePublicKeyHex(cookie)
	if err != nil {
		return err
	}

	var lnk *link.Link

	if _, err = mod.node.Contacts.Find(nodeID); err == nil {
		lnk, err = mod.node.Peers.Link(ctx, nodeID)
		if err != nil {
			return err
		}
	} else {
		peer := mod.node.Peers.Pool.Peer(nodeID)
		if peer == nil {
			return errors.New("node unavailable")
		}

		lnk = peer.PreferredLink()
		if lnk == nil {
			return errors.New("node unavailable")
		}
	}

	out, err := lnk.Query(ctx, queryConnect)
	if err != nil {
		return err
	}

	c.Encode("c", true)

	return streams.Join(conn, out)
}
