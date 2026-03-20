package modulegen

import (
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
		return ClusterGenerator{}, nil
	}
	return nil, fmt.Errorf("invalid module type: %s", m)
}

type RunState struct {
	atlasClient *admin.APIClient
	// awsClient, gcpClient, azureClient
}

type ModuleGenArgs struct {
	InputPath  string
	OutputPath string
}

type Input struct {
	Modules      []ModuleType `hcl:"modules"`
	OrgID        string       `hcl:"org_id,optional"`
	ProjectID    string       `hcl:"project_id,optional"`
	ClusterNames []string     `hcl:"cluster_names,optional"`
}

func Run(httpClient *http.Client, userAgent string, args ModuleGenArgs) error {
	log.Debug("[modulegen] Run\n")

	runState := &RunState{}
	var err error

	// == Parse input ==
	log.Debug("[modulegen] Parsing input...\n")
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
		log.Warning("invalid input:\n")
		for field, moduleNames := range invalidFieldsPerModule {
			log.Warningf("\t[%s] `%s` missing or invalid\n", strings.Join(moduleNames, ", "), field)
		}
		return errors.New("invalid input")
	}

	// == Gather resources to fetch ==
	// TODO

	// == Build clients ==
	// Note: Assuming that we always build the Atlas client
	runState.atlasClient, err = newAtlasClient(httpClient, userAgent)
	if err != nil {
		return err
	}

	// == Fetch resources ==

	// == Generate (internal structure) ==

	// == Generate output files ==

	return nil
}

func newAtlasClient(httpClient *http.Client, userAgent string) (*admin.APIClient, error) {
	return admin.NewClient(
		admin.UseHTTPClient(httpClient),
		admin.UseUserAgent(userAgent),
	)
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

type ModuleGenerator interface {
	ModuleType() ModuleType
	// CheckInput returns a list of missing or invalid input fields for the module
	CheckInput(input *Input) []string
}

var _ ModuleGenerator = ProjectGenerator{}

type ProjectGenerator struct{}

func (g ProjectGenerator) ModuleType() ModuleType {
	return ModuleTypeProject
}

func (g ProjectGenerator) CheckInput(input *Input) []string {
	var fields []string
	if input.OrgID == "" {
		fields = append(fields, "org_id")
	}
	if input.ProjectID == "" {
		fields = append(fields, "project_id")
	}
	return fields
}

var _ ModuleGenerator = ClusterGenerator{}

type ClusterGenerator struct{}

func (g ClusterGenerator) ModuleType() ModuleType {
	return ModuleTypeCluster
}

func (g ClusterGenerator) CheckInput(input *Input) []string {
	var fields []string
	if input.ProjectID == "" {
		fields = append(fields, "project_id")
	}
	if len(input.ClusterNames) == 0 {
		fields = append(fields, "cluster_names")
	}
	return fields
}
