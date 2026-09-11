package cache

import (
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type myTestState struct {
	FirstState  string `json:"firstState"`
	SecondState int    `json:"secondState"`
}

var _ = Describe("Cache - Simple marshall-able state-object cache", func() {
	var stateCache StateCache
	var fs FileSystemOps
	JustBeforeEach(func() {
		fs = newFakeFileSystemOps()
		stateCache = &FsStateCache{basePath: CacheDir, fsOps: fs}
	})

	Describe("Get State reference", func() {
		Context("Basic call", func() {
			It("Should return <network>-<cid>-<ifname>", func() {
				ref, err := stateCache.GetStateRef("myNet", "containerUniqueIdentifier", "net1")
				Expect(err).ToNot(HaveOccurred())
				Expect(ref).To(BeEquivalentTo("myNet-containerUniqueIdentifier-net1"))
			})
		})
	})

	Describe("Save and Load State", func() {
		var sRef StateRef
		JustBeforeEach(func() {
			var err error
			sRef, err = stateCache.GetStateRef("mynet", "cid", "net1")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("Save and Load with marshallable object", func() {
			It("Should save/load the state", func() {
				savedState := myTestState{FirstState: "first", SecondState: 42}
				var loadedState myTestState
				Expect(stateCache.Save(sRef, &savedState)).Should(Succeed())
				_, err := fs.Stat(path.Join(CacheDir, string(sRef)))
				Expect(err).ToNot(HaveOccurred())
				Expect(stateCache.Load(sRef, &loadedState)).Should(Succeed())
				Expect(loadedState).Should(Equal(savedState))
			})
		})
		Context("Load non-existent state", func() {
			It("Should fail", func() {
				var loadedState myTestState
				Expect(stateCache.Load(sRef, &loadedState)).ShouldNot(Succeed())
			})
		})
	})

	Describe("Delete State", func() {
		var sRef StateRef
		JustBeforeEach(func() {
			var err error
			sRef, err = stateCache.GetStateRef("mynet", "cid", "net1")
			Expect(err).ToNot(HaveOccurred())
		})

		Context("Delete a saved state", func() {
			It("Should not exist after delete", func() {
				savedState := myTestState{FirstState: "first", SecondState: 42}
				Expect(stateCache.Save(sRef, &savedState)).Should(Succeed())
				_, err := fs.Stat(path.Join(CacheDir, string(sRef)))
				Expect(err).ToNot(HaveOccurred())
				Expect(stateCache.Delete(sRef)).Should(Succeed())
				_, err = fs.Stat(path.Join(CacheDir, string(sRef)))
				Expect(err).To(HaveOccurred())
			})
		})
		Context("Delete a non existent state", func() {
			It("Should Fail", func() {
				altRef, err := stateCache.GetStateRef("alt-mynet", "cid", "net1")
				Expect(err).ToNot(HaveOccurred())
				Expect(stateCache.Delete(altRef)).To(HaveOccurred())
				_, err = fs.Stat(path.Join(CacheDir, string(altRef)))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Input validation for state ref components", func() {
		Context("Network name containing path traversal", func() {
			It("Should reject network name with path traversal components", func() {
				maliciousName := "../../../../tmp/evil"
				_, err := stateCache.GetStateRef(maliciousName, "abc123", "net1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})

			It("Should reject network name with forward slashes", func() {
				_, err := stateCache.GetStateRef("net/with/slashes", "abc123", "net1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})

		})

		Context("Container ID containing path traversal", func() {
			It("Should reject container ID with path traversal components", func() {
				_, err := stateCache.GetStateRef("mynet", "../../../tmp/evil", "net1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})

			It("Should reject container ID with forward slashes", func() {
				_, err := stateCache.GetStateRef("mynet", "cid/with/slashes", "net1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})
		})

		Context("Interface name containing path traversal", func() {
			It("Should reject interface name with path traversal components", func() {
				_, err := stateCache.GetStateRef("mynet", "abc123", "net/../1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})

			It("Should reject interface name with forward slashes", func() {
				_, err := stateCache.GetStateRef("mynet", "abc123", "net/1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state ref component"))
			})
		})
	})
})
