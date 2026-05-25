package deeparchiveintegtests

import (
	"context"
	"os/exec"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/bundle"
	"github.com/noobaa/noobaa-operator/v5/pkg/namespacestore"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace = "test"
	// CLIPath mirrors the constant used in pkg/cli/cli_test.go.
	// Relative to this file: pkg/namespacestore/test/integ/deep-archive/ → ../../../../../
	CLIPath = "../../../../../build/_output/bin/noobaa-operator-local"
)

// newNamespaceStore returns a NamespaceStore CR populated from the bundle template.
func newNamespaceStore(name string, spec nbv1.NamespaceStoreSpec) *nbv1.NamespaceStore {
	ns := util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_namespacestore_cr_yaml).(*nbv1.NamespaceStore)
	ns.Name = name
	ns.Namespace = testNamespace
	ns.Spec = spec
	return ns
}

// waitForReady asserts that the named NamespaceStore exists in K8s and that
// the reconciler drives it to the Ready phase within the operator's normal SLA.
func waitForReady(name string) {
	ns := newNamespaceStore(name, nbv1.NamespaceStoreSpec{})
	_, _, err := util.KubeGet(ns)
	Expect(err).ToNot(HaveOccurred(), "NamespaceStore %q not found", name)
	Expect(namespacestore.WaitReady(ns)).To(BeTrue(),
		"NamespaceStore %q did not reach Ready phase", name)
}

// ensureCredentialSecret creates a generic key/value secret in testNamespace if it does
// not already exist. Intended for seeding credentials required by namespacestore tests.
func ensureCredentialSecret(name string, data map[string]string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		StringData: data,
	}
	err := util.KubeClient().Create(context.TODO(), secret)
	if err != nil && !errors.IsAlreadyExists(err) {
		Fail("failed to create credential secret " + name + ": " + err.Error())
	}
}

// newDeepArchiveSpec returns a NamespaceStoreSpec for a deep-archive store.
func newDeepArchiveSpec(endpoint, bucket string) nbv1.NamespaceStoreSpec {
	return nbv1.NamespaceStoreSpec{
		Type: nbv1.NSStoreTypeDeepArchive,
		DeepArchive: &nbv1.DeepArchiveSpec{
			Endpoint:     endpoint,
			TargetBucket: bucket,
			Secret: corev1.SecretReference{
				Name:      daSecretName,
				Namespace: testNamespace,
			},
		},
	}
}

// ensureDeepArchiveSecret seeds the credential secret used by deep-archive tests.
func ensureDeepArchiveSecret() {
	ensureCredentialSecret(daSecretName, map[string]string{
		"AWS_ACCESS_KEY_ID":     "test-access-key",
		"AWS_SECRET_ACCESS_KEY": "test-secret-key",
	})
}

// RunCLI executes the noobaa CLI binary as a subprocess, prepending "-n <testNamespace>"
// to the provided args. It returns the combined stdout+stderr output and an error that
// is non-nil whenever the process exits with a non-zero status (mirrors pkg/cli/cli_test.go).
func RunCLI(args ...string) (string, error) {
	fullArgs := append([]string{"-n", testNamespace}, args...)
	cmd := exec.Command(CLIPath, fullArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
