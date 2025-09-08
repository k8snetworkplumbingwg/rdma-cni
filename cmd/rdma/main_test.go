package main

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/k8snetworkplumbingwg/rdma-cni/pkg/cache"
	cacheMocks "github.com/k8snetworkplumbingwg/rdma-cni/pkg/cache/mocks"
	"github.com/k8snetworkplumbingwg/rdma-cni/pkg/rdma"
	rdmaMocks "github.com/k8snetworkplumbingwg/rdma-cni/pkg/rdma/mocks"
	rdmaTypes "github.com/k8snetworkplumbingwg/rdma-cni/pkg/types"
)

func generateNetConfCmdDel(netName string) rdmaTypes.RdmaNetConf {
	return rdmaTypes.RdmaNetConf{
		NetConf: types.NetConf{
			CNIVersion:    "0.4.0",
			Name:          netName,
			Type:          "rdma",
			Capabilities:  nil,
			IPAM:          types.IPAM{},
			DNS:           types.DNS{},
			RawPrevResult: nil,
			PrevResult:    nil,
		},
		DeviceID: "",
	}
}

func generateNetConfCmdAdd(netName, cIfname, deviceID string) rdmaTypes.RdmaNetConf {
	prevResult := current.Result{
		CNIVersion: "0.4.0",
		Interfaces: []*current.Interface{{
			Name:    cIfname,
			Mac:     "42:86:24:84:4f:b1",
			Sandbox: "/proc/1/ns/net",
		}},
		IPs:    nil,
		Routes: nil,
		DNS:    types.DNS{},
	}
	bytes, _ := json.Marshal(prevResult)
	var raw map[string]interface{}
	_ = json.Unmarshal(bytes, &raw)

	return rdmaTypes.RdmaNetConf{
		NetConf: types.NetConf{
			CNIVersion:    "0.4.0",
			Name:          netName,
			Type:          "rdma",
			Capabilities:  nil,
			IPAM:          types.IPAM{},
			DNS:           types.DNS{},
			RawPrevResult: raw,
			PrevResult:    nil,
		},
		DeviceID: deviceID,
	}
}

func generateArgs(nsPath, cid, cIfname string, netconf *rdmaTypes.RdmaNetConf) skel.CmdArgs {
	bytes, _ := json.Marshal(*netconf)
	return skel.CmdArgs{
		ContainerID: cid,
		Netns:       nsPath,
		IfName:      cIfname,
		Args:        "",
		Path:        "",
		StdinData:   bytes,
	}
}

func generateRdmaNetState(deviceID, sanboxRdmaDev, containerRdmaDev string) rdmaTypes.RdmaNetState {
	state := rdmaTypes.NewRdmaNetState()
	state.DeviceID = deviceID
	state.SandboxRdmaDevName = sanboxRdmaDev
	state.ContainerRdmaDevName = containerRdmaDev
	return state
}

type dummyNetNs struct {
	fd   uintptr
	path string
}

func (dns *dummyNetNs) Fd() uintptr {
	return dns.fd
}

func (dns *dummyNetNs) Do(toRun func(ns.NetNS) error) error {
	return toRun(&dummyNetNs{fd: 19, path: "dummy/path"})
}

func (dns *dummyNetNs) Set() error {
	return nil
}

func (dns *dummyNetNs) Path() string {
	return dns.path
}

func (dns *dummyNetNs) Close() error {
	return nil
}

type dummyNsMananger struct {
}

func (nsm *dummyNsMananger) GetNS(nspath string) (ns.NetNS, error) {
	return &dummyNetNs{path: nspath, fd: 17}, nil
}

func (nsm *dummyNsMananger) GetCurrentNS() (ns.NetNS, error) {
	return &dummyNetNs{path: "/proc/2/ns/net", fd: 17}, nil
}

var _ = Describe("Main", func() {
	var (
		plugin         rdmaCniPlugin
		dummyNsMgr     dummyNsMananger
		rdmaMgrMock    rdmaMocks.MockManager
		stateCacheMock cacheMocks.MockStateCache
		t              GinkgoTInterface
	)

	JustBeforeEach(func() {
		rdmaMgrMock = rdmaMocks.MockManager{}
		dummyNsMgr = dummyNsMananger{}
		stateCacheMock = cacheMocks.MockStateCache{}
		t = GinkgoT()
		plugin = rdmaCniPlugin{
			rdmaManager: &rdmaMgrMock,
			stateCache:  &stateCacheMock,
			nsManager:   &dummyNsMgr,
		}
	})

	Describe("Test ensureRdmaSystemMode()", func() {

		Context("Bad flows", func() {
			It("Should error out if rdma system namespace mode is not exclusive", func() {
				rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeShared, nil)
				err := plugin.ensureRdmaSystemMode()
				Expect(err).To(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
			})
			It("Should error out on failure to get rdma system namespace mode", func() {
				retErr := fmt.Errorf("error")
				rdmaMgrMock.On("GetSystemRdmaMode").Return("", retErr)
				err := plugin.ensureRdmaSystemMode()
				Expect(err).To(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
		Context("Good flow", func() {
			It("Should succeed if rdma system namespace mode is exclusive", func() {
				rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil)
				Expect(plugin.ensureRdmaSystemMode()).To(Succeed())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
	})

	Describe("Test moveRdmaDevToNs()", func() {

		Context("Good flow", func() {
			It("Should succeed and move RDMA device to given namespace", func() {
				rdmaDev := "mlx5_5"
				nsPath := "/proc/666/ns/net"
				containerNs, _ := dummyNsMgr.GetNS(nsPath)
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, containerNs).Return(nil)
				Expect(plugin.moveRdmaDevToNs(rdmaDev, nsPath)).To(Succeed())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
		Context("Bad flow", func() {
			It("Should fail", func() {
				retErr := fmt.Errorf("error occurred")
				rdmaMgrMock.On(
					"MoveRdmaDevToNs",
					mock.AnythingOfType("string"),
					mock.AnythingOfType("*main.dummyNetNs")).Return(retErr)
				err := plugin.moveRdmaDevToNs("mlx5_5", "/proc/666/ns/net")
				Expect(err).To(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
	})

	Describe("Test moveRdmaDevFromNs()", func() {

		Context("Good flow", func() {
			It("Should succeed and move RDMA device to current namespace", func() {
				rdmaDev := "mlx5_5"
				nsPath := "/proc/666/ns/net"
				currNs, _ := dummyNsMgr.GetCurrentNS()
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, currNs).Return(nil)
				Expect(plugin.moveRdmaDevFromNs(rdmaDev, nsPath)).To(Succeed())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
		Context("Bad flow", func() {
			It("Should fail", func() {
				retErr := fmt.Errorf("error occurred")
				rdmaMgrMock.On("MoveRdmaDevToNs",
					mock.AnythingOfType("string"),
					mock.AnythingOfType("*main.dummyNetNs")).Return(retErr)
				err := plugin.moveRdmaDevFromNs("mlx5_5", "/proc/666/ns/net")
				Expect(err).To(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
			})
		})
	})

	Describe("Test CmdAdd()", func() {
		Context("Valid configuration provided", func() {
			It("Should succeed and move Rdma device associated with provided PCI DeviceID to Namespace", func() {
				pciDev := "0000:04:00.5"
				netName := "rdma-net"
				rdmaDev := "mlx5_4"
				cIfname := "net1"
				cid := "a1b2c3d4e5f6"
				cnsPath := "/proc/12444/ns/net"
				cns, _ := dummyNsMgr.GetNS(cnsPath)
				netconf := generateNetConfCmdAdd(netName, cIfname, pciDev)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil)
				rdmaMgrMock.On("GetRdmaDevsForPciDev", pciDev).Return([]string{rdmaDev}, nil)
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, cns).Return(nil)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"))
				expectedState := generateRdmaNetState(pciDev, rdmaDev, rdmaDev)
				stateCacheMock.On("Save", mock.AnythingOfType("cache.StateRef"), &expectedState).Return(nil)
				err := plugin.CmdAdd(&args)
				Expect(err).ToNot(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
			It("Should succeed and move Rdma device associated with auxiliary device DeviceID to Namespace", func() {
				auxDev := "mlx5_core.sf.6"
				netName := "rdma-net"
				rdmaDev := "mlx5_6"
				cIfname := "net2"
				cid := "a6b5c4d3e2f1"
				cnsPath := "/proc/11142/ns/net"
				cns, _ := dummyNsMgr.GetNS(cnsPath)
				netconf := generateNetConfCmdAdd(netName, cIfname, auxDev)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil)
				rdmaMgrMock.On("GetRdmaDevsForAuxDev", auxDev).Return([]string{rdmaDev}, nil)
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, cns).Return(nil)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"))
				expectedState := generateRdmaNetState(auxDev, rdmaDev, rdmaDev)
				stateCacheMock.On("Save", mock.AnythingOfType("cache.StateRef"), &expectedState).Return(nil)
				err := plugin.CmdAdd(&args)
				Expect(err).ToNot(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
		})
		// TODO(adrian): Add additional tests to cover bad flows / differen network configurations
	})

	Describe("Test CmdDel()", func() {
		Context("Valid configuration provided", func() {
			It("Should succeed and move Rdma device associated with PCI net device back to sandbox namespace", func() {
				pciDev := "0000:04:00.5"
				netName := "rdma-net"
				rdmaDev := "mlx5_4"
				cIfname := "net1"
				cid := "a1b2c3d4e5f6"
				cnsPath := "/proc/12444/ns/net"
				cns, _ := dummyNsMgr.GetCurrentNS()
				rdmaState := generateRdmaNetState(pciDev, rdmaDev, rdmaDev)
				netconf := generateNetConfCmdDel(netName)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"))
				stateCacheMock.On("Load", mock.AnythingOfType("cache.StateRef"),
					mock.AnythingOfType("*types.RdmaNetState")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(1).(*rdmaTypes.RdmaNetState)
					*arg = rdmaState
				})
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, cns).Return(nil)
				stateCacheMock.On("Delete", mock.AnythingOfType("cache.StateRef")).Return(nil)
				err := plugin.CmdDel(&args)
				Expect(err).ToNot(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
			It("Should succeed and move Rdma device associated with auxiliary device back to sandbox namespace", func() {
				auxDev := "mlx5_core.sf.6"
				netName := "rdma-net"
				rdmaDev := "mlx5_6"
				cIfname := "net2"
				cid := "a1b2c3d4e5f6"
				cnsPath := "/proc/12444/ns/net"
				cns, _ := dummyNsMgr.GetCurrentNS()
				rdmaState := generateRdmaNetState(auxDev, rdmaDev, rdmaDev)
				netconf := generateNetConfCmdDel(netName)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"))
				stateCacheMock.On("Load", mock.AnythingOfType("cache.StateRef"),
					mock.AnythingOfType("*types.RdmaNetState")).Return(nil).Run(func(args mock.Arguments) {
					arg := args.Get(1).(*rdmaTypes.RdmaNetState)
					*arg = rdmaState
				})
				rdmaMgrMock.On("MoveRdmaDevToNs", rdmaDev, cns).Return(nil)
				stateCacheMock.On("Delete", mock.AnythingOfType("cache.StateRef")).Return(nil)
				err := plugin.CmdDel(&args)
				Expect(err).ToNot(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
		})
		// TODO(adrian): Add additional tests to cover bad flows / different network configurations
	})

	Describe("Test CmdCheck()", func() {
		It("Should basically do nothing", func() {
			Expect(plugin.CmdCheck(nil)).To(Succeed())
		})
	})
})
