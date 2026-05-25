package deeparchiveintegtests

import (
	"context"
	"os"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ── deep-archive test constants ────────────────────────────────────────────────

const (
	daEndpoint   = "https://deep-archive.example.com:9000"
	daBucket1    = "deep-archive-target-1"
	daBucket2    = "deep-archive-target-2"
	daSecretName = "deep-archive-secret"
	daStoreName1 = "da-ns1"
	daStoreName2 = "da-ns2"
	daStoreName3 = "da-ns3"
)

// ── CLI integration tests ──────────────────────────────────────────────────────

var _ = Describe("deep-archive NamespaceStore CLI integration tests", func() {

	BeforeAll(func() {
		By("Verifying CLI binary is present at " + CLIPath)
		_, err := os.Stat(CLIPath)
		Expect(err).ToNot(HaveOccurred(), "build the CLI binary first: make noobaa-operator-local")
	})

	BeforeEach(func() {
		options.Namespace = testNamespace
	})

	AfterEach(func() {
		for _, name := range []string{daStoreName1, daStoreName2, daStoreName3} {
			util.KubeDelete(newNamespaceStore(name, nbv1.NamespaceStoreSpec{}))
		}
	})

	Describe("create", func() {

		BeforeEach(func() {
			// The secret is needed by the reconciler when driving the store to Ready.
			ensureDeepArchiveSecret()
		})

		It("should create a valid store and reach Ready phase", func() {
			out, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--endpoint", daEndpoint, "--target-bucket", daBucket1, "--secret-name", daSecretName)
			Expect(err).ToNot(HaveOccurred(), "create failed: %s", out)
			waitForReady(daStoreName1)
		})
	})

	Describe("create: missing / invalid parameters", func() {

		It("should fail when --endpoint is missing", func() {
			_, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--target-bucket", daBucket1, "--secret-name", daSecretName)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when --target-bucket is missing", func() {
			_, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--endpoint", daEndpoint, "--secret-name", daSecretName)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when neither --secret-name nor inline credentials are provided", func() {
			_, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--endpoint", daEndpoint, "--target-bucket", daBucket1)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when the store name argument is omitted", func() {
			// cobra receives 0 positional args; createCommon accesses args[0] → panic / os.Exit
			_, err := RunCLI("namespacestore", "create", "deep-archive",
				"--endpoint", daEndpoint, "--target-bucket", daBucket1, "--secret-name", daSecretName)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when --endpoint is an empty string", func() {
			_, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--endpoint", "", "--target-bucket", daBucket1, "--secret-name", daSecretName)
			Expect(err).To(HaveOccurred())
		})

		It("should fail when --target-bucket is an empty string", func() {
			_, err := RunCLI("namespacestore", "create", "deep-archive", daStoreName1,
				"--endpoint", daEndpoint, "--target-bucket", "", "--secret-name", daSecretName)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("list", func() {

		BeforeEach(func() {
			ensureDeepArchiveSecret()
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)
		})

		It("should list the deep-archive store with correct type and target bucket", func() {
			out, err := RunCLI("namespacestore", "list")
			Expect(err).ToNot(HaveOccurred(), "list failed: %s", out)
			Expect(out).To(ContainSubstring(daStoreName1))
			Expect(out).To(ContainSubstring("deep-archive"))
			Expect(out).To(ContainSubstring(daBucket1))
		})
	})

	Describe("status", func() {

		BeforeEach(func() {
			ensureDeepArchiveSecret()
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)
		})

		It("should print the store spec including endpoint and target bucket", func() {
			out, err := RunCLI("namespacestore", "status", daStoreName1)
			Expect(err).ToNot(HaveOccurred(), "status failed: %s", out)
			Expect(out).To(ContainSubstring("deep-archive"))
			Expect(out).To(ContainSubstring(daBucket1))
			Expect(out).To(ContainSubstring(daEndpoint))
		})

		It("should fail for a store that does not exist", func() {
			_, err := RunCLI("namespacestore", "status", "no-such-store")
			Expect(err).To(HaveOccurred())
		})

		It("should fail when the store name argument is omitted", func() {
			_, err := RunCLI("namespacestore", "status")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("delete", func() {

		BeforeEach(func() {
			ensureDeepArchiveSecret()
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)
		})

		It("should delete the store and make it disappear from K8s", func() {
			out, err := RunCLI("namespacestore", "delete", daStoreName1)
			Expect(err).ToNot(HaveOccurred(), "delete failed: %s", out)

			_, _, err = util.KubeGet(newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{}))
			Expect(err).To(HaveOccurred(), "store should no longer exist after delete")
		})

		It("should fail when deleting a store that does not exist", func() {
			_, err := RunCLI("namespacestore", "delete", "no-such-store")
			Expect(err).To(HaveOccurred())
		})

		It("should fail when the store name argument is omitted", func() {
			_, err := RunCLI("namespacestore", "delete")
			Expect(err).To(HaveOccurred())
		})
	})
})
