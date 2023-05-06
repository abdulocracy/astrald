package nat

import (
	"context"
	"errors"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/cslq"
	"github.com/cryptopunkscc/astrald/node/infra/drivers/inet"
	"github.com/cryptopunkscc/astrald/node/link"
	"github.com/cryptopunkscc/astrald/node/network"
	"time"
)

const pingCount = 3
const dialRetries = 10
const dialInterval = 1 * time.Second
const clockSyncThreshold = 100 * time.Millisecond

func (mod *Module) query(ctx context.Context, remoteID id.Identity) error {
	// connect to nat service
	conn, err := mod.node.Query(ctx, remoteID, portName)
	if err != nil {
		return err
	}
	defer conn.Close()
	var c = cslq.NewEndec(conn)

	// initialize nat session
	if err := c.Encode("[c]c", cmdInit); err != nil {
		return err
	}
	if err := c.Decode("x00"); err != nil {
		return err
	}

	triesLeft := dialRetries

	for {
		log.Logv(1, "reties left:", triesLeft)
		// exchange addresses
		var buf []byte
		if err := c.Encode("[c]c[c]c", cmdAddr, mod.mapping.extAddr.Pack()); err != nil {
			return err
		}
		if err := c.Decode("[c]c", &buf); err != nil {
			return err
		}
		remoteAddr, err := inet.Unpack(buf)
		if err != nil {
			return err
		}

		// measure latency
		var pingStart = time.Now()
		for i := 0; i < pingCount; i++ {
			if err := c.Encode("[c]c", cmdPing); err != nil {
				return err
			}
			if err := c.Decode("x00"); err != nil {
				return err
			}
		}
		var ping = time.Now().Sub(pingStart) / pingCount

		log.Log("ping %s avg %s%dms%s",
			remoteAddr,
			log.EmColor(),
			ping.Milliseconds(),
			log.Reset(),
		)

		// signal
		if ping < clockSyncThreshold {
			if err := c.Encode("[c]c", cmdGo); err != nil {
				return err
			}

			// wait half the round-trip
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-time.After(ping / 2):
			}
		} else {
			log.Logv(1, "using clock-based signalling due to high latency")

			startAt := time.Now().Add((2 * ping) + (2 * time.Second)).Unix()

			if err = c.Encode("[c]cq", cmdTime, startAt); err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-time.After(time.Until(time.Unix(startAt, 0))):
			}
		}

		// dial
		authed, err := mod.makeLink(ctx, remoteAddr, remoteID)
		if err == nil {
			l := link.New(authed)
			l.SetPriority(network.NetworkPriority(l.Network()))
			return mod.node.Network().AddLink(l)
		}

		triesLeft--
		if triesLeft == 0 {
			return errors.New("traversal failed")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-time.After(dialInterval):
		}
	}
}
