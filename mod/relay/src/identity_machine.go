package relay

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/cryptopunkscc/astrald/auth/id"
	"github.com/cryptopunkscc/astrald/cslq"
	"github.com/cryptopunkscc/astrald/mod/data"
	"github.com/cryptopunkscc/astrald/mod/relay"
)

// IdentityMachine renders the final identity by applying certificates to the initial identity
type IdentityMachine struct {
	identity id.Identity
}

// NewIdentityMachine returns a new instance of an IdentityMachine with the provided identity as its initial state
func NewIdentityMachine(identity id.Identity) *IdentityMachine {
	return &IdentityMachine{identity: identity}
}

// Apply applies a certificate to the current identity
func (m *IdentityMachine) Apply(certBytes []byte) error {
	var r = bytes.NewReader(certBytes)
	var dataType data.ADC0Header

	err := cslq.Decode(r, "v", &dataType)
	if err != nil {
		return err
	}

	switch dataType {
	case relay.RelayCertType:
		var cert relay.RelayCert
		if err := cslq.Decode(r, "v", &cert); err != nil {
			return err
		}

		if !cert.RelayID.IsEqual(m.identity) {
			fmt.Println(cert.RelayID.String(), m.identity.String())
			return errors.New("relay identity mismatch")
		}

		if err = cert.Validate(); err != nil {
			return fmt.Errorf("invalid certificate: %w", err)
		}

		m.identity = cert.TargetID

		return nil
	}

	return fmt.Errorf("unknown certificate type: %s", dataType)
}

// Identity returns the current identity
func (m *IdentityMachine) Identity() id.Identity {
	return m.identity
}
