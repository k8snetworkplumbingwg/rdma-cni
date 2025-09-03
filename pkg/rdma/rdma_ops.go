package rdma

import (
	"github.com/Mellanox/rdmamap"
	"github.com/vishvananda/netlink"
)

// Interface to be used by RDMA manager for basic operations
type BasicOps interface {
	// Equivalent to netlink.RdmaLinkByName(...)
	RdmaLinkByName(name string) (*netlink.RdmaLink, error)
	// Equivalent to netlink.RdmaLinkSetNsFd(...)
	RdmaLinkSetNsFd(link *netlink.RdmaLink, fd uint32) error
	// Equivalent to netlink.RdmaSystemGetNetnsMode(...)
	RdmaSystemGetNetnsMode() (string, error)
	// Equivalent to netlink.RdmaSystemSetNetnsMode(...)
	RdmaSystemSetNetnsMode(newMode string) error
	// Equivalent to rdmamap.GetRdmaDevicesForPcidev(...)
	GetRdmaDevicesForPcidev(pcidevName string) []string
	// Equivalent to rdmamap.GetRdmaDevicesForAuxdev(...)
	GetRdmaDevicesForAuxdev(auxDev string) []string
}

func newRdmaBasicOps() BasicOps {
	return &rdmaBasicOpsImpl{}
}

type rdmaBasicOpsImpl struct {
}

// Equivalent to netlink.RdmaLinkByName(...)
func (rdma *rdmaBasicOpsImpl) RdmaLinkByName(name string) (*netlink.RdmaLink, error) {
	return netlink.RdmaLinkByName(name)
}

// Equivalent to netlink.RdmaLinkSetNsFd(...)
func (rdma *rdmaBasicOpsImpl) RdmaLinkSetNsFd(link *netlink.RdmaLink, fd uint32) error {
	return netlink.RdmaLinkSetNsFd(link, fd)
}

// Equivalent to netlink.RdmaSystemGetNetnsMode(...)
func (rdma *rdmaBasicOpsImpl) RdmaSystemGetNetnsMode() (string, error) {
	return netlink.RdmaSystemGetNetnsMode()
}

// Equivalent to netlink.RdmaSystemSetNetnsMode(...)
func (rdma *rdmaBasicOpsImpl) RdmaSystemSetNetnsMode(newMode string) error {
	return netlink.RdmaSystemSetNetnsMode(newMode)
}

// Equivalent to rdmamap.GetRdmaDevicesForPcidev(...)
func (rdma *rdmaBasicOpsImpl) GetRdmaDevicesForPcidev(pcidevName string) []string {
	return rdmamap.GetRdmaDevicesForPcidev(pcidevName)
}

// Equivalent to rdmamap.GetRdmaDevicesForAuxdev(...)
func (rdma *rdmaBasicOpsImpl) GetRdmaDevicesForAuxdev(auxDev string) []string {
	return rdmamap.GetRdmaDevicesForAuxdev(auxDev)
}
