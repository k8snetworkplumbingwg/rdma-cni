package utils

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

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
			for _, vf := range link.Attrs().Vfs {
				if vf.Mac.String() == mac {
					vfPath, err = filepath.EvalSymlinks(fmt.Sprintf("/sys/class/net/%s/device/virtfn%d", link.Attrs().Name, vf.ID))
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

// Get RDMA device prefix and index. e.g for mlx5_3: prefix is mlx5 and index is 3
// Note: the index is not related to the kernel RDMA device index
func getRdmaDevNamePrefixIndex(rdmaDev string) (prefix string, idx uint64, err error) {
	s := strings.Split(rdmaDev, `_`)
	if len(s) != 2 {
		return "", 0, fmt.Errorf("unexpeded RDMA device format: %s", rdmaDev)
	}
	prefix = s[0]
	idx, err = strconv.ParseUint(s[1], 0, 32)
	if err != nil {
		err = fmt.Errorf("failed to parse RDMA device index: %s, %v", rdmaDev, err)
	}
	return prefix, idx, err
}

func getRdmaDevIndexFromName(rdmaDev string) (uint64, error) {
	_, idx, err := getRdmaDevNamePrefixIndex(rdmaDev)
	return idx, err
}

// Get the next RDMA device name for a given RDMA device prefix
func GetNextRdmaDeviceName(prefix string, currDevs []string) (string, error) {
	var nextDevIdx uint64
	nextDevIdx = 0
	if len(currDevs) != 0 {
		for _, dev := range currDevs {
			if !strings.HasPrefix(dev, prefix) {
				continue
			}
			// extract index
			idx, err := getRdmaDevIndexFromName(dev)
			if err != nil {
				return "", err
			}
			if idx > nextDevIdx {
				nextDevIdx = idx + 1
			}
		}
	}
	return fmt.Sprintf("%s_%d", prefix, nextDevIdx), nil
}

// Get RDMA device driver prefix. e.g for mlx5_3 the prefix would be mlx5
func GetRdmaDevicePrefix(rdmaDev string) (string, error) {
	prefix, _, err := getRdmaDevNamePrefixIndex(rdmaDev)
	return prefix, err
}
