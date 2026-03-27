package modulegen

import "errors"

var _ ModuleGenerator = ClustersGenerator{}

type ClustersGenerator struct{}

func (g ClustersGenerator) ModuleType() ModuleType {
	return ModuleTypeCluster
}

func (g ClustersGenerator) CheckInput(input *Input) []string {
	var fields []string
	if input.ProjectID == "" {
		fields = append(fields, "project_id")
	}
	if len(input.ClusterNames) == 0 {
		fields = append(fields, "cluster_names")
	}
	return fields
}

func (g ClustersGenerator) GetResourcesToFetch(input *Input, resourcesToFetch map[ResourceType]bool) {
	// TODO@remove: no need to fetch the project for the cluster module. Just testing.
	resourcesToFetch[ResourceTypeProject] = true
	resourcesToFetch[ResourceTypeClusters] = true
}

func (g ClustersGenerator) Generate(input *Input, store *ResourceStore) (*GenerateModuleResult, error) {
	return nil, errors.New("not implemented")
}
