package modulegen

import "errors"

var _ ModuleGenerator = ClustersGenerator{}

type ClustersGenerator struct{}

func (g ClustersGenerator) ModuleType() ModuleType {
	return ModuleTypeCluster
}

func (g ClustersGenerator) CheckInput(input *Input) []string {
	invalidFields := CheckRequiredInputStr([]RequiredStr{
		{input.ProjectID, "project_id"},
	})
	if len(input.ClusterNames) == 0 {
		invalidFields = append(invalidFields, "cluster_names")
	}
	return invalidFields
}

func (g ClustersGenerator) GetResourcesToFetch(_ *Input, resources *ResourcesToFetch) {
	// TODO@remove: no need to fetch the project for the cluster module. Just testing.
	resources.Atlas[AtlasResourceTypeProject] = true
	resources.Atlas[AtlasResourceTypeClusters] = true
}

func (g ClustersGenerator) Generate(input *Input, store *ResourceStore) (*GenerateModuleResult, error) {
	return nil, errors.New("not implemented")
}
