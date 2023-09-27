// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

// tools to to parse ccache configuration file

package azureblobstoragepkg

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/microsoft/CBL-Mariner/toolkit/tools/internal/logger"
)

const (
	AnonymousAccess = 0
	AuthenticatedAccess = 1
)

type AzureBlobStorage struct {
	theClient *azblob.Client
}

// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob#section-readme
func (abs *AzureBlobStorage) Upload(
	ctx context.Context,
	fullFileName string,
	containerName string,
	blobName string) (err error) {

	uploadStartTime := time.Now()

	localFile, err := os.OpenFile(fullFileName, os.O_RDONLY, 0)
	if err != nil {
		logger.Log.Infof("  failed to open local file for upload. Error: %v", err)
		return err
	}
	defer localFile.Close()

	_, err = abs.theClient.UploadFile(ctx, containerName, blobName, localFile, nil)
	if err != nil {
		logger.Log.Infof("  failed to upload local file to blob. Error: %v", err)
		return err
	}

	uploadEndTime := time.Now()
	logger.Log.Infof("  upload time: %s", uploadEndTime.Sub(uploadStartTime))

	return nil
}

func (abs *AzureBlobStorage) Download(
	ctx context.Context,
	containerName string,
	blobName string,
	fullFileName string) (err error) {

	downloadStartTime := time.Now()

	localFile, err := os.Create(fullFileName)
	if err != nil {
		logger.Log.Infof("  failed to create local file for download. Error: %v", err)
		return err
	}
	defer localFile.Close()

	_, err = abs.theClient.DownloadFile(ctx, containerName, blobName, localFile, nil)
	if err != nil {
		logger.Log.Infof("  failed to download blob to local file. Error: %v", err)
		return err
	}

	downloadEndTime := time.Now()
	logger.Log.Infof("  download time: %v", downloadEndTime.Sub(downloadStartTime))

	return nil
}

func Create(tenantId string, userName string, password string, storageAccount string, authenticationType int) (abs *AzureBlobStorage, err error) {

	url := "https://" + storageAccount + ".blob.core.windows.net/"

	abs = &AzureBlobStorage{}

	if authenticationType == AnonymousAccess {

		abs.theClient, err = azblob.NewClientWithNoCredential(url, nil)
		if err != nil {
			logger.Log.Warnf("Unable to init azure blob storage read-only client. Error: %v", err)
			return nil, err
		}

		return abs, nil

	} else if authenticationType == AuthenticatedAccess {

		credential, err := azidentity.NewClientSecretCredential(tenantId, userName, password, nil)
		if err != nil {
			logger.Log.Warnf("Unable to init azure identity. Error: %v", err)
			return nil, err
		}

		abs.theClient, err = azblob.NewClient(url, credential, nil)
		if err != nil {
			logger.Log.Warnf("Unable to init azure blob storage read-write client. Error: %v", err)
			return nil, err
		}

		return abs, nil

	}

	logger.Log.Warnf("Unknown authentication type.")
	return nil, errors.New("Unknown authentication type.")
}
