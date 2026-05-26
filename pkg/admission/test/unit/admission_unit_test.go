package admissionunittests

import (
	"fmt"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/bucketclass"
	"github.com/noobaa/noobaa-operator/v5/pkg/bundle"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	"github.com/noobaa/noobaa-operator/v5/pkg/validations"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Pointerify(number int) *int {
	return &number
}

var _ = Describe("BackingStore admission unit tests", func() {

	var (
		bs  *nbv1.BackingStore
		err error
	)

	BeforeEach(func() {
		bs = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_backingstore_cr_yaml).(*nbv1.BackingStore)
		bs.Name = "bs-name"
		bs.Namespace = "test"

	})

	Describe("Validate create operations", func() {
		Describe("General backingstore validations", func() {
			Context("Invalid spec for declared type", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
					}
					err = validations.ValidateBSInValidSpec(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("AWSS3 spec must be provided for aws-s3 type BackingStore"))
				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSInValidSpec(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("Empty secret name", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the Backingstore, please provide a valid ARN or secret name"))
				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "full-secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Empty Target Bucket", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "",
						},
					}
					err = validations.ValidateBSEmptyTargetBucket(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the Backingstore, please provide target bucket"))
				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
						},
					}
					err = validations.ValidateBSEmptyTargetBucket(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Invalid store type", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: "invalid",
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}

					err = validations.ValidateBackingStore(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Invalid Backingstore type, please provide a valid Backingstore type"))
				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}

					err = validations.ValidateBackingStore(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
		Describe("Azure Blob backingstore", func() {
			Context("Invalid spec for declared type", func() {
				It("Should Deny when AzureBlob spec is nil", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
					}
					err = validations.ValidateBSInValidSpec(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("AzureBlob spec must be provided for azure-blob type BackingStore"))
				})
				It("Should Allow when AzureBlob spec is provided", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret: corev1.SecretReference{
								Name:      "azure-secret",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSInValidSpec(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Empty secret name", func() {
				It("Should Deny when secret is empty and no Azure STS credentials", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret: corev1.SecretReference{
								Name:      "",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the Backingstore, please provide secret name or Azure STS clientId"))
				})
				It("Should Allow when secret is empty but Azure STS clientId is set (tenantId optional)", func() {
					clientID := "azure-client-id"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
				It("Should Allow when secret is empty with clientId and tenantId", func() {
					clientID := "azure-client-id"
					tenantID := "azure-tenant-id"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
							TenantId:           &tenantID,
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
				It("Should Allow when secret name is provided", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret: corev1.SecretReference{
								Name:      "azure-secret",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptySecretName(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("ValidateAzureSTSCredsPresent (tenant, account name, clientID in secret/flags)", func() {
				targetBlobContainer := "my-container"
				accountName := "myaccount"
				tenantID := "tenant-id"
				clientID := "client-id"
				It("Should Deny when target blob container is missing", func() {
					err = validations.ValidateAzureSTSCredsPresent(nil, &accountName, &tenantID, &clientID)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Target blob container is required"))
				})
				It("Should Deny when account name is missing", func() {
					err = validations.ValidateAzureSTSCredsPresent(&targetBlobContainer, nil, &tenantID, &clientID)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Azure storage account name is required"))
				})
				It("Should Deny when tenant ID is missing", func() {
					err = validations.ValidateAzureSTSCredsPresent(&targetBlobContainer, &accountName, nil, &clientID)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Azure tenant ID is required"))
				})
				It("Should Deny when client ID is missing", func() {
					err = validations.ValidateAzureSTSCredsPresent(&targetBlobContainer, &accountName, &tenantID, nil)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Azure client ID is required"))
				})
				It("Should Allow when targetBlobContainer, accountName, tenantID and clientID are present", func() {
					err = validations.ValidateAzureSTSCredsPresent(&targetBlobContainer, &accountName, &tenantID, &clientID)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Empty target blob container", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "",
							Secret: corev1.SecretReference{
								Name:      "azure-secret",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptyTargetBucket(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the Backingstore, please provide target bucket"))
				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret: corev1.SecretReference{
								Name:      "azure-secret",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateBSEmptyTargetBucket(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Full Azure STS backingstore validation", func() {
				It("Should Allow Azure STS with clientId only", func() {
					clientID := "azure-client-id"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
						},
					}
					err = validations.ValidateBackingStore(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
				It("Should Allow Azure STS with clientId and tenantId", func() {
					clientID := "azure-client-id"
					tenantID := "azure-tenant-id"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
							TenantId:           &tenantID,
						},
					}
					err = validations.ValidateBackingStore(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
				It("Should Allow Azure STS with clientId, tenantId, subscriptionId and resourcegroupId", func() {
					clientID := "azure-client-id"
					tenantID := "azure-tenant-id"
					subscriptionID := "azure-subscription-id"
					resourceGroupID := "azure-resource-group"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:              corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:            &clientID,
							TenantId:            &tenantID,
							SubscriptionId:     &subscriptionID,
							ResourcegroupId:    &resourceGroupID,
						},
					}
					err = validations.ValidateBackingStore(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
		Describe("Pvpool backingstore", func() {
			Context("Resource name too long", func() {
				It("Should Deny", func() {
					bs.Name = "pvpool-too-long-name-should-fail-after-exceeding-43-characters"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
					}

					err = validations.ValidatePvpoolNameLength(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Unsupported BackingStore name length, please provide a name shorter then 43 characters"))
				})

				It("Should Allow", func() {
					bs.Name = "pvpool-not-too-long-name"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
					}

					err = validations.ValidatePvpoolNameLength(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Minimum volume count", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: -5,
						},
					}

					err = validations.ValidateMinVolumeCount(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Unsupported volume count, the minimum supported volume count is 1"))

				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 5,
						},
					}

					err = validations.ValidateMinVolumeCount(*bs)
					Ω(err).ShouldNot(HaveOccurred())

				})
			})
			Context("Maximum volume count", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 25,
						},
					}

					err = validations.ValidateMaxVolumeCount(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Unsupported volume count, the maximum supported volume count is 20"))

				})
				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 15,
						},
					}

					err = validations.ValidateMaxVolumeCount(*bs)
					Ω(err).ShouldNot(HaveOccurred())

				})
			})
			Context("Minimum volume size", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							VolumeResources: &corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: *resource.NewScaledQuantity(int64(5), resource.Giga),
								},
							},
						},
					}

					err = validations.ValidatePvpoolMinVolSize(*bs)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Invalid volume size, minimum volume size is 16Gi"))
				})

				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							VolumeResources: &corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: *resource.NewScaledQuantity(int64(20), resource.Giga),
								},
							},
						},
					}

					err = validations.ValidatePvpoolMinVolSize(*bs)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
		Describe("S3 Compatible backingstore", func() {
			Context("Invalid signature version", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeS3Compatible,
						S3Compatible: &nbv1.S3CompatibleSpec{
							SignatureVersion: "v5",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}

					err = validations.ValidateSigVersion(bs.Spec.S3Compatible.SignatureVersion)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Invalid S3 compatible Backingstore signature version, please choose either v2/v4"))
				})

				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeS3Compatible,
						S3Compatible: &nbv1.S3CompatibleSpec{
							SignatureVersion: "v4",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}

					err = validations.ValidateSigVersion(bs.Spec.S3Compatible.SignatureVersion)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Validate update operations", func() {
		var (
			updatedBS *nbv1.BackingStore
		)

		BeforeEach(func() {
			updatedBS = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_backingstore_cr_yaml).(*nbv1.BackingStore)
			updatedBS.Name = "bs-name"
			updatedBS.Namespace = "test"
		})

		Describe("Pvpool backingstore", func() {
			Context("Scale down node count", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 10,
						},
					}

					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 15,
						},
					}

					err = validations.ValidatePvpoolScaleDown(*bs, *updatedBS)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Scaling down the number of nodes is not currently supported"))
				})

				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 15,
						},
					}

					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypePVPool,
						PVPool: &nbv1.PVPoolSpec{
							NumVolumes: 10,
						},
					}

					err = validations.ValidatePvpoolScaleDown(*bs, *updatedBS)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
		Describe("Cloud backingstore", func() {
			Context("Update target bucket", func() {
				It("Should Deny", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
						},
					}

					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-other-bucket",
						},
					}

					err = validations.ValidateTargetBSBucketChange(*bs, *updatedBS)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Changing a Backingstore target bucket is unsupported"))
				})

				It("Should Allow", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "same-target-bucket",
						},
					}

					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "same-target-bucket",
						},
					}

					err = validations.ValidateTargetBSBucketChange(*bs, *updatedBS)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Update Azure Blob target container", func() {
				It("Should Deny when target blob container is changed", func() {
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "original-container",
							Secret: corev1.SecretReference{Name: "azure-secret", Namespace: "test"},
						},
					}
					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "different-container",
							Secret: corev1.SecretReference{Name: "azure-secret", Namespace: "test"},
						},
					}
					err = validations.ValidateTargetBSBucketChange(*bs, *updatedBS)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Changing a Backingstore target bucket is unsupported"))
				})
				It("Should Allow when target blob container is unchanged", func() {
					clientID := "azure-client-id"
					tenantID := "azure-tenant-id"
					bs.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "same-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
							TenantId:           &tenantID,
						},
					}
					updatedBS.Spec = nbv1.BackingStoreSpec{
						Type: nbv1.StoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "same-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
							ClientId:           &clientID,
							TenantId:           &tenantID,
						},
					}
					err = validations.ValidateTargetBSBucketChange(*bs, *updatedBS)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("NamespaceStore admission unit tests", func() {

	var (
		ns  *nbv1.NamespaceStore
		err error
	)

	BeforeEach(func() {
		ns = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_namespacestore_cr_yaml).(*nbv1.NamespaceStore)
		ns.Name = "ns-name"
		ns.Namespace = "test"
	})

	Describe("Validate create operations", func() {
		Describe("General namespacestore validations", func() {
			Context("Invalid spec for declared type", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
					}
					err = validations.ValidateNSInValidSpec(*ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("AWSS3 spec must be provided for aws-s3 type Namespacestore"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateNSInValidSpec(*ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("Empty secret name", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateNSEmptySecretName(*ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the NamespaceStore, please provide a valid ARN or secret name"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateNSEmptySecretName(*ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Empty Target Bucket", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "",
						},
					}
					err = validations.ValidateNSEmptyTargetBucket(*ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the namespacestore, please provide target bucket"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
						},
					}
					err = validations.ValidateNSEmptyTargetBucket(*ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Azure Blob namespacestore", func() {
				It("Should Deny when AzureBlob spec is nil", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAzureBlob,
					}
					err = validations.ValidateNSInValidSpec(*ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("AzureBlob spec must be provided for azure-blob type Namespacestore"))
				})
				It("Should Deny when secret is empty (Azure namespacestore requires secret)", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:             corev1.SecretReference{Name: "", Namespace: "test"},
						},
					}
					err = validations.ValidateNSEmptySecretName(*ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Failed creating the namespacestore: secret name (secret must contain AccountName and AccountKey) and target blob container are both required"))
				})
				It("Should Allow when AzureBlob spec with secret is provided", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAzureBlob,
						AzureBlob: &nbv1.AzureBlobSpec{
							TargetBlobContainer: "my-container",
							Secret:              corev1.SecretReference{Name: "azure-secret", Namespace: "test"},
						},
					}
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Invalid store type", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: "invalid",
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Invalid Namespacestore type, please provide a valid Namespacestore type"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
		Describe("NSFS validations", func() {
			Context("Empty pvc name", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "",
						},
					}
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("PvcName must not be empty"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
						},
					}
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Invalid SubPath", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
							SubPath: "/path",
						},
					}
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("SubPath /path must be a relative path"))
				})
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
							SubPath: "../path",
						},
					}
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("SubPath ../path must not contain '..'"))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
							SubPath: "valid/sub/path",
						},
					}
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
			Context("Validate too long mount path", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
							SubPath: "valid/sub/path",
						},
					}
					ns.Name = "nsfs-too-long-name-should-fail-after-exceeding-63-characters"
					mountPath := "/nsfs/" + ns.Name
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal(fmt.Sprintf("MountPath %v must be no more than 63 characters", mountPath)))
				})
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeNSFS,
						NSFS: &nbv1.NSFSSpec{
							PvcName: "pvc-name",
							SubPath: "valid/sub/path",
						},
					}
					ns.Name = "nsfs-not-too-long-name"
					err = validations.ValidateNsStoreNSFS(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Describe("S3-compatible namespacestore", func() {
			Context("signature version and non-secure endpoint", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeS3Compatible,
						S3Compatible: &nbv1.S3CompatibleSpec{
							Endpoint:         "http://test.com",
							SignatureVersion: "v4",
							TargetBucket:     "test",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					ns.Name = "nsfs-signV4-http-endpoint"
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Non-secure endpoint works only with signature-version \"v2\". Please select signature version v2 for namespacestore"))
				})

				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeS3Compatible,
						S3Compatible: &nbv1.S3CompatibleSpec{
							Endpoint:         "http://test.com",
							SignatureVersion: "v2",
							TargetBucket:     "test",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					ns.Name = "nsfs-signV4-http-endpoint"
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})

			Context("signature version and secure endpoint", func() {
				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeS3Compatible,
						S3Compatible: &nbv1.S3CompatibleSpec{
							Endpoint:         "https://test.com",
							SignatureVersion: "v4",
							TargetBucket:     "test",
							Secret: corev1.SecretReference{
								Name:      "secret-name",
								Namespace: "test",
							},
						},
					}
					ns.Name = "nsfs-signV4-https-endpoint"
					err = validations.ValidateNamespaceStore(ns)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Validate update operations", func() {
		var (
			updatedNS *nbv1.NamespaceStore
		)

		BeforeEach(func() {
			updatedNS = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_namespacestore_cr_yaml).(*nbv1.NamespaceStore)
			updatedNS.Name = "ns-name"
			updatedNS.Namespace = "test"
		})

		Describe("Cloud namespacestore", func() {
			Context("Update target bucket", func() {
				It("Should Deny", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-target-bucket",
						},
					}

					updatedNS.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "some-other-bucket",
						},
					}

					err = validations.ValidateTargetNSBucketChange(*ns, *updatedNS)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Changing a NamespaceStore target bucket is unsupported"))
				})

				It("Should Allow", func() {
					ns.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "same-target-bucket",
						},
					}

					updatedNS.Spec = nbv1.NamespaceStoreSpec{
						Type: nbv1.NSStoreTypeAWSS3,
						AWSS3: &nbv1.AWSS3Spec{
							TargetBucket: "same-target-bucket",
						},
					}

					err = validations.ValidateTargetNSBucketChange(*ns, *updatedNS)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("BucketClass admission unit tests", func() {
	var (
		bc  *nbv1.BucketClass
		err error
	)

	BeforeEach(func() {
		bc = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_bucketclass_cr_yaml).(*nbv1.BucketClass)
		bc.Name = "bc-name"
		bc.Namespace = "test"
	})

	Describe("Validate create operations", func() {
		Context("Unsupported tiers number", func() {
			It("Should Deny", func() {
				bc.Spec.PlacementPolicy = &nbv1.PlacementPolicy{
					Tiers: []nbv1.Tier{{
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}, {
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}, {
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}},
				}
				err = validations.ValidateTiersNumber(bc.Spec.PlacementPolicy.Tiers)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("unsupported number of tiers, bucketclass supports only 1 or 2 tiers"))
			})
			It("Should Allow", func() {
				bc.Spec.PlacementPolicy = &nbv1.PlacementPolicy{
					Tiers: []nbv1.Tier{{
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}, {
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}},
				}
				err = validations.ValidateTiersNumber(bc.Spec.PlacementPolicy.Tiers)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
		Context("Validate quota", func() {
			It("Should Deny", func() {
				bc.Spec.Quota = &nbv1.Quota{
					MaxSize:    "2Gi",
					MaxObjects: "-1",
				}
				err = validations.ValidateQuotaConfig(bc.Name, bc.Spec.Quota)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("ob \"bc-name\" validation error: invalid maxObjects value. O or any positive number "))
			})
			It("Should Deny", func() {
				bc.Spec.Quota = &nbv1.Quota{
					MaxSize:    "-1Gi",
					MaxObjects: "10",
				}
				err = validations.ValidateQuotaConfig(bc.Name, bc.Spec.Quota)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("ob \"bc-name\" validation error: invalid obcMaxSizeValue value: min 1Gi, max 1023Pi, 0 to remove quota"))
			})
			It("Should Allow", func() {
				bc.Spec.Quota = &nbv1.Quota{
					MaxSize:    "20Gi",
					MaxObjects: "10",
				}
				err = validations.ValidateQuotaConfig(bc.Name, bc.Spec.Quota)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
		Context("Validate archivePolicy", func() {
			It("Should Allow when archivePolicy is nil", func() {
				bc.Spec.ArchivePolicy = nil
				err = validations.ValidateArchivePolicy(bc)
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("Should Deny when archivePolicy is set without placementPolicy", func() {
				bc.Spec.PlacementPolicy = nil
				bc.Spec.ArchivePolicy = &nbv1.ArchivePolicy{
					DeepArchiveResource: "my-deep-archive-ns",
				}
				err = validations.ValidateArchivePolicy(bc)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("placementPolicy is required when archivePolicy is set"))
			})
			It("Should Deny when archivePolicy is set with only namespacePolicy (no placementPolicy)", func() {
				bc.Spec.PlacementPolicy = nil
				bc.Spec.NamespacePolicy = &nbv1.NamespacePolicy{
					Type: nbv1.NSBucketClassTypeSingle,
					Single: &nbv1.SingleNamespacePolicy{
						Resource: "my-ns",
					},
				}
				bc.Spec.ArchivePolicy = &nbv1.ArchivePolicy{
					DeepArchiveResource: "my-deep-archive-ns",
				}
				err = validations.ValidateArchivePolicy(bc)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("placementPolicy is required when archivePolicy is set"))
			})
			It("Should deny when archivePolicy references a NamespaceStore that does not exist", func() {
				bc.Spec.PlacementPolicy = &nbv1.PlacementPolicy{
					Tiers: []nbv1.Tier{{
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}},
				}
				bc.Spec.ArchivePolicy = &nbv1.ArchivePolicy{
					DeepArchiveResource: "my-deep-archive-ns",
				}
				// Structural check (placementPolicy present) and uniqueness check pass.
				// Then ValidateArchivePolicyNS fires: in a unit test environment there is
				// no K8s API, so the NamespaceStore lookup fails with "does not exist".
				err = validations.ValidateArchivePolicy(bc)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("my-deep-archive-ns"))
				Expect(err.Error()).To(ContainSubstring("does not exist"))
			})
			It("Should Allow when archivePolicy is set with empty DeepArchiveResource", func() {
				bc.Spec.PlacementPolicy = &nbv1.PlacementPolicy{
					Tiers: []nbv1.Tier{{
						Placement:     "",
						BackingStores: []string{"bs-name"},
					}},
				}
				bc.Spec.ArchivePolicy = &nbv1.ArchivePolicy{
					DeepArchiveResource: "",
				}
				err = validations.ValidateArchivePolicy(bc)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("BucketClass CLI populate unit tests", func() {
	Describe("PopulatePlacementBucketClass --deep-archive-resource flag", func() {
		var (
			cmd  = bucketclass.CmdCreatePlacementBucketClass()
			spec *nbv1.BucketClassSpec
		)

		BeforeEach(func() {
			cmd = bucketclass.CmdCreatePlacementBucketClass()
			spec = &nbv1.BucketClassSpec{
				PlacementPolicy: &nbv1.PlacementPolicy{Tiers: []nbv1.Tier{}},
			}
			Expect(cmd.Flags().Set("backingstores", "bs1")).To(Succeed())
		})

		Context("flag is provided", func() {
			It("Should set ArchivePolicy on the spec", func() {
				Expect(cmd.Flags().Set("deep-archive-resource", "my-da-ns")).To(Succeed())
				nsArr, bsArr := bucketclass.PopulatePlacementBucketClass(cmd, spec)
				Expect(spec.ArchivePolicy).NotTo(BeNil())
				Expect(spec.ArchivePolicy.DeepArchiveResource).To(Equal("my-da-ns"))
				Expect(nsArr).To(ContainElement("my-da-ns"))
				Expect(bsArr).To(ContainElement("bs1"))
			})
		})

		Context("flag is not provided", func() {
			It("Should leave ArchivePolicy nil", func() {
				nsArr, bsArr := bucketclass.PopulatePlacementBucketClass(cmd, spec)
				Expect(spec.ArchivePolicy).To(BeNil())
				Expect(nsArr).To(BeEmpty())
				Expect(bsArr).To(ContainElement("bs1"))
			})
		})
	})
})

var _ = Describe("NooBaaAccount admission unit tests", func() {

	var (
		na  *nbv1.NooBaaAccount
		err error
	)

	BeforeEach(func() {
		na = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_noobaaaccount_cr_yaml).(*nbv1.NooBaaAccount)
		na.Name = "na-name"
		na.Namespace = "test"
	})

	Describe("Validate create operations", func() {
		Describe("Noobaaaccount NSFS create validations", func() {
			Context("UID and GID are a whole positive number", func() {
				It("Should Deny Negative UID", func() {
					na.Spec = nbv1.NooBaaAccountSpec{
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(-3),
							GID:            Pointerify(2),
							NewBucketsPath: "/",
							NsfsOnly:       true,
						},
					}
					err = validations.ValidateNSFSConfig(*na)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("UID must be a whole positive number"))
				})
				It("Should Deny Negative GID", func() {
					na.Spec = nbv1.NooBaaAccountSpec{
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(3),
							GID:            Pointerify(-2),
							NewBucketsPath: "/",
							NsfsOnly:       true,
						},
					}
					err = validations.ValidateNSFSConfig(*na)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("GID must be a whole positive number"))
				})
				It("Should Allow", func() {
					na.Spec = nbv1.NooBaaAccountSpec{
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(3),
							GID:            Pointerify(2),
							NewBucketsPath: "/",
							NsfsOnly:       true,
						},
					}
					err = validations.ValidateNSFSConfig(*na)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

	Describe("Validate update operations", func() {
		var (
			updatedNA *nbv1.NooBaaAccount
		)

		BeforeEach(func() {
			updatedNA = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_noobaaaccount_cr_yaml).(*nbv1.NooBaaAccount)
			updatedNA.Name = "na-name"
			updatedNA.Namespace = "test"
		})

		Describe("Noobaaaccount NSFS update validations", func() {
			Context("Remove NSFSAccountConfig from NooBaaAccountSpec", func() {
				It("Should Deny", func() {
					na.Spec = nbv1.NooBaaAccountSpec{
						AllowBucketCreate: true,
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(3),
							GID:            Pointerify(2),
							NewBucketsPath: "/",
							NsfsOnly:       true,
						},
					}

					updatedNA.Spec = nbv1.NooBaaAccountSpec{}

					err = validations.ValidateRemoveNSFSConfig(*updatedNA, *na)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Removing the NsfsAccountConfig is unsupported"))
				})
			})
			Context("Update NSFSAccountConfig In NooBaaAccountSpec", func() {
				It("Should Allow", func() {
					na.Spec = nbv1.NooBaaAccountSpec{
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(3),
							GID:            Pointerify(2),
							NewBucketsPath: "/",
							NsfsOnly:       true,
						},
					}

					updatedNA.Spec = nbv1.NooBaaAccountSpec{
						NsfsAccountConfig: &nbv1.AccountNsfsConfig{
							UID:            Pointerify(30),
							GID:            Pointerify(20),
							NewBucketsPath: "/new/",
							NsfsOnly:       false,
						},
					}

					err = validations.ValidateRemoveNSFSConfig(*na, *updatedNA)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})

var _ = Describe("NamespaceStore deep-archive admission unit tests", func() {

	var (
		ns  *nbv1.NamespaceStore
		err error
	)

	newDeepArchiveNS := func(name, endpoint, bucket string) *nbv1.NamespaceStore {
		o := util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_namespacestore_cr_yaml).(*nbv1.NamespaceStore)
		o.Name = name
		o.Namespace = "test"
		o.Spec = nbv1.NamespaceStoreSpec{
			Type: nbv1.NSStoreTypeDeepArchive,
			DeepArchive: &nbv1.DeepArchiveSpec{
				Endpoint:     endpoint,
				TargetBucket: bucket,
				Secret:       corev1.SecretReference{Name: "secret", Namespace: "test"},
			},
		}
		return o
	}

	BeforeEach(func() {
		ns = newDeepArchiveNS("da-store", "https://archive.example.com:9000", "archive-bucket")
	})

	Describe("Validate create operations", func() {

		Context("Spec consistency", func() {
			It("Should Deny when DeepArchive spec is missing", func() {
				ns.Spec = nbv1.NamespaceStoreSpec{Type: nbv1.NSStoreTypeDeepArchive}
				err = validations.ValidateNSInValidSpec(*ns)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("DeepArchive spec must be provided for deep-archive type Namespacestore"))
			})
			It("Should Allow when spec is complete", func() {
				err = validations.ValidateNSInValidSpec(*ns)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Empty secret name", func() {
			It("Should Deny", func() {
				ns.Spec.DeepArchive.Secret.Name = ""
				err = validations.ValidateNSEmptySecretName(*ns)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("Failed creating the namespacestore, please provide secret name"))
			})
			It("Should Allow when secret name is set", func() {
				err = validations.ValidateNSEmptySecretName(*ns)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Empty target bucket", func() {
			It("Should Deny", func() {
				ns.Spec.DeepArchive.TargetBucket = ""
				err = validations.ValidateNSEmptyTargetBucket(*ns)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("Failed creating the namespacestore, please provide target bucket"))
			})
			It("Should Allow when target bucket is set", func() {
				err = validations.ValidateNSEmptyTargetBucket(*ns)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("Endpoint validation", func() {
			It("Should Deny invalid endpoint", func() {
				ns.Spec.DeepArchive.Endpoint = "hostname:port"
				err = validations.ValidateNsStoreDeepArchive(ns)
				Ω(err).Should(HaveOccurred())
			})
			It("Should Allow valid https endpoint", func() {
				err = validations.ValidateNsStoreDeepArchive(ns)
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("Should Deny empty endpoint", func() {
				ns.Spec.DeepArchive.Endpoint = ""
				err = validations.ValidateNsStoreDeepArchive(ns)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("Failed creating the namespacestore, please provide endpoint"))
			})
		})

		Context("Duplicate endpoint+targetBucket detection", func() {
			It("Should Deny when an existing store uses the same endpoint+bucket", func() {
				existing := *newDeepArchiveNS("existing-store", "https://archive.example.com:9000", "archive-bucket")
				err = validations.ValidateDuplicateDeepArchiveNS(ns, []nbv1.NamespaceStore{existing})
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("existing-store"))
				Expect(err.Error()).To(ContainSubstring("archive-bucket"))
			})
			It("Should Allow when bucket differs", func() {
				existing := *newDeepArchiveNS("existing-store", "https://archive.example.com:9000", "other-bucket")
				err = validations.ValidateDuplicateDeepArchiveNS(ns, []nbv1.NamespaceStore{existing})
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("Should Allow when endpoint differs", func() {
				existing := *newDeepArchiveNS("existing-store", "https://other.example.com:9000", "archive-bucket")
				err = validations.ValidateDuplicateDeepArchiveNS(ns, []nbv1.NamespaceStore{existing})
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("Should Allow a store to match itself (idempotent / update path)", func() {
				existing := *newDeepArchiveNS(ns.Name, "https://archive.example.com:9000", "archive-bucket")
				err = validations.ValidateDuplicateDeepArchiveNS(ns, []nbv1.NamespaceStore{existing})
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("Should ignore non-deep-archive stores even if endpoint+bucket match", func() {
				existing := nbv1.NamespaceStore{}
				existing.Name = "s3-store"
				existing.Namespace = "test"
				existing.Spec = nbv1.NamespaceStoreSpec{
					Type: nbv1.NSStoreTypeS3Compatible,
					S3Compatible: &nbv1.S3CompatibleSpec{
						Endpoint:     "https://archive.example.com:9000",
						TargetBucket: "archive-bucket",
						Secret:       corev1.SecretReference{Name: "secret", Namespace: "test"},
					},
				}
				err = validations.ValidateDuplicateDeepArchiveNS(ns, []nbv1.NamespaceStore{existing})
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("Validate update operations", func() {
		Context("Target bucket immutability", func() {
			It("Should Deny when target bucket changes", func() {
				oldNS := *newDeepArchiveNS(ns.Name, ns.Spec.DeepArchive.Endpoint, "original-bucket")
				ns.Spec.DeepArchive.TargetBucket = "changed-bucket"
				err = validations.ValidateTargetNSBucketChange(*ns, oldNS)
				Ω(err).Should(HaveOccurred())
				Expect(err.Error()).To(Equal("Changing a NamespaceStore target bucket is unsupported"))
			})
			It("Should Allow when target bucket is unchanged", func() {
				oldNS := *newDeepArchiveNS(ns.Name, ns.Spec.DeepArchive.Endpoint, ns.Spec.DeepArchive.TargetBucket)
				err = validations.ValidateTargetNSBucketChange(*ns, oldNS)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("Noobaa admission unit tests", func() {

	var (
		nb  *nbv1.NooBaa
		err error
	)

	BeforeEach(func() {
		nb = util.KubeObject(bundle.File_deploy_crds_noobaa_io_v1alpha1_noobaa_cr_yaml).(*nbv1.NooBaa)
		nb.Name = "noobaa"
		nb.Namespace = "test"

	})

	Describe("Validate delete operations", func() {
		Describe("General noobaa validations", func() {
			Context("cleanup policy not set", func() {
				It("Should Deny", func() {
					err = validations.ValidateNoobaaDeletion(*nb)
					Ω(err).Should(HaveOccurred())
					Expect(err.Error()).To(Equal("Noobaa cleanup policy is not set, blocking Noobaa deletion"))
				})
				It("Should Allow", func() {
					nb.Spec = nbv1.NooBaaSpec{
						CleanupPolicy: nbv1.CleanupPolicySpec{
							Confirmation:        "confirmed",
							AllowNoobaaDeletion: true,
						},
					}
					err = validations.ValidateNoobaaDeletion(*nb)
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})
})
