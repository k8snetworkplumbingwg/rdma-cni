package types

import (
	"github.com/containernetworking/cni/pkg/types"
)

type RdmaNetConf struct {
	types.NetConf
	DeviceID string  `json:"deviceID"` // PCI address of a VF in valid sysfs format
	Args     CNIArgs `json:"args"`     // optional arguments passed to CNI as defined in CNI spec 0.2.0
}

type CNIArgs struct {
	CNI RdmaCNIArgs `json:"cni"`
}

type RdmaCNIArgs struct {
	types.CommonArgs
	Debug bool `json:"debug"` // Run CNI in debug mode
}

// RDMA Network state struct version
// minor should be bumped when new fields are added
// major should be bumped when non backward compatible changes are introduced
const RdmaNetStateVersion = "1.0"

func NewRdmaNetState() RdmaNetState {
	return RdmaNetState{Version: RdmaNetStateVersion}
}

type RdmaNetState struct {
	// RDMA network state struct version
	Version string `json:"version"`
	// PCI device ID associated with the RDMA device
	DeviceID string `json:"deviceID"`
	// RDMA device name as originally appeared in sandbox
	SandboxRdmaDevName string `json:"sandboxRdmaDevName"`
	// RDMA device name in container
	ContainerRdmaDevName string `json:"containerRdmaDevName"`
}
