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

func (m ModuleType) GetModuleGenerator() (ModuleGenerator, error) {
	switch m {
	case ModuleTypeProject:
		return ProjectGenerator{}, nil
	case ModuleTypeCluster:
		return ClustersGenerator{}, nil
	}
	return nil, fmt.Errorf("invalid module type: %s", m)
}

type ResourceType int

const (
	ResourceTypeOrganization ResourceType = iota
	ResourceTypeProject
	// TODO@non-spike: See project_generator.go
	// ResourceTypeProjectLimits
	ResourceTypeProjectSettings
	ResourceTypeProjectIPAccessList
	ResourceTypeProjectMaintenanceWindow
	ResourceTypeClusters
)

type ResourceStore struct {
	Organization             *admin.AtlasOrganization
	Project                  *admin.Group
	ProjectLimits            []admin.DataFederationLimit
	ProjectSettings          *admin.GroupSettings
	ProjectIPAccessList      *admin.PaginatedNetworkAccess
	ProjectMaintenanceWindow *admin.GroupMaintenanceWindow
	Clusters                 []*admin.ClusterDescription20240805
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
	Providers    []ProviderRequirement
	ImportBlocks []*ImportBlock
	ModuleBlocks []*ModuleBlock
	Variables    []*Variable
	// TODO add outputs
	// OutputBlocks []*OutputBlock // nolint:gocritic
	TerraformVersion Version
}

type ProviderRequirement struct {
	ProviderType ProviderType
	Version      Version
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
	Attributes []Attribute
	Version    Version
}

type ProviderType string

const (
	ProviderTypeAtlas ProviderType = "mongodbatlas"
)

type ImportBlock struct {
	ID      string
	To      string
	ForEach []string
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
}

type Variable struct {
	Value        cty.Value
	Type         cty.Type
	DefaultValue *cty.Value
	Name         string
	Description  string
}

type ProviderInfo struct {
	Name       string
	Source     string
	Attributes []Attribute
	Version    Version
}

type GenerateVersionsResult struct {
	Blocks    []*ProviderInfo
	Variables []*Variable
	TFVersion Version
}
