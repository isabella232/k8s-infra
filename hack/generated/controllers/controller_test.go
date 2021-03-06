/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

package controllers_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	storage "github.com/Azure/k8s-infra/hack/generated/apis/microsoft.storage/v20190401"
	"github.com/Azure/k8s-infra/hack/generated/pkg/armclient"
	"github.com/Azure/k8s-infra/hack/generated/pkg/testcommon"
)

func Test_ResourceGroup_CRUD(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()
	testContext, err := testContext.ForTest(t)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a resource group
	rg := testContext.NewTestResourceGroup()
	err = testContext.KubeClient.Create(ctx, rg)
	g.Expect(err).ToNot(HaveOccurred())

	// It should be created in Kubernetes
	g.Eventually(rg).Should(testContext.Match.BeProvisioned(ctx))

	g.Expect(rg.Status.Location).To(Equal(testContext.AzureRegion))
	g.Expect(rg.Status.Properties.ProvisioningState).To(Equal(string(armclient.SucceededProvisioningState)))
	g.Expect(rg.Status.ID).ToNot(BeNil())
	armId := rg.Status.ID

	// Delete the resource group
	err = testContext.KubeClient.Delete(ctx, rg)
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(rg).Should(testContext.Match.BeDeleted(ctx))

	// Ensure that the resource group was really deleted in Azure
	// TODO: Do we want to just use an SDK here? This process is quite icky as is...
	exists, err := testContext.AzureClient.HeadResource(
		ctx,
		armId,
		"2020-06-01")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exists).To(BeFalse())
}

func Test_StorageAccount_CRUD(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()
	testContext, err := testContext.ForTest(t)
	g.Expect(err).ToNot(HaveOccurred())

	rg, err := testContext.CreateNewTestResourceGroup(testcommon.WaitForCreation)
	g.Expect(err).ToNot(HaveOccurred())

	// Custom namer because storage accounts have strict names
	namer := testContext.Namer.WithSeparator("")

	// Create a storage account
	accessTier := storage.StorageAccountPropertiesCreateParametersAccessTierHot
	acct := &storage.StorageAccount{
		ObjectMeta: testContext.MakeObjectMetaWithName(namer.GenerateName("stor")),
		Spec: storage.StorageAccounts_Spec{
			Location:   testContext.AzureRegion,
			ApiVersion: "2019-04-01", // TODO [apiversion]: This should be removed from the storage type eventually
			Owner:      testcommon.AsOwner(rg.ObjectMeta),
			Kind:       storage.StorageAccountsSpecKindBlobStorage,
			Sku: storage.Sku{
				Name: storage.SkuNameStandardLRS,
			},
			// TODO: They mark this property as optional but actually it is required
			Properties: &storage.StorageAccountPropertiesCreateParameters{
				AccessTier: &accessTier,
			},
		},
	}
	err = testContext.KubeClient.Create(ctx, acct)
	g.Expect(err).ToNot(HaveOccurred())

	// It should be created in Kubernetes
	g.Eventually(acct).Should(testContext.Match.BeProvisioned(ctx))

	g.Expect(acct.Status.Location).To(Equal(testContext.AzureRegion))
	expectedKind := storage.StorageAccountStatusKindBlobStorage
	g.Expect(acct.Status.Kind).To(Equal(&expectedKind))
	g.Expect(acct.Status.Id).ToNot(BeNil())
	armId := *acct.Status.Id

	// Run sub-tests on storage account
	t.Run("Blob Services CRUD", func(t *testing.T) {
		StorageAccount_BlobServices_CRUD(t, testContext, acct.ObjectMeta)
	})

	// Delete the storage account
	err = testContext.KubeClient.Delete(ctx, acct)
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(acct).Should(testContext.Match.BeDeleted(ctx))

	// Ensure that the resource group was really deleted in Azure
	exists, err := testContext.AzureClient.HeadResource(
		ctx,
		armId,
		"2019-04-01")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(exists).To(BeFalse())
}

func StorageAccount_BlobServices_CRUD(t *testing.T, testContext testcommon.KubePerTestContext, storageAccount metav1.ObjectMeta) {
	ctx := context.Background()

	g := NewGomegaWithT(t)

	blobService := &storage.StorageAccountsBlobService{
		ObjectMeta: testContext.MakeObjectMeta("blobservice"),
		Spec: storage.StorageAccountsBlobServices_Spec{
			ApiVersion: "2019-04-01", // TODO [apiversion]: to be removed eventually
			Owner:      testcommon.AsOwner(storageAccount),
		},
	}

	// Create
	err := testContext.KubeClient.Create(ctx, blobService)
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(blobService).Should(testContext.Match.BeProvisioned(ctx))

	t.Run("Container CRUD", func(t *testing.T) {
		StorageAccount_BlobServices_Container_CRUD(t, testContext, blobService.ObjectMeta)
	})

	// TODO: Delete doesn't seem to work?
	// — is this because it is not a real resource but properties on the storage account?
	// We probably need to ask the Storage team.
	/*
		err = testContext.KubeClient.Delete(ctx, blobService)
		g.Expect(err).ToNot(HaveOccurred())
		g.Eventually(blobService).Should(testContext.Match.BeDeleted(ctx))
	*/
}

func StorageAccount_BlobServices_Container_CRUD(t *testing.T, testContext testcommon.KubePerTestContext, blobService metav1.ObjectMeta) {
	ctx := context.Background()
	g := NewGomegaWithT(t)

	blobContainer := &storage.StorageAccountsBlobServicesContainer{
		ObjectMeta: testContext.MakeObjectMeta("container"),
		Spec: storage.StorageAccountsBlobServicesContainers_Spec{
			ApiVersion: "2019-04-01", // TODO [apiversion]: to be removed eventually
			Owner:      testcommon.AsOwner(blobService),
		},
	}

	// Create
	err := testContext.KubeClient.Create(ctx, blobContainer)
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(blobContainer).Should(testContext.Match.BeProvisioned(ctx))

	// Delete
	err = testContext.KubeClient.Delete(ctx, blobContainer)
	g.Expect(err).ToNot(HaveOccurred())
	g.Eventually(blobContainer).Should(testContext.Match.BeDeleted(ctx))
}
