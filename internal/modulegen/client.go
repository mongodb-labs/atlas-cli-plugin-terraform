package modulegen

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"go.mongodb.org/atlas-sdk/v20250312014/admin"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	CloudServiceURL    = "https://cloud.mongodb.com"
	CloudGovServiceURL = "https://cloud.mongodbgov.com"
	CloudDevServiceURL = "https://cloud-dev.mongodb.com"
)

var _ Client = &DefaultClient{}

// Client mock-able interface for tests
type Client interface {
	// FetchResources fetches all resources needed for generating the requested modules.
	FetchResources(ctx context.Context, input *Input, resourcesToFetch *ResourcesToFetch) (*ResourceStore, error)
}

type DefaultClient struct {
	HTTPClient   *http.Client
	AtlasBaseURL string
	UserAgent    string
}

type apiClients struct {
	atlas *admin.APIClient
	aws   awsClients
	azure azureClients
	gcp   gcpClients
}

type awsClients struct {
	S3 *s3.Client
	// Other AWS clients...
}

type azureClients struct {
	Graph          *msgraphsdk.GraphServiceClient
	ResourceGroups *armresources.ResourceGroupsClient
	// Other Azure clients...
}

type gcpClients struct {
	Storage *storage.Client
	// Other GCP clients...
}

func (c *DefaultClient) initAPIClients(
	ctx context.Context,
	input *Input,
	resourcesToFetch *ResourcesToFetch,
) (*apiClients, error) {
	var err error
	clients := apiClients{}

	if len(resourcesToFetch.Atlas) > 0 {
		clients.atlas, err = admin.NewClient(
			// Uncomment to see Atlas SDK debug logs when tool is run in debug mode.
			// admin.UseDebug(log.IsDebugLevel()) //nolint:gocritic
			admin.UseBaseURL(c.AtlasBaseURL),
			admin.UseHTTPClient(c.HTTPClient),
			admin.UseUserAgent(c.UserAgent),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create atlas client: %w", err)
		}
	}

	if len(resourcesToFetch.AWS) > 0 {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		for resourceType := range resourcesToFetch.AWS {
			switch resourceType { //nolint:gocritic
			case AWSResourceTypeS3Bucket:
				if clients.aws.S3 == nil {
					clients.aws.S3 = s3.NewFromConfig(cfg)
				}
			}
		}
	}

	if len(resourcesToFetch.Azure) > 0 {
		creds, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credential: %w", err)
		}
		for resourceType := range resourcesToFetch.Azure {
			switch resourceType { //nolint:gocritic
			case AzureResourceTypeADGroup:
				if clients.azure.ResourceGroups == nil {
					clients.azure.ResourceGroups, err = armresources.NewResourceGroupsClient(
						input.MultiSDK.AzureSubscriptionID, creds, nil,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create Azure resource groups client: %w", err)
					}
				}
			case AzureResourceTypeResourceGroup:
				if clients.azure.Graph == nil {
					clients.azure.Graph, err = msgraphsdk.NewGraphServiceClientWithCredentials(creds, nil) // nil for default scopes
					if err != nil {
						return nil, fmt.Errorf("failed to create Azure Graph client: %w", err)
					}
				}
			}
		}
	}

	if len(resourcesToFetch.GCP) > 0 {
		creds, err := google.FindDefaultCredentials(ctx, storage.ScopeReadOnly)
		if err != nil {
			return nil, fmt.Errorf("failed to load GCP credentials: %w", err)
		}
		for resourceType := range resourcesToFetch.GCP {
			switch resourceType { //nolint:gocritic
			case GCPResourceTypeStorageBucket:
				if clients.gcp.Storage == nil {
					clients.gcp.Storage, err = storage.NewClient(ctx, option.WithCredentials(creds))
					if err != nil {
						return nil, fmt.Errorf("failed to create GCP storage client: %w", err)
					}
				}
			}
		}
	}

	return &clients, nil
}

func (c *DefaultClient) FetchResources(
	ctx context.Context,
	input *Input,
	resourcesToFetch *ResourcesToFetch,
) (*ResourceStore, error) {
	clients, err := c.initAPIClients(ctx, input, resourcesToFetch)
	if err != nil {
		return nil, err
	}

	// TODO: Parallelize
	// TODO: Parallelize
	// TODO: Parallelize
	resourceStore := ResourceStore{}

	for resourceType := range resourcesToFetch.Atlas {
		switch resourceType {
		case AtlasResourceTypeOrganization:
			logger.Infof("Reading organization `%s` from MongoDB Atlas...\n", input.OrgID)
			org, _, err := clients.atlas.OrganizationsApi.GetOrg(ctx, input.OrgID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading organization `%s` from MongoDB Atlas: %w", input.OrgID, err)
			}
			resourceStore.Atlas.Organization = org
		case AtlasResourceTypeProject:
			logger.Infof("Reading project `%s` from MongoDB Atlas...\n", input.ProjectID)
			project, _, err := clients.atlas.ProjectsApi.GetGroup(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.Atlas.Project = project
		/* TODO@non-spike: See project_generator.go
		case AtlasResourceTypeProjectLimits:
			logger.Infof("Reading project limits for `%s` from MongoDB Atlas...\n", input.ProjectID)
			projectLimits, _, err := clients.atlas.ProjectsApi.ListGroupLimits(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project limits for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.ProjectLimits = projectLimits
		*/
		case AtlasResourceTypeProjectSettings:
			logger.Infof("Reading project settings for `%s` from MongoDB Atlas...\n", input.ProjectID)
			ps, _, err := clients.atlas.ProjectsApi.GetGroupSettings(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project settings for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.Atlas.ProjectSettings = ps
		case AtlasResourceTypeProjectIPAccessList:
			logger.Infof("Reading project IP access list for `%s` from MongoDB Atlas...\n", input.ProjectID)
			list, _, err := clients.atlas.ProjectIPAccessListApi.ListAccessListEntries(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project IP access list for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.Atlas.ProjectIPAccessList = list
		case AtlasResourceTypeProjectMaintenanceWindow:
			logger.Infof("Reading project maintenance window for `%s` from MongoDB Atlas...\n", input.ProjectID)
			mw, _, err := clients.atlas.MaintenanceWindowsApi.GetMaintenanceWindow(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf(
					"error reading project maintenance window for `%s` from MongoDB Atlas: %w", input.ProjectID, err,
				)
			}
			resourceStore.Atlas.ProjectMaintenanceWindow = mw
		case AtlasResourceTypeClusters:
			logger.Infof("Reading clusters [`%s`] from MongoDB Atlas...\n", strings.Join(input.ClusterNames, "`, `"))
			clusters := make([]*admin.ClusterDescription20240805, len(input.ClusterNames))
			for i, clusterName := range input.ClusterNames {
				cluster, _, err := clients.atlas.ClustersApi.GetCluster(ctx, input.ProjectID, clusterName).Execute()
				if err != nil {
					return nil, fmt.Errorf("error reading cluster `%s` from MongoDB Atlas: %w", clusterName, err)
				}
				clusters[i] = cluster
			}
			resourceStore.Atlas.Clusters = clusters
		case AtlasResourceTypeCloudProviderAccessRoles:
			logger.Infof("Reading cloud provider access roles for project `%s` from MongoDB Atlas...\n", input.ProjectID)
			roles, _, err := clients.atlas.CloudProviderAccessApi.ListCloudProviderAccess(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf(
					"error reading cloud provider access roles for project `%s` from MongoDB Atlas: %w",
					input.ProjectID, err,
				)
			}
			if input.MultiSDK.AtlasRoleIDAWS != "" {
				awsRoles := roles.GetAwsIamRoles()
				for i := range awsRoles {
					role := &awsRoles[i]
					if role.GetRoleId() == input.MultiSDK.AtlasRoleIDAWS {
						resourceStore.Atlas.CPARoleAWS = role
						break
					}
				}
				if resourceStore.Atlas.CPARoleAWS == nil {
					return nil, fmt.Errorf(
						"AWS CPA role `%s` not found in project `%s`", input.MultiSDK.AtlasRoleIDAWS, input.ProjectID,
					)
				}
			}
			if input.MultiSDK.AtlasRoleIDAzure != "" {
				azRoles := roles.GetAzureServicePrincipals()
				for i := range azRoles {
					role := &azRoles[i]
					if role.GetId() == input.MultiSDK.AtlasRoleIDAzure {
						resourceStore.Atlas.CPARoleAzure = role
						break
					}
				}
				if resourceStore.Atlas.CPARoleAzure == nil {
					//nolint:staticcheck // capitalized error due to "Azure"
					return nil, fmt.Errorf(
						"Azure CPA role `%s` not found in project `%s`", input.MultiSDK.AtlasRoleIDAzure, input.ProjectID,
					)
				}
			}
			if input.MultiSDK.AtlasRoleIDGCP != "" {
				gcpRoles := roles.GetGcpServiceAccounts()
				for i := range gcpRoles {
					role := &gcpRoles[i]
					if role.GetRoleId() == input.MultiSDK.AtlasRoleIDGCP {
						resourceStore.Atlas.CPARoleGCP = role
						break
					}
				}
				if resourceStore.Atlas.CPARoleGCP == nil {
					return nil, fmt.Errorf(
						"GCP CPA role `%s` not found in project `%s`", input.MultiSDK.AtlasRoleIDGCP, input.ProjectID,
					)
				}
			}
		}
	}

	for resourceType := range resourcesToFetch.AWS {
		switch resourceType { //nolint:gocritic
		case AWSResourceTypeS3Bucket:
			logger.Infof("Reading S3 bucket `%s` from AWS...\n", input.MultiSDK.AWSBucketName)
			bucket, err := clients.aws.S3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &input.MultiSDK.AWSBucketName})
			if err != nil {
				return nil, fmt.Errorf("error reading S3 bucket `%s` from AWS: %w", input.MultiSDK.AWSBucketName, err)
			}
			resourceStore.AWS.S3Bucket = bucket
		}
	}

	for resourceType := range resourcesToFetch.Azure {
		switch resourceType {
		case AzureResourceTypeADGroup:
			logger.Infof("Reading active directory group `%s` from Azure...\n", input.MultiSDK.AzureADGroupID)
			groupable, err := clients.azure.Graph.Groups().ByGroupId(input.MultiSDK.AzureADGroupID).Get(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading active directory group `%s` from Azure: %w", input.MultiSDK.AzureADGroupID, err,
				)
			}
			group, ok := groupable.(*graphmodels.Group)
			if !ok {
				return nil, fmt.Errorf("unexpected type for Azure AD group: %T", groupable)
			}
			resourceStore.Azure.ADGroup = group
		case AzureResourceTypeResourceGroup:
			logger.Infof("Reading resource group `%s` from Azure...\n", input.MultiSDK.AzureResourceGroupName)
			response, err := clients.azure.ResourceGroups.Get(ctx, input.MultiSDK.AzureResourceGroupName, nil)
			if err != nil {
				return nil, fmt.Errorf(
					"error reading resource group `%s` from Azure: %w", input.MultiSDK.AzureResourceGroupName, err,
				)
			}
			resourceStore.Azure.ResourceGroup = &response.ResourceGroup
		}
	}

	for resourceType := range resourcesToFetch.GCP {
		switch resourceType { //nolint:gocritic
		case GCPResourceTypeStorageBucket:
			logger.Infof("Reading GCS bucket `%s` from GCP...\n", input.MultiSDK.GCPBucketName)
			attrs, err := clients.gcp.Storage.Bucket(input.MultiSDK.GCPBucketName).Attrs(ctx)
			if err != nil {
				return nil, fmt.Errorf("error reading GCS bucket `%s` from GCP: %w", input.MultiSDK.GCPBucketName, err)
			}
			resourceStore.GCP.StorageBucket = attrs
		}
	}

	return &resourceStore, nil
}
