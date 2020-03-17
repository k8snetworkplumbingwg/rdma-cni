package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Mellanox/rdma-cni/pkg/cache"
	"github.com/Mellanox/rdma-cni/pkg/rdma"
	rdmatypes "github.com/Mellanox/rdma-cni/pkg/types"
	"github.com/Mellanox/rdma-cni/pkg/utils"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Sets the initial log level configurations
// this is overridden by the "debug" CNI arg
var (
	logLevel = zerolog.InfoLevel
)

type NsManager interface {
	GetNS(string) (ns.NetNS, error)
	GetCurrentNS() (ns.NetNS, error)
}

type nsManagerImpl struct {
}

func (nsm *nsManagerImpl) GetNS(nspath string) (ns.NetNS, error) {
	return ns.GetNS(nspath)
}

func (nsm *nsManagerImpl) GetCurrentNS() (ns.NetNS, error) {
	return ns.GetCurrentNS()
}

func newNsManager() NsManager {
	return &nsManagerImpl{}
}

type rdmaCniPlugin struct {
	rdmaManager rdma.RdmaManager
	nsManager   NsManager
	stateCache  cache.StateCache
}

// Ensure RDMA subsystem mode is set to exclusive.
func (plugin *rdmaCniPlugin) ensureRdmaSystemMode() error {
	mode, err := plugin.rdmaManager.GetSystemRdmaMode()
	if err != nil {
		return fmt.Errorf("failed to get RDMA subsystem namespace awareness mode. %v", err)
	}
	log.Debug().Msgf("RDMA subsystem mode: %s", mode)
	if mode != rdma.RdmaSysModeExclusive {
		return fmt.Errorf("RDMA subsystem namespace awareness mode is set to %s, "+
			"expecting it to be set to %s, invalid system configurations", mode, rdma.RdmaSysModeExclusive)
	}
	return nil
}

func (plugin *rdmaCniPlugin) deriveDeviceIDFromResult(result *current.Result) (string, error) {
	log.Warn().Msgf("DeviceID attribute in network configuration is empty, " +
		"this may indicated that the delegate plugin is out of date.")

	var deviceID string
	var err error
	if len(result.Interfaces) == 1 {
		log.Debug().Msgf("Attempting to derive DeviceID from MAC.")
		deviceID, err = utils.GetVfPciDevFromMAC(result.Interfaces[0].Mac)
		if err != nil {
			return deviceID, fmt.Errorf("failed to derive PCI device ID from mac %q. %v", result.Interfaces[0].Mac, err)
		}
	} else {
		return deviceID, fmt.Errorf("\"DeviceID\" network configuration attribute is required for rdma CNI")
	}
	return deviceID, nil
}

// Parse network configurations
func (plugin *rdmaCniPlugin) parseConf(data []byte, envArgs string) (*rdmatypes.RdmaNetConf, error) {
	conf := rdmatypes.RdmaNetConf{}
	// Parse CNI args passed as env variables (not used ATM)
	if envArgs != "" {
		commonCniArgs := &conf.Args.CNI
		err := types.LoadArgs(envArgs, commonCniArgs)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("ENV CNI_ARGS: %+v", commonCniArgs)
	}

	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, fmt.Errorf("failed to load netconf: %+v", err)
	}
	log.Debug().Msgf("Network Configuration: %+v", conf)
	return &conf, nil
}

// Move RDMA device to namespace
func (plugin *rdmaCniPlugin) moveRdmaDevToNs(rdmaDev string, nsPath string) error {
	log.Debug().Msgf("moving RDMA device %s to namespace %s", rdmaDev, nsPath)

	targetNs, err := plugin.nsManager.GetNS(nsPath)
	if err != nil {
		return fmt.Errorf("failed to open network namespace %s: %v", nsPath, err)
	}
	defer targetNs.Close()

	err = plugin.rdmaManager.MoveRdmaDevToNs(rdmaDev, targetNs)
	if err != nil {
		return fmt.Errorf("failed to move RDMA device %s to namespace. %v", rdmaDev, err)
	}
	return nil
}

// Move RDMA device from namespace to current (default) namespace
func (plugin *rdmaCniPlugin) moveRdmaDevFromNs(rdmaDev string, nsPath string) error {
	log.Debug().Msgf("INFO: moving RDMA device %s from namespace %s to default namespace", rdmaDev, nsPath)

	sourceNs, err := plugin.nsManager.GetNS(nsPath)
	if err != nil {
		return fmt.Errorf("failed to open network namespace %s: %v", nsPath, err)
	}
	defer sourceNs.Close()

	targetNs, err := plugin.nsManager.GetCurrentNS()
	if err != nil {
		return fmt.Errorf("failed to open current network namespace: %v", err)
	}
	defer targetNs.Close()

	err = sourceNs.Do(func(_ ns.NetNS) error {
		// Move RDMA device to default namespace
		return plugin.rdmaManager.MoveRdmaDevToNs(rdmaDev, targetNs)
	})
	if err != nil {
		return fmt.Errorf("failed to move RDMA device %s to default namespace. %v", rdmaDev, err)
	}
	return err
}

func (plugin *rdmaCniPlugin) CmdAdd(args *skel.CmdArgs) error {
	log.Info().Msgf("RDMA-CNI: cmdAdd")
	conf, err := plugin.parseConf(args.StdinData, args.Args)
	if err != nil {
		return err
	}
	if conf.Args.CNI.Debug {
		setDebugMode()
	}

	log.Debug().Msgf("cmdAdd: args: %+v ", args)

	// Ensure RDMA-CNI was called as part of a chain, and parse PrevResult
	if conf.RawPrevResult == nil {
		return fmt.Errorf("RDMA-CNI is expected to be called as part of a plugin chain")
	}
	if err = version.ParsePrevResult(&conf.NetConf); err != nil {
		return err
	}
	result, err := current.NewResultFromResult(conf.PrevResult)
	if err != nil {
		return err
	}
	log.Debug().Msgf("prev results: %+v", result)

	// Ensure RDMA subsystem mode
	err = plugin.ensureRdmaSystemMode()
	if err != nil {
		return err
	}

	// Delegate plugin may not add Device ID to the network configuration, if so,
	// attempt to derive it from PrevResult Mac address with some sysfs voodoo
	if conf.DeviceID == "" {
		if conf.DeviceID, err = plugin.deriveDeviceIDFromResult(result); err != nil {
			return err
		}
	}

	rdmaDevs, err := plugin.rdmaManager.GetRdmaDevsForPciDev(conf.DeviceID)
	if err != nil || len(rdmaDevs) == 0 {
		return fmt.Errorf("failed to get RDMA devices for PCI device: %s. %v", conf.DeviceID, err)
	}

	if len(rdmaDevs) != 1 {
		// Expecting exactly one RDMA device
		return fmt.Errorf(
			"discovered more than one RDMA device %v for PCI device %s. Unsupported state", rdmaDevs, conf.DeviceID)
	}

	// Move RDMA device to container namespace
	rdmaDev := rdmaDevs[0]

	err = plugin.moveRdmaDevToNs(rdmaDev, args.Netns)
	if err != nil {
		return fmt.Errorf("failed to move RDMA device %s to namespace. %v", rdmaDev, err)
	}

	// Save RDMA state
	state := rdmatypes.NewRdmaNetState()
	state.DeviceID = conf.DeviceID
	state.SandboxRdmaDevName = rdmaDev
	state.ContainerRdmaDevName = rdmaDev
	pRef := plugin.stateCache.GetStateRef(conf.Name, args.ContainerID, args.IfName)
	err = plugin.stateCache.Save(pRef, &state)
	if err != nil {
		// Move RDMA dev back to current namespace
		restoreErr := plugin.moveRdmaDevFromNs(state.ContainerRdmaDevName, args.Netns)
		if restoreErr != nil {
			return fmt.Errorf("save to cache failed %v, failed while restoring namespace for RDMA device %s. %v", err, rdmaDev, restoreErr)
		}
		return err
	}
	return types.PrintResult(result, conf.CNIVersion)
}

func (plugin *rdmaCniPlugin) CmdCheck(args *skel.CmdArgs) error {
	log.Info().Msgf("cmdCheck() not Implemented. args: %v ", args)
	return nil
}

func (plugin *rdmaCniPlugin) CmdDel(args *skel.CmdArgs) error {
	log.Info().Msgf("RDMA-CNI: cmdDel")
	conf, err := plugin.parseConf(args.StdinData, args.Args)
	if err != nil {
		return err
	}
	if conf.Args.CNI.Debug {
		setDebugMode()
	}
	log.Debug().Msgf("CmdDel() args: %v ", args)

	// Container already exited, so no Namespace. if no Namespace, we got nothing to clean.
	// this may happen in Infra containers as described in https://github.com/kubernetes/kubernetes/pull/35240
	if args.Netns == "" {
		return nil
	}

	// Load RDMA device state from cache
	rdmaState := rdmatypes.RdmaNetState{}
	pRef := plugin.stateCache.GetStateRef(conf.Name, args.ContainerID, args.IfName)
	err = plugin.stateCache.Load(pRef, &rdmaState)
	if err != nil {
		return err
	}

	// Move RDMA device to default namespace
	err = plugin.moveRdmaDevFromNs(rdmaState.ContainerRdmaDevName, args.Netns)
	if err != nil {
		return fmt.Errorf(
			"failed to restore RDMA device %s to default namespace. %v", rdmaState.ContainerRdmaDevName, err)
	}

	err = plugin.stateCache.Delete(pRef)
	if err != nil {
		log.Warn().Msgf("failed to delete cache entry(%q). %v", pRef, err)
	}
	return nil
}

func setupLogging() {
	zerolog.SetGlobalLevel(logLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: zerolog.TimeFieldFormat,
		NoColor:    true})
}

func setDebugMode() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	setupLogging()
	plugin := rdmaCniPlugin{
		rdmaManager: rdma.NewRdmaManager(),
		nsManager:   newNsManager(),
		stateCache:  cache.NewStateCache(),
	}
	skel.PluginMain(plugin.CmdAdd, plugin.CmdCheck, plugin.CmdDel, version.All, "")
}
