package deeparchiveintegtests

import (
	"context"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("deep-archive NamespaceStore CR integration tests", func() {

	BeforeEach(func() {
		options.Namespace = testNamespace
	})

	AfterEach(func() {
		for _, name := range []string{daStoreName1, daStoreName2, daStoreName3} {
			util.KubeDelete(newNamespaceStore(name, nbv1.NamespaceStoreSpec{}))
		}
	})

	Context("missing secret name", func() {
		It("should be denied by the admission webhook", func() {
			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{
				Type: nbv1.NSStoreTypeDeepArchive,
				DeepArchive: &nbv1.DeepArchiveSpec{
					Endpoint:     daEndpoint,
					TargetBucket: daBucket1,
					Secret:       corev1.SecretReference{Name: "", Namespace: testNamespace},
				},
			})
			err := util.KubeClient().Create(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("please provide secret name"))
		})
	})

	Context("missing target bucket", func() {
		It("should be denied by the admission webhook", func() {
			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{
				Type: nbv1.NSStoreTypeDeepArchive,
				DeepArchive: &nbv1.DeepArchiveSpec{
					Endpoint:     daEndpoint,
					TargetBucket: "",
					Secret:       corev1.SecretReference{Name: daSecretName, Namespace: testNamespace},
				},
			})
			err := util.KubeClient().Create(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("please provide target bucket"))
		})
	})

	Context("missing endpoint", func() {
		It("should be denied by the admission webhook", func() {
			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{
				Type: nbv1.NSStoreTypeDeepArchive,
				DeepArchive: &nbv1.DeepArchiveSpec{
					Endpoint:     "",
					TargetBucket: daBucket1,
					Secret:       corev1.SecretReference{Name: daSecretName, Namespace: testNamespace},
				},
			})
			err := util.KubeClient().Create(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("please provide endpoint"))
		})
	})

	Context("nil DeepArchive spec with deep-archive type", func() {
		It("should be denied by the admission webhook", func() {
			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{
				Type: nbv1.NSStoreTypeDeepArchive,
			})
			err := util.KubeClient().Create(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("DeepArchive spec must be provided"))
		})
	})

	Context("all required fields empty", func() {
		It("should be denied; secret name is the first failing check", func() {
			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{
				Type: nbv1.NSStoreTypeDeepArchive,
				DeepArchive: &nbv1.DeepArchiveSpec{
					Endpoint:     "",
					TargetBucket: "",
					Secret:       corev1.SecretReference{Name: "", Namespace: testNamespace},
				},
			})
			err := util.KubeClient().Create(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("please provide secret name"))
		})
	})

	Context("duplicate endpoint+targetBucket", func() {
		BeforeEach(func() {
			ensureDeepArchiveSecret()
		})

		It("should deny the second store", func() {
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)

			err := util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName2, newDeepArchiveSpec(daEndpoint, daBucket1)))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring(daStoreName1))
			Expect(err.Error()).To(ContainSubstring(daBucket1))
		})

		It("should allow a second store with a different bucket", func() {
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)

			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName2, newDeepArchiveSpec(daEndpoint, daBucket2)))).ToNot(HaveOccurred())
			waitForReady(daStoreName2)
		})
	})

	Context("update: target bucket change", func() {
		BeforeEach(func() {
			ensureDeepArchiveSecret()
		})

		It("should be denied by the admission webhook", func() {
			Expect(util.KubeClient().Create(context.TODO(), newNamespaceStore(daStoreName1, newDeepArchiveSpec(daEndpoint, daBucket1)))).ToNot(HaveOccurred())
			waitForReady(daStoreName1)

			ns := newNamespaceStore(daStoreName1, nbv1.NamespaceStoreSpec{})
			_, _, err := util.KubeGet(ns)
			Expect(err).ToNot(HaveOccurred())
			ns.Spec.DeepArchive.TargetBucket = daBucket2
			err = util.KubeClient().Update(context.TODO(), ns)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("admission webhook"))
			Expect(err.Error()).To(ContainSubstring("Changing a NamespaceStore target bucket is unsupported"))
		})
	})
})
