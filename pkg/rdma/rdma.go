package rdma

import (
	"fmt"
	"syscall"

	"github.com/containernetworking/plugins/pkg/ns"
)

const (
	RdmaSysModeExclusive = "exclusive"
	RdmaSysModeShared    = "shared"
)

func NewRdmaManager() RdmaManager {
	return &rdmaManagerNetlink{rdmaOps: newRdmaBasicOps()}
}

type RdmaManager interface {
	// Move RDMA device from current network namespace to network namespace
	MoveRdmaDevToNs(rdmaDev string, netNs ns.NetNS) error
	// Get RDMA devices in the current network namespace
	GetRdmaDeviceList() ([]string, error)
	// Get RDMA devices associated with the given PCI device in D:B:D.f format e.g 0000:04:00.0
	GetRdmaDevsForPciDev(pciDev string) ([]string, error)
	// Get RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
	GetSystemRdmaMode() (string, error)
	// Set RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
	SetSystemRdmaMode(mode string) error
	// Change RDMA device name
	SetRdmaDevName(oldName string, newName string) error
	// Set RDMA device temporary name
	SetRdmaDevTempName(rdmaDev string) (string, error)
}

type rdmaManagerNetlink struct {
	rdmaOps RdmaBasicOps
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
func (rmn *rdmaManagerNetlink) GetRdmaDevsForPciDev(pciDev string) ([]string, error) {
	return rmn.rdmaOps.GetRdmaDevicesForPcidev(pciDev), nil
}

// Get RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
func (rmn *rdmaManagerNetlink) GetSystemRdmaMode() (string, error) {
	return rmn.rdmaOps.RdmaSystemGetNetnsMode()
}

// Set RDMA subsystem namespace awareness mode ["exclusive" | "shared"]
func (rmn *rdmaManagerNetlink) SetSystemRdmaMode(mode string) error {
	return rmn.rdmaOps.RdmaSystemSetNetnsMode(mode)
}

// Change RDMA device name
func (rmn *rdmaManagerNetlink) SetRdmaDevName(oldName string, newName string) error {
	rdmaLink, err := rmn.rdmaOps.RdmaLinkByName(oldName)
	if err != nil {
		return fmt.Errorf("cannot find RDMA link from name: %s", oldName)
	}
	err = rmn.rdmaOps.RdmaLinkSetName(rdmaLink, newName)
	if err != nil {
		return fmt.Errorf("failed to change RDMA device name from %s to %s. %v", oldName, newName, err)
	}
	return nil
}

// Get RDMA devices in the current network namespace
func (rmn *rdmaManagerNetlink) GetRdmaDeviceList() ([]string, error) {
	links, err := rmn.rdmaOps.GetRdmaLinkList()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(links))
	for _, link := range links {
		names = append(names, link.Attrs.Name)
	}
	return names, nil
}

// Set RDMA device to a unique temporary name
func (rmn *rdmaManagerNetlink) SetRdmaDevTempName(rdmaDev string) (string, error) {
	link, err := rmn.rdmaOps.RdmaLinkByName(rdmaDev)
	if err != nil {
		return "", err
	}
	tmpName := fmt.Sprintf("rdmadev_%d", link.Attrs.Index)[:(syscall.IFNAMSIZ - 1)]
	return tmpName, rmn.rdmaOps.RdmaLinkSetName(link, tmpName)
}
