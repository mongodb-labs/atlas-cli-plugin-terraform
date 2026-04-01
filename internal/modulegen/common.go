package modulegen

import (
	"fmt"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/zclconf/go-cty/cty"
	"go.mongodb.org/atlas-sdk/v20250312014/admin"
)

type Input struct {
	MultiSDK     *MultiSDKInput `hcl:"multi_sdk,block"` // TODO@remove: testing CSP SDKs
	OrgID        string         `hcl:"org_id,optional"`
	ProjectID    string         `hcl:"project_id,optional"`
	Modules      []ModuleType   `hcl:"modules"`
	ClusterNames []string       `hcl:"cluster_names,optional"`
}

// MultiSDKInput Using a separate struct for the multi-sdk test module inputs to avoid fieldalignment from mixing inputs
type MultiSDKInput struct {
	ModuleSource           string `hcl:"module_source,optional"` // Path to the local module dir
	AtlasRoleIDAWS         string `hcl:"atlas_role_id_aws,optional"`
	AtlasRoleIDAzure       string `hcl:"atlas_role_id_azure,optional"`
	AtlasRoleIDGCP         string `hcl:"atlas_role_id_gcp,optional"`
	AWSRegion              string `hcl:"aws_region,optional"`
	AWSBucketName          string `hcl:"aws_bucket_name,optional"`
	AzureSubscriptionID    string `hcl:"azure_subscription_id,optional"`
	AzureTenantID          string `hcl:"azure_tenant_id,optional"`
	AzureResourceGroupName string `hcl:"azure_resource_group_name,optional"`
	AzureADGroupID         string `hcl:"azure_ad_group_id,optional"`
	GCPProjectID           string `hcl:"gcp_project_id,optional"`
	GCPRegion              string `hcl:"gcp_region,optional"`
	GCPBucketName          string `hcl:"gcp_bucket_name,optional"`
}

type RequiredStr struct {
	value string
	name  string
}

func CheckRequiredInputStr(required []RequiredStr) []string {
	var missing []string
	for _, r := range required {
		if r.value == "" {
			missing = append(missing, r.name)
		}
	}
	return missing
}

type ModuleType string

const (
	ModuleTypeProject ModuleType = "project"
	ModuleTypeCluster ModuleType = "cluster"

	ModuleTypeMultiSDK ModuleType = "multi-sdk" // TODO@remove: Module for testing CSP SDKs
)

func (m ModuleType) GetModuleGenerator() (ModuleGenerator, error) {
	switch m {
	case ModuleTypeProject:
		return ProjectGenerator{}, nil
	case ModuleTypeCluster:
		return ClustersGenerator{}, nil
	case ModuleTypeMultiSDK:
		return MultiSDKGenerator{}, nil
	}
	return nil, fmt.Errorf("invalid module type: %s", m)
}

type AtlasResourceType int

const (
	AtlasResourceTypeOrganization AtlasResourceType = iota
	AtlasResourceTypeProject
	// TODO@non-spike: See project_generator.go
	// AtlasResourceTypeProjectLimits
	AtlasResourceTypeProjectSettings
	AtlasResourceTypeProjectIPAccessList
	AtlasResourceTypeProjectMaintenanceWindow
	AtlasResourceTypeClusters
	AtlasResourceTypeCloudProviderAccessRoles
)

type AWSResourceType int

const (
	AWSResourceTypeS3Bucket AWSResourceType = iota
)

type AzureResourceType int

const (
	AzureResourceTypeADGroup AzureResourceType = iota
	AzureResourceTypeResourceGroup
)

type GCPResourceType int

const (
	GCPResourceTypeStorageBucket GCPResourceType = iota
)

type ResourcesToFetch struct {
	Atlas map[AtlasResourceType]bool
	AWS   map[AWSResourceType]bool
	Azure map[AzureResourceType]bool
	GCP   map[GCPResourceType]bool
}

func NewResourcesToFetch() *ResourcesToFetch {
	return &ResourcesToFetch{
		Atlas: make(map[AtlasResourceType]bool),
		AWS:   make(map[AWSResourceType]bool),
		Azure: make(map[AzureResourceType]bool),
		GCP:   make(map[GCPResourceType]bool),
	}
}

type ResourceStore struct {
	Azure AzureResources
	AWS   AWSResources
	GCP   GCPResources
	Atlas AtlasResources
}

type AtlasResources struct {
	Organization             *admin.AtlasOrganization                        `json:",omitempty"`
	Project                  *admin.Group                                    `json:",omitempty"`
	ProjectSettings          *admin.GroupSettings                            `json:",omitempty"`
	ProjectIPAccessList      *admin.PaginatedNetworkAccess                   `json:",omitempty"`
	ProjectMaintenanceWindow *admin.GroupMaintenanceWindow                   `json:",omitempty"`
	CPARoleAWS               *admin.CloudProviderAccessAWSIAMRole            `json:",omitempty"`
	CPARoleAzure             *admin.CloudProviderAccessAzureServicePrincipal `json:",omitempty"`
	CPARoleGCP               *admin.CloudProviderAccessGCPServiceAccount     `json:",omitempty"`
	ProjectLimits            []admin.DataFederationLimit                     `json:",omitempty"`
	Clusters                 []*admin.ClusterDescription20240805             `json:",omitempty"`
}

type AWSResources struct {
	S3Bucket *s3.HeadBucketOutput `json:",omitempty"`
}

type AzureResources struct {
	ADGroup       *graphmodels.Group          `json:",omitempty"`
	ResourceGroup *armresources.ResourceGroup `json:",omitempty"`
}

type GCPResources struct {
	StorageBucket *storage.BucketAttrs `json:",omitempty"`
}

type ModuleGenerator interface {
	ModuleType() ModuleType
	// CheckInput returns a list of missing or invalid input fields for the module
	CheckInput(input *Input) []string
	// GetResourcesToFetch populates the resource types that should be fetched for the module given the input.
	// Note: If necessary, we can use each map's bool to indicate whether the resource is required or optional
	// (fetch wouldn't fail for optionals)
	GetResourcesToFetch(input *Input, resources *ResourcesToFetch)
	Generate(input *Input, store *ResourceStore) (*GenerateModuleResult, error)
}

type GenerateModuleResult struct {
	ModuleType   ModuleType
	Providers    []ProviderRequirement
	ImportBlocks []*ImportBlock
	ModuleBlocks []*ModuleBlock
	Variables    []*Variable
	// TODO add outputs
	// OutputBlocks []*OutputBlock // nolint:gocritic
	TerraformVersion Version
}

type GenerateVersionsResult struct {
	Blocks    []*ProviderInfo
	Variables []*Variable
	TFVersion Version
}

type ProviderType string

const (
	ProviderTypeAtlas   ProviderType = "mongodbatlas"
	ProviderTypeAWS     ProviderType = "aws"
	ProviderTypeAzureRM ProviderType = "azurerm"
	ProviderTypeAzureAD ProviderType = "azuread"
	ProviderTypeGoogle  ProviderType = "google"
)

type ProviderRequirement struct {
	ProviderType ProviderType
	Version      Version
}

type ProviderInfo struct {
	Name       string
	Source     string
	Attributes []Attribute
	Version    Version
}

type Version struct {
	Operator string // e.g. ">=", "~>", "=", etc.
	Major    int
	Minor    int
}

func (v Version) String() string {
	if v.Operator == "" {
		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	}
	return fmt.Sprintf("%s %d.%d", v.Operator, v.Major, v.Minor)
}

func (v Version) GreaterThan(o Version) bool {
	return v.Major > o.Major || (v.Major == o.Major && v.Minor > o.Minor)
}

type ImportBlock struct {
	ID      string
	To      string
	ForEach []string
}

type ModuleBlock struct {
	Version    *Version
	Name       string
	Source     string
	Attributes []Attribute
}

type Variable struct {
	Value        cty.Value
	Type         cty.Type
	DefaultValue *cty.Value
	Name         string
	Description  string
}

type Attribute struct {
	Comment        *string
	Name           string
	Value          AttributeValue
	IsDefaultValue bool
}

type AttributeValue struct {
	// Only one of the following is set
	Literal    *cty.Value
	Variable   *Variable
	Object     []Attribute
	ObjectList [][]Attribute
	Block      []Attribute
}

type AttrOption func(*Attribute)

func IsDefault(isDefault bool) AttrOption {
	return func(a *Attribute) { a.IsDefaultValue = isDefault }
}

func Comment(comment string) AttrOption {
	return func(a *Attribute) { a.Comment = &comment }
}

func applyAttrOptions(a *Attribute, opts []AttrOption) {
	for _, opt := range opts {
		opt(a)
	}
}

func LiteralAttr(name string, value cty.Value, opts ...AttrOption) Attribute {
	a := Attribute{Name: name, Value: AttributeValue{Literal: &value}}
	applyAttrOptions(&a, opts)
	return a
}

func StringAttr(name, value string, opts ...AttrOption) Attribute {
	return LiteralAttr(name, cty.StringVal(value), opts...)
}

func BoolAttr(name string, value bool, opts ...AttrOption) Attribute {
	return LiteralAttr(name, cty.BoolVal(value), opts...)
}

func IntAttr(name string, value int, opts ...AttrOption) Attribute {
	return LiteralAttr(name, cty.NumberIntVal(int64(value)), opts...)
}

func ObjectAttr(name string, attrs []Attribute, opts ...AttrOption) Attribute {
	a := Attribute{Name: name, Value: AttributeValue{Object: attrs}}
	applyAttrOptions(&a, opts)
	return a
}

func BlockAttr(name string, attrs []Attribute, opts ...AttrOption) Attribute {
	if attrs == nil {
		attrs = []Attribute{}
	}
	a := Attribute{Name: name, Value: AttributeValue{Block: attrs}}
	applyAttrOptions(&a, opts)
	return a
}

func VarAttr(name string, v *Variable, opts ...AttrOption) Attribute {
	a := Attribute{Name: name, Value: AttributeValue{Variable: v}}
	applyAttrOptions(&a, opts)
	return a
}

func NewStringVar(name, description, value string) *Variable {
	return &Variable{
		Name:        name,
		Description: description,
		Type:        cty.String,
		Value:       cty.StringVal(value),
	}
}
