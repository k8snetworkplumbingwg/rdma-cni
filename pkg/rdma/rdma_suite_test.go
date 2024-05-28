package rdma_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRdma(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RdmaManager Suite")
}
