package deeparchiveintegtests

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDeepArchiveNamespaceStore(t *testing.T) {
	_, ok := os.LookupEnv("OPERATOR_IMAGE")
	if !ok {
		t.Skip()
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "deep-archive NamespaceStore Integration Suite")
}
