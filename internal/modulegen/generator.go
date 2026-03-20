package modulegen

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/log"
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

type ResourceType string

const (
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeProject      ResourceType = "project"
	ResourceTypeClusters     ResourceType = "clusters"
)

type ResourceStore struct {
	organization *admin.AtlasOrganization
	project      *admin.Group
	clusters     []admin.ClusterDescription20240805
}

type AtlasClientArgs struct {
	AtlasBaseUrl string
	UserAgent    string
	HttpClient   *http.Client
}

type ModuleGenArgs struct {
	InputPath  string
	OutputPath string
}

func Run(ctx context.Context, args *ModuleGenArgs, clientArgs *AtlasClientArgs) error {
	var err error
	_, _ = log.Debugln("[modulegen] Run")

	// == Parse input ==
	_, _ = log.Debugln("[modulegen] Parsing input...")
	var input Input
	if err = parseInput(args.InputPath, &input); err != nil {
		return err
	}
	if len(input.Modules) == 0 {
		return errors.New("at least one module must be provided")
	}

	// == Init generators ==
	var generators []ModuleGenerator
	for _, module := range input.Modules {
		moduleGenerator, err := module.GetModuleGenerator()
		if err != nil {
			return err
		}
		generators = append(generators, moduleGenerator)
	}

	// == Check inputs ==
	// Example error: "[project, cluster] `project_id` missing or invalid"
	invalidFieldsPerModule := map[string][]string{}
	for _, generator := range generators {
		for _, field := range generator.CheckInput(&input) {
			invalidFieldsPerModule[field] = append(invalidFieldsPerModule[field], string(generator.ModuleType()))
		}
	}
	if len(invalidFieldsPerModule) > 0 {
		_, _ = log.Warningln("invalid input:")
		for field, moduleNames := range invalidFieldsPerModule {
			_, _ = log.Warningf("\t[%s] `%s` missing or invalid\n", strings.Join(moduleNames, ", "), field)
		}
		return errors.New("invalid input")
	}

	// == Gather resources to fetch ==
	// It is currently safe to assume that there is only one resource of each type (except clusters), so gathering
	// the resource type is enough in this step. If this assumption is no longer true, we can collect resource type & id instead.
	resourcesToFetch := map[ResourceType]bool{}
	for _, generator := range generators {
		generator.GetResourcesToFetch(&input, resourcesToFetch)
	}

	// == Fetch resources ==
	// Fetch all resources needed for generating the requested modules.
	// No network calls are made outside of this step.
	resourceStore, err := fetchResources(ctx, clientArgs, &input, resourcesToFetch)
	if err != nil {
		return err
	}

	// == Generate (internal structure) ==
	// TODO
	for _, generator := range generators {
		generator.Generate(&input, resourceStore)
	}

	// == Generate output files ==
	// TODO

	return nil
}

func parseInput(inputPath string, input *Input) error {
	src, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	file, diags := hclsyntax.ParseConfig(src, inputPath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil {
		return fmt.Errorf("failed to parse input file: %w", diags)
	}

	if diags = gohcl.DecodeBody(file.Body, nil, input); diags.HasErrors() {
		return fmt.Errorf("failed to decode input file: %w", diags)
	}

	return nil
}

type Clients struct {
	atlasClient *admin.APIClient
	// awsClient, gcpClient, azureClient
}

func initClients(clientArgs *AtlasClientArgs, resourcesToFetch map[ResourceType]bool) (Clients, error) {
	var err error
	clients := Clients{}

	// Note: Assuming that we always need the Atlas client
	clients.atlasClient, err = admin.NewClient(
		//admin.UseDebug(log.IsDebugLevel()) // Uncomment to see Atlas SDK debug logs when tool is run in debug mode.
		admin.UseBaseURL(clientArgs.AtlasBaseUrl),
		admin.UseHTTPClient(clientArgs.HttpClient),
		admin.UseUserAgent(clientArgs.UserAgent),
	)
	if err != nil {
		return clients, fmt.Errorf("failed to create atlas client: %w", err)
	}

	// TODO: Initialize other clients based on the resources to fetch

	return clients, nil
}

// fetchResources fetches all resources needed for generating the requested modules and populates them in resourceStore.
// Note: For testing, mock this whole function. No network calls are made outside of this function.
func fetchResources(ctx context.Context, clientArgs *AtlasClientArgs, input *Input, resourcesToFetch map[ResourceType]bool) (*ResourceStore, error) {
	clients, err := initClients(clientArgs, resourcesToFetch)
	if err != nil {
		return nil, err
	}

	// TODO: Parallelize
	// TODO: Parallelize
	// TODO: Parallelize
	resourceStore := ResourceStore{}
	for resourceType := range resourcesToFetch {
		switch resourceType {
		case ResourceTypeOrganization:
			_, _ = log.Infof("Reading organization `%s` from MongoDB Atlas...\n", input.OrgID)
			org, _, err := clients.atlasClient.OrganizationsApi.GetOrg(ctx, input.OrgID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading organization `%s` from MongoDB Atlas: %w", input.OrgID, err)
			}
			resourceStore.organization = org
		case ResourceTypeProject:
			_, _ = log.Infof("Reading project `%s` from MongoDB Atlas...\n", input.ProjectID)
			project, _, err := clients.atlasClient.ProjectsApi.GetGroup(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.project = project
		case ResourceTypeClusters:
			_, _ = log.Infof("Reading clusters [`%s`] from MongoDB Atlas...\n", strings.Join(input.ClusterNames, "`, `"))
			clusters := make([]*admin.ClusterDescription20240805, len(input.ClusterNames))
			for i, clusterName := range input.ClusterNames {
				cluster, _, err := clients.atlasClient.ClustersApi.GetCluster(ctx, input.ProjectID, clusterName).Execute()
				if err != nil {
					return nil, fmt.Errorf("error reading cluster `%s` from MongoDB Atlas: %w", clusterName, err)
				}
				clusters[i] = cluster
			}
		}
	}
	return &resourceStore, nil
}

// ==== Generators ====

type ModuleGenerator interface {
	ModuleType() ModuleType
	// CheckInput returns a list of missing or invalid input fields for the module
	CheckInput(input *Input) []string
	// GetResourcesToFetch returns a map (set) of the resource types that should be fetched for the module given the input.
	// Note: If necessary, we can use the bool to indicate whether the resource is required or optional (fetch wouldn't fail for optionals)
	GetResourcesToFetch(input *Input, resourcesToFetch map[ResourceType]bool)
	Generate(input *Input, store *ResourceStore) error
}

// === Project Generator ===

var _ ModuleGenerator = ProjectGenerator{}

type ProjectGenerator struct{}

func (g ProjectGenerator) Generate(input *Input, store *ResourceStore) error {
	// TODO
	return nil
}

func (g ProjectGenerator) ModuleType() ModuleType {
	return ModuleTypeProject
}

func (g ProjectGenerator) CheckInput(input *Input) []string {
	var fields []string
	// TODO@remove: We don't actually need the org id for the project module. We can get it from the fetched project.
	if input.OrgID == "" {
		fields = append(fields, "org_id")
	}
	if input.ProjectID == "" {
		fields = append(fields, "project_id")
	}
	return fields
}

func (g ProjectGenerator) GetResourcesToFetch(input *Input, resourcesToFetch map[ResourceType]bool) {
	// TODO@remove: We don't actually need to fetch the org for the project module. Doing it for now just for testing.
	resourcesToFetch[ResourceTypeOrganization] = true
	resourcesToFetch[ResourceTypeProject] = true
}

// === Cluster Generator ===

var _ ModuleGenerator = ClustersGenerator{}

type ClustersGenerator struct{}

func (g ClustersGenerator) Generate(input *Input, store *ResourceStore) error {
	// TODO
	return nil
}

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
	//TODO@remove: no need to fetch the project for the cluster module. Just testing.
	resourcesToFetch[ResourceTypeProject] = true
	resourcesToFetch[ResourceTypeClusters] = true
}
