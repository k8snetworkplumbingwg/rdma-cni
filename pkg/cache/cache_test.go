package cache

import (
	"path"

	. "github.com/onsi/ginkgo"
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
				Expect(stateCache.GetStateRef("myNet", "containerUniqueIdentifier", "net1")).To(BeEquivalentTo("myNet-containerUniqueIdentifier-net1"))
			})
		})
	})

	Describe("Save and Load State", func() {
		var sRef StateRef
		JustBeforeEach(func() {
			sRef = stateCache.GetStateRef("mynet", "cid", "net1")
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
			sRef = stateCache.GetStateRef("mynet", "cid", "net1")
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
				altRef := stateCache.GetStateRef("alt-mynet", "cid", "net1")
				Expect(stateCache.Delete(altRef)).To(HaveOccurred())
				_, err := fs.Stat(path.Join(CacheDir, string(altRef)))
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
