package tlsintegtests

import (
	"context"
	"fmt"
	"time"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace = "test"
)

func getEndpointDeployment() *appsv1.Deployment {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "noobaa-endpoint",
			Namespace: options.Namespace,
		},
	}
	Expect(util.KubeCheck(deploy)).To(BeTrue(), "endpoint deployment should exist")
	return deploy
}

func getEndpointContainerEnv(deploy *appsv1.Deployment) []corev1.EnvVar {
	for _, c := range deploy.Spec.Template.Spec.Containers {
		if c.Name == "endpoint" {
			return c.Env
		}
	}
	Fail("endpoint container not found in deployment")
	return nil
}

func findEnvVar(env []corev1.EnvVar, name string) (string, bool) {
	for _, e := range env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func patchNooBaaTLS(apiServerTLS nbv1.TLSSecuritySpec) {
	noobaa := &nbv1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "noobaa",
			Namespace: options.Namespace,
		},
	}
	Expect(util.KubeCheck(noobaa)).To(BeTrue(), "NooBaa CR should exist")

	noobaa.Spec.Security.APIServerSecurity = apiServerTLS
	Expect(util.KubeClient().Update(context.TODO(), noobaa)).To(Succeed(),
		"should update NooBaa CR with TLS settings")
}

func waitForEndpointRollout(timeout time.Duration) {
	Eventually(func() bool {
		deploy := getEndpointDeployment()
		return deploy.Status.UpdatedReplicas == *deploy.Spec.Replicas &&
			deploy.Status.AvailableReplicas == *deploy.Spec.Replicas
	}, timeout, 5*time.Second).Should(BeTrue(), "endpoint deployment should roll out")
}

var _ = Describe("TLS configuration integration", func() {

	BeforeEach(func() {
		options.Namespace = namespace
	})

	Context("When APIServerSecurity TLS is configured on the NooBaa CR", func() {

		It("Should propagate TLSMinVersion as TLS_MIN_VERSION env var", func() {
			tlsVersion := nbv1.TLSVersionTLS13
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSMinVersion: &tlsVersion,
			})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			val, found := findEnvVar(env, "TLS_MIN_VERSION")
			Expect(found).To(BeTrue(), "TLS_MIN_VERSION should be set")
			Expect(val).To(Equal("TLSv1.3"))
		})

		It("Should propagate TLSCiphers as TLS_CIPHERS env var", func() {
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSCiphers: []string{
					"TLS_AES_128_GCM_SHA256",
					"TLS_AES_256_GCM_SHA384",
				},
			})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			val, found := findEnvVar(env, "TLS_CIPHERS")
			Expect(found).To(BeTrue(), "TLS_CIPHERS should be set")
			Expect(val).To(Equal("TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384"))
		})

		It("Should propagate TLSGroups as TLS_GROUPS env var", func() {
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSGroups: []nbv1.TLSGroup{nbv1.TLSGroupX25519, nbv1.TLSGroupSecp256r1},
			})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			val, found := findEnvVar(env, "TLS_GROUPS")
			Expect(found).To(BeTrue(), "TLS_GROUPS should be set")
			Expect(val).To(Equal("X25519:secp256r1"))
		})

		It("Should propagate all TLS fields together", func() {
			tlsVersion := nbv1.TLSVersionTLS13
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSMinVersion: &tlsVersion,
				TLSCiphers:    []string{"TLS_AES_256_GCM_SHA384"},
				TLSGroups:     []nbv1.TLSGroup{nbv1.TLSGroupX25519MLKEM768, nbv1.TLSGroupX25519},
			})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			val, found := findEnvVar(env, "TLS_MIN_VERSION")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("TLSv1.3"))

			val, found = findEnvVar(env, "TLS_CIPHERS")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("TLS_AES_256_GCM_SHA384"))

			val, found = findEnvVar(env, "TLS_GROUPS")
			Expect(found).To(BeTrue())
			Expect(val).To(Equal("X25519MLKEM768:X25519"))
		})

		It("Should not set TLS env vars when APIServerSecurity is empty", func() {
			patchNooBaaTLS(nbv1.TLSSecuritySpec{})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			_, found := findEnvVar(env, "TLS_MIN_VERSION")
			Expect(found).To(BeFalse(), "TLS_MIN_VERSION should not be set")

			_, found = findEnvVar(env, "TLS_CIPHERS")
			Expect(found).To(BeFalse(), "TLS_CIPHERS should not be set")

			_, found = findEnvVar(env, "TLS_GROUPS")
			Expect(found).To(BeFalse(), "TLS_GROUPS should not be set")
		})

		It("Should use APIServerSecurity for endpoint env vars", func() {
			apiVersion := nbv1.TLSVersionTLS12
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSMinVersion: &apiVersion,
				TLSCiphers:    []string{"ECDHE-RSA-AES128-GCM-SHA256"},
			})

			waitForEndpointRollout(3 * time.Minute)

			deploy := getEndpointDeployment()
			env := getEndpointContainerEnv(deploy)

			val, found := findEnvVar(env, "TLS_MIN_VERSION")
			Expect(found).To(BeTrue(),
				"TLS_MIN_VERSION should be set from APIServerSecurity")
			Expect(val).To(Equal("TLSv1.2"))

			val, found = findEnvVar(env, "TLS_CIPHERS")
			Expect(found).To(BeTrue(),
				"TLS_CIPHERS should be set from APIServerSecurity")
			Expect(val).To(Equal("ECDHE-RSA-AES128-GCM-SHA256"))
		})
	})

	Context("NooBaa Core TLS negotiation (requires endpoint pod access)", func() {

		It("Should enforce TLS 1.3 minimum when configured", func() {
			tlsVersion := nbv1.TLSVersionTLS13
			patchNooBaaTLS(nbv1.TLSSecuritySpec{
				TLSMinVersion: &tlsVersion,
			})

			waitForEndpointRollout(3 * time.Minute)

			pods := &corev1.PodList{}
			Expect(util.KubeList(pods,
				client.InNamespace(options.Namespace),
				client.MatchingLabels{"noobaa-s3": "noobaa"},
			)).To(BeTrue())
			Expect(pods.Items).ToNot(BeEmpty(), "endpoint pods should exist")

			pod := pods.Items[0]
			found := false
			for _, c := range pod.Spec.Containers {
				if c.Name != "endpoint" {
					continue
				}
				for _, e := range c.Env {
					if e.Name == "TLS_MIN_VERSION" && e.Value == "TLSv1.3" {
						found = true
					}
				}
			}
			Expect(found).To(BeTrue(),
				fmt.Sprintf("endpoint pod %s should have TLS_MIN_VERSION=TLSv1.3", pod.Name))
		})
	})

	// NOTE: testssl.sh end-to-end tests have been moved to noobaa-core
	// (src/test/integration_tests/test_tls_config.js) where the endpoint is
	// already running in-process during coretests — no manual port-forward needed.
})
