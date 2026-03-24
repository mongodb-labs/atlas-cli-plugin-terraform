package modulegen

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"go.mongodb.org/atlas-sdk/v20250312014/admin"
)

type Input struct {
	Modules      []ModuleType `hcl:"modules"`
	OrgID        string       `hcl:"org_id,optional"`
	ProjectID    string       `hcl:"project_id,optional"`
	ClusterNames []string     `hcl:"cluster_names,optional"`
}

type ModuleType string

const (
	ModuleTypeProject ModuleType = "project"
	ModuleTypeCluster ModuleType = "cluster"
)

type ResourceType string

const (
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeProject      ResourceType = "project"
	ResourceTypeClusters     ResourceType = "clusters"
)

type ResourceStore struct {
	Organization *admin.AtlasOrganization
	Project      *admin.Group
	Clusters     []*admin.ClusterDescription20240805
}

type ModuleGenerator interface {
	ModuleType() ModuleType
	// CheckInput returns a list of missing or invalid input fields for the module
	CheckInput(input *Input) []string
	// GetResourcesToFetch returns a map (set) of the resource types that should be fetched for the module given the input.
	// Note: If necessary, we can use the bool to indicate whether the resource is required or optional
	// (fetch wouldn't fail for optionals)
	GetResourcesToFetch(input *Input, resourcesToFetch map[ResourceType]bool)
	Generate(input *Input, store *ResourceStore) (*GenerateModuleResult, error)
}

type GenerateModuleResult struct {
	ModuleType   ModuleType
	Providers    []ProviderInfo
	ImportBlocks []*ImportBlock
	ModuleBlocks []*ModuleBlock
	Variables    []*Variable
	// TODO add outputs
	// OutputBlocks []*OutputBlock // nolint:gocritic
	TerraformVersion Version
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

type ModuleBlock struct {
	Name       string
	Source     string
	Version    string
	Attributes []Attribute
}

type ProviderType string

const (
	ProviderTypeAtlas ProviderType = "mongodbatlas"
)

func ProviderSource(providerType ProviderType) string {
	switch providerType { //nolint:gocritic
	case ProviderTypeAtlas:
		return "mongodb/mongodbatlas"
	}
	panic("unsupported provider type")
}

// TODO@check: Provider type should be enough to generate the provider version + provider block. So we should only need
// the min version per module to then use the highest one across modules.
type ProviderInfo struct {
	ProviderType ProviderType
	Version      Version
}

type ImportBlock struct {
	ID string
	To string
}

type Attribute struct {
	Name    string
	Comment *string // Comments to be included right above this input in the generated config

	// The attribute value, only one of the following is set
	Literal      *cty.Value
	Variable     *Variable
	NestedInputs []Attribute
}

type Variable struct {
	Value       cty.Value
	Type        cty.Type
	Name        string
	Description string
}
