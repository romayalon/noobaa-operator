package namespacestore

import (
	"fmt"
	"reflect"
	"testing"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/validations"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// const configuration values for the validation checks
const (
	defaultEndPointURI     = "https://127.0.0.1:6443"
	MaximumMountPathLength = 63
)

func TestNamespaceStoreNSFS(t *testing.T) {

	//Valid namespacestore
	defaultNs := getDefaultNSFSNsStore()
	err := validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid namespacestore validation is failed")

	//Pvcname is empty
	defaultNs = getDefaultNSFSNsStore()
	defaultNs.Spec.NSFS.PvcName = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Validation empty pvcName is failed")

	//SubPath is not relative
	defaultNs = getDefaultNSFSNsStore()
	defaultNs.Spec.NSFS.SubPath = "/"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Validation relative subPath %s is failed", defaultNs.Spec.NSFS.SubPath)

	//SubPath contains '..'
	defaultNs = getDefaultNSFSNsStore()
	defaultNs.Spec.NSFS.SubPath = "test/../test2"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Validation relative subPath %s is failed", defaultNs.Spec.NSFS.SubPath)

}

func TestNamespaceStoreS3Compatible(t *testing.T) {

	//Valid namespacestore
	defaultNs := getDefaultS3CompatibleNsStore()
	err := validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid namespacestore validation is failed")

	//Signature version is empty
	defaultNs = getDefaultS3CompatibleNsStore()
	defaultNs.Spec.S3Compatible.SignatureVersion = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Empty sugnature version validation is failed")

	//Valid v2 signature version
	defaultNs = getDefaultS3CompatibleNsStore()
	defaultNs.Spec.S3Compatible.SignatureVersion = "v2"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid sugnature version %s validation is failed", defaultNs.Spec.S3Compatible.SignatureVersion)

	//Ivalid signature version
	defaultNs = getDefaultS3CompatibleNsStore()
	defaultNs.Spec.S3Compatible.SignatureVersion = "v5"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Invalid sugnature version %s validation is failed", defaultNs.Spec.S3Compatible.SignatureVersion)

	//Empty endPoint
	defaultNs = getDefaultS3CompatibleNsStore()
	defaultNs.Spec.S3Compatible.Endpoint = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Empty endPoint validation is failed")
	AssertEqual(t, defaultEndPointURI, defaultNs.Spec.S3Compatible.Endpoint,
		"EndPoint has no the default value, %s : %s", defaultNs.Spec.S3Compatible.Endpoint, defaultEndPointURI)

	//Invalid endPoint
	defaultNs = getDefaultS3CompatibleNsStore()
	defaultNs.Spec.S3Compatible.Endpoint = "hostname:port"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Invalid endPoint %s validation is failed", defaultNs.Spec.S3Compatible.Endpoint)

}

func TestNamespaceStoreAzureBlob(t *testing.T) {
	// Valid namespacestore with secret (Azure blob requires secret; no STS path for namespace store)
	defaultNs := getDefaultAzureBlobNsStore()
	err := validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid Azure blob namespacestore validation failed")

	// AzureBlob spec is nil
	defaultNs = nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeAzureBlob,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-azure"},
	}
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "AzureBlob spec nil should be denied")

	// Empty secret name (namespace store Azure requires secret)
	defaultNs = getDefaultAzureBlobNsStore()
	defaultNs.Spec.AzureBlob.Secret.Name = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Empty secret name for Azure blob namespacestore should be denied")

	// Empty target blob container
	defaultNs = getDefaultAzureBlobNsStore()
	defaultNs.Spec.AzureBlob.TargetBlobContainer = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Empty target blob container should be denied")
}

func TestNamespaceStoreIBMCos(t *testing.T) {

	//Valid namespacestore
	defaultNs := getDefaultIBMCosNsStore()
	err := validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid namespacestore validation is failed")

	//Signature version is empty
	defaultNs = getDefaultIBMCosNsStore()
	defaultNs.Spec.IBMCos.SignatureVersion = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Empty sugnature version validation is failed")

	//Valid v2 signature version
	defaultNs = getDefaultIBMCosNsStore()
	defaultNs.Spec.IBMCos.SignatureVersion = "v2"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid sugnature version %s validation is failed", defaultNs.Spec.IBMCos.SignatureVersion)

	//Ivalid signature version
	defaultNs = getDefaultIBMCosNsStore()
	defaultNs.Spec.IBMCos.SignatureVersion = "v5"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Invalid sugnature version %s validation is failed", defaultNs.Spec.IBMCos.SignatureVersion)

	//Empty endPoint
	defaultNs = getDefaultIBMCosNsStore()
	defaultNs.Spec.IBMCos.Endpoint = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Empty endPoint validation is failed")
	AssertEqual(t, defaultEndPointURI, defaultNs.Spec.IBMCos.Endpoint,
		"EndPoint has no the default value, %s : %s", defaultNs.Spec.IBMCos.Endpoint, defaultEndPointURI)

	//Invalid endPoint
	defaultNs = getDefaultIBMCosNsStore()
	defaultNs.Spec.IBMCos.Endpoint = "hostname:port"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Invalid endPoint %s validation is failed", defaultNs.Spec.IBMCos.Endpoint)

}

func TestNamespaceStoreDeepArchive(t *testing.T) {

	// Valid deep-archive namespacestore
	defaultNs := getDefaultDeepArchiveNsStore()
	err := validations.ValidateNamespaceStore(&defaultNs)
	AssertNotError(t, err, "Valid deep-archive namespacestore validation failed")

	// DeepArchive spec is nil (wrong type spec)
	defaultNs = nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeDeepArchive,
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-deep-archive"},
	}
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Nil DeepArchive spec should be denied")

	// Empty secret name
	defaultNs = getDefaultDeepArchiveNsStore()
	defaultNs.Spec.DeepArchive.Secret.Name = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Empty secret name for deep-archive namespacestore should be denied")

	// Empty target bucket
	defaultNs = getDefaultDeepArchiveNsStore()
	defaultNs.Spec.DeepArchive.TargetBucket = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Empty target bucket for deep-archive namespacestore should be denied")

	// Empty endpoint must be rejected for deep-archive (no defaulting)
	defaultNs = getDefaultDeepArchiveNsStore()
	defaultNs.Spec.DeepArchive.Endpoint = ""
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Empty endpoint for deep-archive namespacestore should be denied")

	// Invalid endpoint (no scheme)
	defaultNs = getDefaultDeepArchiveNsStore()
	defaultNs.Spec.DeepArchive.Endpoint = "hostname:port"
	err = validations.ValidateNamespaceStore(&defaultNs)
	AssertError(t, err, "Invalid endpoint %q should be denied", defaultNs.Spec.DeepArchive.Endpoint)
}

func AssertNotError(t *testing.T, err error, format string, a ...interface{}) {
	if err != nil {
		msg := fmt.Sprintf(format, a...)
		t.Errorf("%s: %s", msg, err)
	}
}

func AssertError(t *testing.T, err error, format string, a ...interface{}) {
	if err == nil {
		msg := fmt.Sprintf(format, a...)
		t.Errorf("%s", msg)
	}
}

func AssertEqual(t *testing.T, actual, expected interface{}, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)

	if (actual == nil || expected == nil) && actual != expected {
		t.Errorf("%s", msg)
		return
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s", msg)
	}
}

func getDefaultIBMCosNsStore() nbv1.NamespaceStore {
	return nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeIBMCos,
			IBMCos: &nbv1.IBMCosSpec{
				SignatureVersion: nbv1.S3SignatureVersionV4,
				Endpoint:         defaultEndPointURI,
				Secret: corev1.SecretReference{
					Name:      "secret-name",
					Namespace: "namespace",
				},
				TargetBucket: "some-target-bucket",
			},
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test1"},
	}
}

func getDefaultS3CompatibleNsStore() nbv1.NamespaceStore {
	return nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeS3Compatible,
			S3Compatible: &nbv1.S3CompatibleSpec{
				SignatureVersion: nbv1.S3SignatureVersionV4,
				Endpoint:         defaultEndPointURI,
				Secret: corev1.SecretReference{
					Name:      "secret-name",
					Namespace: "namespace",
				},
				TargetBucket: "some-target-bucket",
			},
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test1"},
	}
}

func getDefaultNSFSNsStore() nbv1.NamespaceStore {
	return nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeNSFS,
			NSFS: &nbv1.NSFSSpec{
				PvcName: "pv-pool",
				SubPath: "subpath/",
			},
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test1"},
	}
}

func getDefaultDeepArchiveNsStore() nbv1.NamespaceStore {
	return nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeDeepArchive,
			DeepArchive: &nbv1.DeepArchiveSpec{
				Endpoint:     defaultEndPointURI,
				TargetBucket: "archive-bucket",
				Secret: corev1.SecretReference{
					Name:      "archive-secret",
					Namespace: "namespace",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-deep-archive", Namespace: "namespace"},
	}
}

func getDefaultAzureBlobNsStore() nbv1.NamespaceStore {
	return nbv1.NamespaceStore{
		Spec: nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeAzureBlob,
			AzureBlob: &nbv1.AzureBlobSpec{
				TargetBlobContainer: "azure-container",
				Secret: corev1.SecretReference{
					Name:      "azure-secret",
					Namespace: "namespace",
				},
			},
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-azure"},
	}
}
