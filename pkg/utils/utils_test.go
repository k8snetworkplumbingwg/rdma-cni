package utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	Describe("IsPCIAddress", func() {
		It("Should accept valid PCI address", func() {
			Expect(IsPCIAddress("0000:04:00.5")).To(BeTrue(),
				"IsPCIAddress input %q: expected acceptance", "0000:04:00.5")
		})
		It("Should accept PCI address with hex digits", func() {
			Expect(IsPCIAddress("abcd:ef:01.7")).To(BeTrue(),
				"IsPCIAddress input %q: expected acceptance", "abcd:ef:01.7")
		})
		It("Should reject empty string", func() {
			Expect(IsPCIAddress("")).To(BeFalse(),
				"IsPCIAddress input %q: expected rejection", "")
		})
		It("Should reject PCI address with invalid function number", func() {
			Expect(IsPCIAddress("0000:04:00.8")).To(BeFalse(),
				"IsPCIAddress input %q: expected rejection", "0000:04:00.8")
		})
		It("Should reject PCI address with extra characters", func() {
			Expect(IsPCIAddress("0000:04:00.5/foo")).To(BeFalse(),
				"IsPCIAddress input %q: expected rejection", "0000:04:00.5/foo")
		})
	})

	Describe("IsAuxDevAddress", func() {
		It("Should accept valid auxiliary device ID", func() {
			Expect(IsAuxDevAddress("mlx5_core.sf.4")).To(BeTrue(),
				"IsAuxDevAddress input %q: expected acceptance", "mlx5_core.sf.4")
		})
		It("Should accept aux device with numeric module name", func() {
			Expect(IsAuxDevAddress("123.abc.0")).To(BeTrue(),
				"IsAuxDevAddress input %q: expected acceptance", "123.abc.0")
		})
		It("Should accept aux device with ID of 0", func() {
			Expect(IsAuxDevAddress("mod.name.0")).To(BeTrue(),
				"IsAuxDevAddress input %q: expected acceptance", "mod.name.0")
		})
		It("Should accept aux device with large ID", func() {
			Expect(IsAuxDevAddress("mod.name.99999")).To(BeTrue(),
				"IsAuxDevAddress input %q: expected acceptance", "mod.name.99999")
		})
		It("Should accept aux device with hyphens and underscores", func() {
			Expect(IsAuxDevAddress("my-mod_1.dev-name_2.42")).To(BeTrue(),
				"IsAuxDevAddress input %q: expected acceptance", "my-mod_1.dev-name_2.42")
		})
		It("Should reject empty string", func() {
			Expect(IsAuxDevAddress("")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "")
		})
		It("Should reject input with only two components", func() {
			Expect(IsAuxDevAddress("mod.name")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "mod.name")
		})
		It("Should reject input with four components", func() {
			Expect(IsAuxDevAddress("a.b.c.1")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "a.b.c.1")
		})
		It("Should reject input with non-numeric ID", func() {
			Expect(IsAuxDevAddress("mod.name.abc")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "mod.name.abc")
		})
		It("Should reject input with trailing newline", func() {
			Expect(IsAuxDevAddress("mod.name.1\n")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "mod.name.1\n")
		})
		It("Should reject input with leading whitespace", func() {
			Expect(IsAuxDevAddress(" mod.name.1")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", " mod.name.1")
		})
		It("Should reject input with path separator", func() {
			Expect(IsAuxDevAddress("mod/name.sf.1")).To(BeFalse(),
				"IsAuxDevAddress input %q: expected rejection", "mod/name.sf.1")
		})
	})
})
