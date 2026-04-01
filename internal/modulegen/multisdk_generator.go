package modulegen

import (
	"fmt"
)

var _ ModuleGenerator = MultiSDKGenerator{}

type MultiSDKGenerator struct{}

func (g MultiSDKGenerator) ModuleType() ModuleType {
	return ModuleTypeMultiSDK
}

func (g MultiSDKGenerator) CheckInput(input *Input) []string {
	if input.MultiSDK == nil {
		return []string{"multi_sdk"}
	}
	return CheckRequiredInputStr([]RequiredStr{
		{input.ProjectID, "project_id"},
		{input.MultiSDK.ModuleSource, "module_source"},
		{input.MultiSDK.AWSRegion, "aws_region"},
		{input.MultiSDK.AWSBucketName, "aws_bucket_name"},
		{input.MultiSDK.AzureSubscriptionID, "azure_subscription_id"},
		{input.MultiSDK.AzureTenantID, "azure_tenant_id"},
		{input.MultiSDK.AzureResourceGroupName, "azure_resource_group_name"},
		{input.MultiSDK.AzureADGroupID, "azure_ad_group_id"},
		{input.MultiSDK.GCPProjectID, "gcp_project_id"},
		{input.MultiSDK.GCPRegion, "gcp_region"},
		{input.MultiSDK.GCPBucketName, "gcp_bucket_name"},
		{input.MultiSDK.AtlasRoleIDAWS, "atlas_role_id_aws"},
		{input.MultiSDK.AtlasRoleIDAzure, "atlas_role_id_azure"},
		{input.MultiSDK.AtlasRoleIDGCP, "atlas_role_id_gcp"},
	})
}

func (g MultiSDKGenerator) GetResourcesToFetch(_ *Input, resources *ResourcesToFetch) {
	resources.Atlas[AtlasResourceTypeCloudProviderAccessRoles] = true
	resources.AWS[AWSResourceTypeS3Bucket] = true
	resources.Azure[AzureResourceTypeADGroup] = true
	resources.Azure[AzureResourceTypeResourceGroup] = true
	resources.GCP[GCPResourceTypeStorageBucket] = true
}

func (g MultiSDKGenerator) Generate(input *Input, store *ResourceStore) (*GenerateModuleResult, error) {
	awsRole := store.Atlas.CPARoleAWS
	azureRole := store.Atlas.CPARoleAzure
	gcpRole := store.Atlas.CPARoleGCP

	result := GenerateModuleResult{
		ModuleType:       g.ModuleType(),
		TerraformVersion: Version{Operator: ">=", Major: 1, Minor: 9},
		Providers: []ProviderRequirement{
			{ProviderType: ProviderTypeAtlas, Version: Version{Operator: "~>", Major: 2, Minor: 7}},
			{ProviderType: ProviderTypeAWS, Version: Version{Operator: ">=", Major: 6, Minor: 0}},
			{ProviderType: ProviderTypeAzureRM, Version: Version{Operator: ">=", Major: 3, Minor: 0}},
			{ProviderType: ProviderTypeAzureAD, Version: Version{Operator: ">=", Major: 2, Minor: 53}},
			{ProviderType: ProviderTypeGoogle, Version: Version{Operator: ">=", Major: 6, Minor: 0}},
		},
		ImportBlocks: []*ImportBlock{
			// Atlas cloud provider access setup
			{
				ID: fmt.Sprintf("%s-AWS-%s", input.ProjectID, awsRole.GetRoleId()),
				To: "module.multi_cloud_test.mongodbatlas_cloud_provider_access_setup.aws",
			},
			{
				ID: fmt.Sprintf("%s-AZURE-%s", input.ProjectID, azureRole.GetId()),
				To: "module.multi_cloud_test.mongodbatlas_cloud_provider_access_setup.azure",
			},
			{
				ID: fmt.Sprintf("%s-GCP-%s", input.ProjectID, gcpRole.GetRoleId()),
				To: "module.multi_cloud_test.mongodbatlas_cloud_provider_access_setup.gcp",
			},
			// AWS
			{
				ID: input.MultiSDK.AWSBucketName,
				To: "module.multi_cloud_test.aws_s3_bucket.atlas_test",
			},
			// Azure
			{
				ID: fmt.Sprintf(
					"/subscriptions/%s/resourceGroups/%s", input.MultiSDK.AzureSubscriptionID, input.MultiSDK.AzureResourceGroupName,
				),
				To: "module.multi_cloud_test.azurerm_resource_group.main",
			},
			{
				ID: fmt.Sprintf("/groups/%s", input.MultiSDK.AzureADGroupID),
				To: "module.multi_cloud_test.azuread_group.atlas_test",
			},
			// GCP
			{
				ID: input.MultiSDK.GCPBucketName,
				To: "module.multi_cloud_test.google_storage_bucket.atlas_test",
			},
		},
		ModuleBlocks: []*ModuleBlock{{
			Name:   "multi_cloud_test",
			Source: input.MultiSDK.ModuleSource,
		}},
	}

	var attributes []Attribute
	var variables []*Variable

	addVar := func(name, description, value string) {
		v := NewStringVar(name, description, value)
		variables = append(variables, v)
		attributes = append(attributes, VarAttr(v.Name, v))
	}

	addVar("project_id", "MongoDB Atlas project ID", input.ProjectID)

	// AWS
	addVar("aws_bucket_name", "Name of the S3 bucket", input.MultiSDK.AWSBucketName)

	// Azure
	addVar("azure_tenant_id", "Azure tenant ID", input.MultiSDK.AzureTenantID)
	addVar("atlas_azure_app_id", "MongoDB Atlas Azure application ID", azureRole.GetAtlasAzureAppId())
	addVar(
		"azure_service_principal_id",
		"Object ID of the Atlas service principal in your Azure tenant",
		azureRole.GetServicePrincipalId(),
	)
	addVar("azure_resource_group_name", "Name for the Azure resource group", *store.Azure.ResourceGroup.Name)
	addVar("azure_location", "Azure region", *store.Azure.ResourceGroup.Location)
	addVar("azure_ad_group_name", "Display name for the Azure AD security group", *store.Azure.ADGroup.GetDisplayName())

	// GCP
	addVar("gcp_region", "GCP region", input.MultiSDK.GCPRegion)
	addVar("gcp_bucket_name", "Name of the GCS bucket", input.MultiSDK.GCPBucketName)

	result.ModuleBlocks[0].Attributes = attributes
	result.Variables = variables
	return &result, nil
}
