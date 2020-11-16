package systemd

import (
	"github.com/miekg/vks"
	"github.com/miekg/vks/pkg/debian"
	"github.com/miekg/vks/pkg/manager"
	"github.com/virtual-kubelet/node-cli/provider"
)

type P struct {
	p vks.PackageManager
	m *manager.UnitManager
}

func NewProvider() (*P, error) {
	m, err := manager.New("/tmp/bla", false)
	if err != nil {
		return nil, err
	}
	p := &P{m: m}
	// figure out the os from /etc/os-release
	switch id() {
	case "debian", "ubuntu":
		p.p = new(debian.Debian)
	}

	return p, nil
}

var _ provider.Provider = new(P)
