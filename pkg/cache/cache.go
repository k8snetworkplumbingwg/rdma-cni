package cache

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	// By default, used for caching CNI network state
	CacheDir = "/var/lib/cni/rdma"
)

type StateRef string

type StateCache interface {
	// Get State reference identifier for <networkName, containerID, interfaceName>
	GetStateRef(network string, cid string, ifname string) StateRef
	// Save state to cache
	Save(ref StateRef, state interface{}) error
	// Load state from cache
	Load(ref StateRef, state interface{}) error
	// Delete state from cache
	Delete(ref StateRef) error
}

// Create a new RDMA state Cache that will Save/Load state
func NewStateCache() StateCache {
	return &FsStateCache{basePath: CacheDir, fsOps: newFsOps()}
}

type FsStateCache struct {
	basePath string
	fsOps    FileSystemOps
}

func (sc *FsStateCache) GetStateRef(network string, cid string, ifname string) StateRef {
	return StateRef(strings.Join([]string{network, cid, ifname}, "-"))
}

func (sc *FsStateCache) Save(ref StateRef, state interface{}) error {
	sRef := string(ref)
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}

	if err = sc.fsOps.MkdirAll(sc.basePath, 0700); err != nil {
		return fmt.Errorf("failed to create data cache directory(%q): %v", sc.basePath, err)
	}

	path := filepath.Join(sc.basePath, sRef)

	err = sc.fsOps.WriteFile(path, bytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write cache data in the path(%q): %v", path, err)
	}

	return err
}

func (sc *FsStateCache) Load(ref StateRef, state interface{}) error {
	sRef := string(ref)
	path := filepath.Join(sc.basePath, sRef)
	bytes, err := sc.fsOps.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read cache data in the path(%q): %v", path, err)
	}
	return json.Unmarshal(bytes, state)
}

func (sc *FsStateCache) Delete(ref StateRef) error {
	sRef := string(ref)
	path := filepath.Join(sc.basePath, sRef)
	if err := sc.fsOps.Remove(path); err != nil {
		return fmt.Errorf("error removing cache file %q: %v", path, err)
	}
	return nil
}
