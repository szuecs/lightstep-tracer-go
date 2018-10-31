package testtimex_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTestTimex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TestTimex Suite")
}
