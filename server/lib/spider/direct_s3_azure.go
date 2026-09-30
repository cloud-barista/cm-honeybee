package spider

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"
)

// azureBlobServiceURL is the service URL cb-spider's newAzureBlobClient()
// (S3Manager_azure.go) builds from the storage account name.
func azureBlobServiceURL(accountName string) string {
	return fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
}

// newAzureBlobClient is cb-spider's newAzureBlobClient() (S3Manager_azure.go).
func newAzureBlobClient(connInfo *s3ConnInfo) (*azblob.Client, error) {
	sharedKeyCred, err := azblob.NewSharedKeyCredential(connInfo.AccessKey, connInfo.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure SharedKeyCredential: %w", err)
	}

	client, err := azblob.NewClientWithSharedKeyCredential(azureBlobServiceURL(connInfo.AccessKey), sharedKeyCred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Blob client: %w", err)
	}

	return client, nil
}

// azureContainerEntries converts one page of containers the way the AZURE
// branch of cb-spider's ListAllS3BucketInfo() (S3Manager.go) does: the
// container's LastModified stands in for CreationDate.
func azureContainerEntries(items []*service.ContainerItem) []s3BucketEntry {
	var entries []s3BucketEntry
	for _, item := range items {
		if item == nil || item.Name == nil {
			continue
		}
		lastModified := time.Time{}
		if item.Properties != nil && item.Properties.LastModified != nil {
			lastModified = *item.Properties.LastModified
		}
		entries = append(entries, s3BucketEntry{Name: *item.Name, CreationDate: lastModified})
	}
	return entries
}

// listAzureContainers is the AZURE branch of cb-spider's ListAllS3BucketInfo()
// (S3Manager.go).
func listAzureContainers(connInfo *s3ConnInfo) ([]s3BucketEntry, error) {
	client, err := newAzureBlobClient(connInfo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var entries []s3BucketEntry
	pager := client.NewListContainersPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure containers: %w", err)
		}
		entries = append(entries, azureContainerEntries(page.ContainerItems)...)
	}
	return entries, nil
}
