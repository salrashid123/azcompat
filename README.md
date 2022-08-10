## Exchange Google and Firebase OIDC tokens for Azure STS

Tutorial and golang library that will federate a [google id token](https://github.com/salrashid123/google_id_token) token for Azure access tokens (**GCP Credentials --> Azure Resources**)

You can use this procedure to access azure resources directly from `Google Cloud Run`, `Cloud Functions`, `GCP VMs` or any other system where you can get a `google id_token`. 

This is the opposite of [Azure Credentials --> GCP resources](https://cloud.google.com/iam/docs/configuring-workload-identity-federation#azure) and basically uses [Azure Workload Identity Federation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation#how-it-works) to trust a google-issued `OIDC` token.  The final api call to azure will use an [access token request with a federated credential](https://docs.microsoft.com/en-us/azure/active-directory/develop/v2-oauth2-client-creds-grant-flow#third-case-access-token-request-with-a-federated-credential)


---

### References

* for **AWS Credentials --> GCP Resources** see [GCP Workload Identity Federation using AWS Credentials](https://github.com/salrashid123/gcpcompat-aws)
* for **GCP Credentials --> AWS Resources** see [Exchange Google and Firebase OIDC tokens for AWS STS](https://github.com/salrashid123/awscompat)
* for **Arbitrary OIDC Provider --> GCP Resources** see [GCP Workload Identity Federation using OIDC Credentials](https://github.com/salrashid123/gcpcompat-oidc)
* for **Arbitrary SAML Provider --> GCP Resources** see [GCP Workload Identity Federation using SAML Credentials](https://github.com/salrashid123/gcpcompat-saml)

---

### Azure Tenant, Subscription and Resources


In this tutorial, we will start with

An `Azure Subscription` with resource group consisting of a `VM` and `Storage Container`. I'm assuming you would have already set this up.

From there we will [Use the portal to create an Azure AD application and service principal that can access resources](https://docs.microsoft.com/en-us/azure/active-directory/develop/howto-create-service-principal-portal)


First note down your

- [tenant](https://docs.microsoft.com/en-us/azure/active-directory/develop/quickstart-create-new-tenant)

![images/tenant.png](images/tenant.png)

-  [subscription](https://docs.microsoft.com/en-us/dynamics-nav/how-to--sign-up-for-a-microsoft-azure-subscription), vm and container i'll use are

![images/subscription.png](images/subscription.png)

- [resource_group](https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/manage-resource-groups-portal)

![images/resource_group.png](images/resource_group.png)

- [vm name](https://docs.microsoft.com/en-us/azure/virtual-machines/linux/quick-create-portal)

![images/vm.png](images/vm.png)

- [storage account and container name](https://docs.microsoft.com/en-us/azure/storage/common/storage-account-overview)

![images/storage_account.png](images/storage_account.png)


[https://accounts.google.com/.well-known/openid-configuration](https://accounts.google.com/.well-known/openid-configuration)



```bash

export PROJECT_ID=`gcloud config get-value core/project`
gcloud iam service-accounts create elevate --display-name "Federated Service Account"
gcloud iam service-accounts keys create svc_account.json --iam-account elevate@$PROJECT_ID.iam.gserviceaccount.com
gcloud auth activate-service-account --key-file=`pwd`/svc_account.json

export FEDERATED_TOKEN=`gcloud auth print-identity-token --audiences api://AzureADTokenExchange`
echo $FEDERATED_TOKEN
```

```json
      {
        "aud": "api://AzureADTokenExchange",
        "azp": "elevate@fabled-ray-104117.iam.gserviceaccount.com",
        "email": "elevate@fabled-ray-104117.iam.gserviceaccount.com",
        "email_verified": true,
        "exp": 1660052382,
        "iat": 1660048782,
        "iss": "https://accounts.google.com",
        "sub": "117471943676050750091"
      }
```



az login   --service-principal \
    -u cffeaee2-5617-4784-8a4b-b647efd676d2  \
    --federated-token $FEDERATED_TOKEN --tenant srashid123hotmail.onmicrosoft.com --allow-no-subscriptions --output table

CloudName    HomeTenantId                          IsDefault    Name           State    TenantId
-----------  ------------------------------------  -----------  -------------  -------  ------------------------------------
AzureCloud   45243fbe-b73f-4f7d-8213-a104a99e228e  True         Pay-As-You-Go  Enabled  45243fbe-b73f-4f7d-8213-a104a99e228e


$ az account get-access-token


$ az storage blob list     --account-name mineralminutia    --container-name mineral-minutia  --output table --only-show-errors
Name    IsDirectory    Blob Type    Blob Tier    Length    Content Type         Last Modified              Snapshot
------  -------------  -----------  -----------  --------  -------------------  -------------------------  ----------
go.mod                 BlockBlob    Hot          21        application/xml-dtd  2022-08-08T13:45:44+00:00


$ az vm list --output table
Name    ResourceGroup    Location    Zones
------  ---------------  ----------  -------
vm1     RG1              eastus      1


```

```bash

export TENANT="srashid123hotmail.onmicrosoft.com"
export CLIENT_ID="cffeaee2-5617-4784-8a4b-b647efd676d2"
export FEDERATED_TOKEN=`gcloud auth print-identity-token --audiences api://AzureADTokenExchange`

curl -s https://login.microsoftonline.com/$TENANT/oauth2/v2.0/token \
--data-urlencode "client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer" \
--data-urlencode "grant_type=client_credentials" \
--data-urlencode "client_id=$CLIENT_ID" \
--data-urlencode "client_assertion=$FEDERATED_TOKEN" \
--data-urlencode "scope=https://management.core.windows.net/.default"


$ curl -s -H "Authorization: Bearer $AZURE_TOKEN" \
    "https://management.azure.com/subscriptions/450b3122-bc25-49b7-86be-7dc86269a2e4/providers/Microsoft.Compute/virtualMachines?api-version=2022-03-01" | jq '.'
{
  "value": [
    {
      "name": "vm1",
      "id": "/subscriptions/450b3122-bc25-49b7-86be-7dc86269a2e4/resourceGroups/RG1/providers/Microsoft.Compute/virtualMachines/vm1",
      "type": "Microsoft.Compute/virtualMachines",
      "location": "eastus",
      "properties": {
        "vmId": "48296592-9898-4778-bcc2-cd4cefad74f1",
        "hardwareProfile": {
          "vmSize": "Standard_B1ls"
        },
        "storageProfile": {
          "imageReference": {




```

---

```bash
export GODEBUG=http2debug=2

export GOOGLE_APPLICATION_CREDENTIALS=/home/srashid/gcp_misc/certs/elevate-fabled-ray-104117.json

go run main.go

```


