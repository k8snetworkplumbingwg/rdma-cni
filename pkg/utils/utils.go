package utils

import (
	"fmt"
	"path"
	"path/filepath"
	"regexp"

	"github.com/vishvananda/netlink"
)

// Get VF PCI device associated with the given MAC.
// this method compares with administrative MAC for SRIOV configured net devices
// TODO: move this method to github: Mellanox/sriovnet
func GetVfPciDevFromMAC(mac string) (string, error) {
	var err error
	var links []netlink.Link
	var vfPath string
	links, err = netlink.LinkList()
	if err != nil {
		return "", err
	}
	matchDevs := []string{}
	for _, link := range links {
		if len(link.Attrs().Vfs) > 0 {
			for i := range link.Attrs().Vfs {
				if link.Attrs().Vfs[i].Mac.String() == mac {
					vfPath, err = filepath.EvalSymlinks(
						fmt.Sprintf("/sys/class/net/%s/device/virtfn%d", link.Attrs().Name, link.Attrs().Vfs[i].ID))
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

// IsPCIAddress returns whether the input is a valid PCI address.
func IsPCIAddress(pciAddress string) bool {
	re := regexp.MustCompile(`^[0-9a-fA-F]{4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-7]$`)
	return re.MatchString(pciAddress)
}
