package rdma

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
)

const (
	RdmaSysModeExclusive = "exclusive"
	RdmaSysModeShared    = "shared"
)

func NewRdmaManager() Manager {
	return &rdmaManagerNetlink{rdmaOps: newRdmaBasicOps()}
}

type Manager interface {
	// Move RDMA device from current network namespace to network namespace
	MoveRdmaDevToNs(rdmaDev string, netNs ns.NetNS) error
	// Get RDMA devices associated with the given PCI device in D:B:D.f format e.g 0000:04:00.0
	GetRdmaDevsForPciDev(pciDev string) []string
	// Get RDMA devices associated with the given auxiliary device. For example, for input mlx5_core.sf.4, returns
	// [mlx5_0,mlx5_10,..]
	GetRdmaDevsForAuxDev(auxDev string) []string
	// Get RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
	GetSystemRdmaMode() (string, error)
	// Set RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
	SetSystemRdmaMode(mode string) error
}

type rdmaManagerNetlink struct {
	rdmaOps BasicOps
}

// Move RDMA device to network namespace
func (rmn *rdmaManagerNetlink) MoveRdmaDevToNs(rdmaDev string, netNs ns.NetNS) error {
	rdmaLink, err := rmn.rdmaOps.RdmaLinkByName(rdmaDev)
	if err != nil {
		return fmt.Errorf("cannot find RDMA link from name: %s", rdmaDev)
	}
	err = rmn.rdmaOps.RdmaLinkSetNsFd(rdmaLink, uint32(netNs.Fd()))
	if err != nil {
		return fmt.Errorf("failed to move RDMA dev %s to namespace. %v", rdmaDev, err)
	}
	return nil
}

// Get RDMA device associated with the given PCI device in D:B:D.f format e.g 0000:04:00.1
func (rmn *rdmaManagerNetlink) GetRdmaDevsForPciDev(pciDev string) []string {
	return rmn.rdmaOps.GetRdmaDevicesForPcidev(pciDev)
}

// Get RDMA devices associated with the given auxiliary device. For example, for input mlx5_core.sf.4, returns
// [mlx5_0,mlx5_10,..]
func (rmn *rdmaManagerNetlink) GetRdmaDevsForAuxDev(auxDev string) []string {
	return rmn.rdmaOps.GetRdmaDevicesForAuxdev(auxDev)
}

// Get RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
func (rmn *rdmaManagerNetlink) GetSystemRdmaMode() (string, error) {
	return rmn.rdmaOps.RdmaSystemGetNetnsMode()
}

// Set RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
func (rmn *rdmaManagerNetlink) SetSystemRdmaMode(mode string) error {
	return rmn.rdmaOps.RdmaSystemSetNetnsMode(mode)
}
