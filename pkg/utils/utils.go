package utils

import (
	"fmt"
	"path"
	"path/filepath"

	"github.com/vishvananda/netlink"
)

// Get VF PCI device associated with the given MAC.
// this method compares with administrative MAC for SRIOV configured net devices
// TODO: move this method to github: Mellanox/sriovnet
func GetVfPciDevFromMAC(mac string) (string, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return "", err
	}
	matchDevs := []string{}
	for _, link := range links {
		if len(link.Attrs().Vfs) > 0 {
			for _, vf := range link.Attrs().Vfs {
				if vf.Mac.String() == mac {
					vfPath, err := filepath.EvalSymlinks(fmt.Sprintf("/sys/class/net/%s/device/virtfn%d", link.Attrs().Name, vf.ID))
					if err == nil {
						matchDevs = append(matchDevs, path.Base(vfPath))
					}
				}
			}
		}
	}

	var dev string
	switch len(matchDevs) {
	case 1:
		dev = matchDevs[0]
		err = nil
	case 0:
		err = fmt.Errorf("could not find VF PCI device according to administrative mac address set on PF")
	default:
		err = fmt.Errorf("found more than one VF PCI device matching provided administrative mac address")
	}
	return dev, err
}
