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

	Describe("Embedded dots with the filesystem cache", func() {
		BeforeEach(func() {
			previousDir := cache.CacheDir
			cache.CacheDir = GinkgoT().TempDir()
			DeferCleanup(func() { cache.CacheDir = previousDir })
		})
		JustBeforeEach(func() {
			plugin.stateCache = cache.NewStateCache()
		})

		It("ADD discovers, moves and saves a network with embedded dots", func() {
			conf := generateNetConfCmdAdd("my..net", "net1", "0000:04:00.5")
			args := generateArgs("/dummy/netns", "cid", "net1", &conf)
			containerNs, _ := dummyNsMgr.GetNS(args.Netns)
			rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil).Once()
			rdmaMgrMock.On("GetRdmaDevsForPciDev", conf.DeviceID).Return([]string{"mlx5_4"}).Once()
			rdmaMgrMock.On("MoveRdmaDevToNs", "mlx5_4", containerNs).Return(nil).Once()

			Expect(plugin.CmdAdd(&args)).To(Succeed())
			var loaded rdmaTypes.RdmaNetState
			Expect(plugin.stateCache.Load(cache.StateRef("my..net-cid-net1"), &loaded)).To(Succeed())
			Expect(loaded).To(Equal(generateRdmaNetState(conf.DeviceID, "mlx5_4", "mlx5_4")))
			rdmaMgrMock.AssertExpectations(t)
		})

		It("DEL restores the device and deletes pre-existing legacy state", func() {
			ref := cache.StateRef("my..net-cid-net1")
			state := generateRdmaNetState("0000:04:00.5", "mlx5_4", "mlx5_4")
			Expect(plugin.stateCache.Save(ref, &state)).To(Succeed())
			conf := generateNetConfCmdDel("my..net")
			args := generateArgs("/dummy/netns", "cid", "net1", &conf)
			hostNs, _ := dummyNsMgr.GetCurrentNS()
			rdmaMgrMock.On("MoveRdmaDevToNs", "mlx5_4", hostNs).Return(nil).Once()

			Expect(plugin.CmdDel(&args)).To(Succeed())
			var loaded rdmaTypes.RdmaNetState
			Expect(plugin.stateCache.Load(ref, &loaded)).NotTo(Succeed())
			rdmaMgrMock.AssertExpectations(t)
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
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"), nil)
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
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"), nil)
				expectedState := generateRdmaNetState(auxDev, rdmaDev, rdmaDev)
				stateCacheMock.On("Save", mock.AnythingOfType("cache.StateRef"), &expectedState).Return(nil)
				err := plugin.CmdAdd(&args)
				Expect(err).ToNot(HaveOccurred())
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
		})
		Context("GetStateRef returns error", func() {
			It("Should reject the reference before device discovery or movement", func() {
				pciDev := "0000:04:00.5"
				netName := "net/with/slashes"
				cIfname := "net1"
				cid := "a1b2c3d4e5f6"
				cnsPath := "/proc/12444/ns/net"
				netconf := generateNetConfCmdAdd(netName, cIfname, pciDev)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(
					cache.StateRef(""), fmt.Errorf("invalid state ref component: %q", netName))
				err := plugin.CmdAdd(&args)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state reference"))
				rdmaMgrMock.AssertExpectations(t)
				stateCacheMock.AssertExpectations(t)
			})
		})
	})

	Describe("Cache-save rollback", func() {
		It("restores the device when saving state fails", func() {
			conf := generateNetConfCmdAdd("mynet", "net1", "0000:04:00.5")
			args := generateArgs("/proc/12444/ns/net", "cid", "net1", &conf)
			cns, _ := dummyNsMgr.GetNS(args.Netns)
			curNs, _ := dummyNsMgr.GetCurrentNS()
			rdmaMgrMock.On("GetSystemRdmaMode").Return(rdma.RdmaSysModeExclusive, nil)
			stateCacheMock.On("GetStateRef", "mynet", "cid", "net1").Return(cache.StateRef("ref"), nil)
			rdmaMgrMock.On("GetRdmaDevsForPciDev", conf.DeviceID).Return([]string{"mlx5_0"})
			rdmaMgrMock.On("MoveRdmaDevToNs", "mlx5_0", cns).Return(nil).Once()
			rdmaMgrMock.On("MoveRdmaDevToNs", "mlx5_0", curNs).Return(nil).Once()
			stateCacheMock.On("Save", cache.StateRef("ref"), mock.Anything).Return(fmt.Errorf("save failed"))
			Expect(plugin.CmdAdd(&args)).To(MatchError("save failed"))
			rdmaMgrMock.AssertExpectations(t)
			stateCacheMock.AssertExpectations(t)
		})
	})

	Describe("Test getRDMADevice()", func() {
		DescribeTable("rejects zero or multiple discovered devices",
			func(devices []string, message string) {
				rdmaMgrMock.On("GetRdmaDevsForAuxDev", "mlx5_core.sf.4").Return(devices)
				_, err := plugin.getRDMADevice("mlx5_core.sf.4")
				Expect(err).To(MatchError(ContainSubstring(message)))
			},
			Entry("none", []string{}, "no RDMA devices"),
			Entry("multiple", []string{"mlx5_0", "mlx5_1"}, "more than one"),
		)

		Context("Input validation for non-PCI device IDs", func() {
			It("Should reject deviceID containing path traversal components", func() {
				_, err := plugin.getRDMADevice("../../pci/devices/0000:03:00.0")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid device ID"))
			})

			It("Should reject deviceID containing forward slashes", func() {
				_, err := plugin.getRDMADevice("foo/bar")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid device ID"))
			})

			It("Should reject deviceID that is just dot-dot", func() {
				_, err := plugin.getRDMADevice("..")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid device ID"))
			})

			It("Should accept valid auxiliary device ID", func() {
				validAuxDev := "mlx5_core.sf.4"
				rdmaMgrMock.On("GetRdmaDevsForAuxDev", validAuxDev).Return([]string{"mlx5_0"})
				dev, err := plugin.getRDMADevice(validAuxDev)
				Expect(err).ToNot(HaveOccurred())
				Expect(dev).To(Equal("mlx5_0"))
				rdmaMgrMock.AssertExpectations(t)
			})

			It("Should accept valid PCI device ID", func() {
				validPciDev := "0000:04:00.5"
				rdmaMgrMock.On("GetRdmaDevsForPciDev", validPciDev).Return([]string{"mlx5_0"})
				dev, err := plugin.getRDMADevice(validPciDev)
				Expect(err).ToNot(HaveOccurred())
				Expect(dev).To(Equal("mlx5_0"))
				rdmaMgrMock.AssertExpectations(t)
			})
		})
	})

	Describe("Test CmdDel()", func() {
		It("succeeds without cache access when the namespace is absent", func() {
			conf := generateNetConfCmdDel("mynet")
			args := generateArgs("", "cid", "net1", &conf)
			Expect(plugin.CmdDel(&args)).To(Succeed())
			Expect(stateCacheMock.Calls).To(BeEmpty())
			Expect(rdmaMgrMock.Calls).To(BeEmpty())
		})
		It("succeeds without movement when the cache entry is absent", func() {
			conf := generateNetConfCmdDel("mynet")
			args := generateArgs("/proc/12444/ns/net", "cid", "net1", &conf)
			stateCacheMock.On("GetStateRef", "mynet", "cid", "net1").Return(cache.StateRef("ref"), nil)
			stateCacheMock.On("Load", cache.StateRef("ref"), mock.Anything).Return(fmt.Errorf("not found"))
			Expect(plugin.CmdDel(&args)).To(Succeed())
			Expect(rdmaMgrMock.Calls).To(BeEmpty())
			stateCacheMock.AssertExpectations(t)
		})

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
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"), nil)
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
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(cache.StateRef("some-ref"), nil)
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
		Context("GetStateRef returns error", func() {
			It("Should return nil without attempting cleanup", func() {
				netName := "net/with/slashes"
				cIfname := "net1"
				cid := "a1b2c3d4e5f6"
				cnsPath := "/proc/12444/ns/net"
				netconf := generateNetConfCmdDel(netName)
				args := generateArgs(cnsPath, cid, cIfname, &netconf)
				stateCacheMock.On("GetStateRef", netName, cid, cIfname).Return(
					cache.StateRef(""), fmt.Errorf("invalid state ref component: %q", netName))
				err := plugin.CmdDel(&args)
				Expect(err).ToNot(HaveOccurred())
				stateCacheMock.AssertExpectations(t)
				stateCacheMock.AssertNotCalled(t, "Load")
				stateCacheMock.AssertNotCalled(t, "Delete")
			})
		})
	})

	Describe("Test CmdCheck()", func() {
		It("Should basically do nothing", func() {
			Expect(plugin.CmdCheck(nil)).To(Succeed())
		})
	})
})
