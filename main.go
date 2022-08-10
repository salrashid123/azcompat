package main

import (
	"context"
	"fmt"
	"os"

	//"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	azcompat "github.com/salrashid123/azcompat/google"
	//azcompat "github.com/salrashid123/azcompat/google"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

const (
	clientID = "cffeaee2-5617-4784-8a4b-b647efd676d2"
	audience = "api://AzureADTokenExchange"
	tenantID = "45243fbe-b73f-4f7d-8213-a104a99e228e"

	subscriptionID = "450b3122-bc25-49b7-86be-7dc86269a2e4"
	resourceGroup  = "rg1"
	vmName         = "vm1"

	containerName = "mineral-minutia"
	accountName   = "mineralminutia"
)

func main() {

	ctx := context.Background()

	//cred, err := azidentity.NewDefaultAzureCredential(nil)

	cred, err := azcompat.NewGCPAZCredentials(&azcompat.GCPAZCredentialsOptions{
		ClientID: clientID,
		Audience: audience,
		TenantID: tenantID,
	})
	if err != nil {
		fmt.Printf("Invalid credentials with error: " + err.Error())
		os.Exit(1)
	}

	// a, err := cred.GetToken(ctx, policy.TokenRequestOptions{
	// 	Scopes: []string{"https://management.core.windows.net/.default"},
	// })
	// if err != nil {
	// 	fmt.Printf("unable to get Token: %v", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("AZ Token %s\n", a.Token)

	client, err := armcompute.NewVirtualMachinesClient(subscriptionID, cred, nil)
	if err != nil {
		fmt.Printf("Invalid NewVirtualMachinesClient client error: " + err.Error())
		os.Exit(1)
	}

	v, err := client.Get(ctx, resourceGroup, vmName, nil)
	if err != nil {
		fmt.Printf("Error getting vm: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("VM: %s\n", *v.ID)

	// ****************************************************************

	sharedKey := ""

	aclient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		fmt.Printf("Error getting armstorage client: " + err.Error())
		os.Exit(1)
	}

	resp, err := aclient.ListKeys(context.Background(), resourceGroup, accountName, nil)
	if err != nil {
		fmt.Printf("Error getting keys: " + err.Error())
		os.Exit(1)
	}
	for _, k := range resp.Keys {
		//fmt.Printf("account key: %s,  value %s\n", *k.KeyName, *k.Value)
		//just get the first one
		sharedKey = *k.Value
		break
	}
	if sharedKey == "" {
		fmt.Printf("Error getting shared key")
		os.Exit(1)
	}

	sharedCred, err := azblob.NewSharedKeyCredential(accountName, sharedKey)
	if err != nil {
		fmt.Printf("Error getting keys: " + err.Error())
		os.Exit(1)
	}
	serviceClient, err := azblob.NewServiceClientWithSharedKey(fmt.Sprintf("https://%s.blob.core.windows.net/", accountName), sharedCred, &azblob.ClientOptions{})
	if err != nil {
		fmt.Printf("Invalid service client error: " + err.Error())
		os.Exit(1)
	}
	containerClient, err := serviceClient.NewContainerClient(containerName)
	if err != nil {
		fmt.Printf("Invalid NewContainerClient with error: " + err.Error())
		os.Exit(1)
	}

	pager := containerClient.ListBlobsFlat(nil)
	for pager.NextPage(ctx) {
		resp := pager.PageResponse()
		for _, v := range resp.Segment.BlobItems {
			fmt.Printf("File %s\n", *v.Name)
		}
	}

}
