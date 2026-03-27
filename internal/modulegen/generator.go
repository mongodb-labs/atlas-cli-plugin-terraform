package modulegen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"github.com/zclconf/go-cty/cty"
	"go.mongodb.org/atlas-sdk/v20250312014/admin"
)

type AtlasClientArgs struct {
	HTTPClient   *http.Client
	AtlasBaseURL string
	UserAgent    string
}

type GenArgs struct {
	InputPath  string
	OutputPath string
}

func Run(ctx context.Context, args *GenArgs, clientArgs *AtlasClientArgs) error {
	_, _ = logger.Debugln("[modulegen] Run")

	// == Parse input ==
	_, _ = logger.Infof("Reading input from %s...\n", args.InputPath)
	var input Input
	if err := parseInput(args.InputPath, &input); err != nil {
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
		_, _ = logger.Warningln("invalid input:")
		for field, moduleNames := range invalidFieldsPerModule {
			_, _ = logger.Warningf("\t[%s] `%s` missing or invalid\n", strings.Join(moduleNames, ", "), field)
		}
		return errors.New("invalid input")
	}

	// == Gather resources to fetch ==
	// It is currently safe to assume that there is only one resource of each type (except clusters), so gathering
	// the resource type is enough in this step. If this assumption is no longer true, we can collect resource type &
	// id instead.
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

	// == Generate internal modules & versions structures ==
	generatedModules := make([]*GenerateModuleResult, len(generators))
	for i, generator := range generators {
		generatedModules[i], err = generator.Generate(&input, resourceStore)
		if err != nil {
			return fmt.Errorf("failed to generate module %s: %w", string(generator.ModuleType()), err)
		}
	}
	generatedVersions := generateVersions(generatedModules)

	// == Render and write output files ==
	_, _ = logger.Infoln("Generating output...")

	// read/write/execute for owner. read/execute for group and others.
	const fileDirPermissions = 0o755
	if err = os.MkdirAll(args.OutputPath, fileDirPermissions); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", args.OutputPath, err)
	}

	// Module files
	for _, generatedModule := range generatedModules {
		var buffer bytes.Buffer
		// Note: very easy to collect all imports and write them to a separate import.tf file instead.
		buffer.Write(RenderImportBlocks(generatedModule.ImportBlocks))
		buffer.WriteByte('\n')
		buffer.Write(RenderModuleBlocks(generatedModule.ModuleBlocks))

		filePath := filepath.Join(args.OutputPath, string(generatedModule.ModuleType)+".tf")
		if err := os.WriteFile(filePath, buffer.Bytes(), fileDirPermissions); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	{ // versions.tf
		// Render and write
		filePath := filepath.Join(args.OutputPath, "versions.tf")
		rendered := RenderVersionsAndProviders(generatedVersions.TFVersion, generatedVersions.Blocks)
		if err := os.WriteFile(filePath, rendered, fileDirPermissions); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	{ // variables.tf and terraform.tfvars
		// Collect all variables and deduplicate using the name set.
		// Provider variables go first, other variables follow the order in which they were generated within a module,
		// no extra sorting needed.
		var variables []*Variable
		variableNameSet := make(map[string]bool)

		for _, variable := range generatedVersions.Variables {
			if _, ok := variableNameSet[variable.Name]; ok {
				_, _ = logger.Debugf("[modulegen] Deduplicated variable: %s\n", variable.Name)
				continue
			}
			variableNameSet[variable.Name] = true
			variables = append(variables, variable)
		}

		for _, generatedModule := range generatedModules {
			for _, variable := range generatedModule.Variables {
				if _, ok := variableNameSet[variable.Name]; ok {
					_, _ = logger.Debugf("[modulegen] Deduplicated variable: %s\n", variable.Name)
					continue
				}
				variableNameSet[variable.Name] = true
				variables = append(variables, variable)
			}
		}

		// Render and write
		if len(variables) > 0 {
			rendered := RenderVariables(variables)

			variablesPath := filepath.Join(args.OutputPath, "variables.tf")
			if err := os.WriteFile(variablesPath, rendered.Blocks, fileDirPermissions); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", variablesPath, err)
			}

			tfvarsPath := filepath.Join(args.OutputPath, "terraform.tfvars")
			if err := os.WriteFile(tfvarsPath, rendered.Definitions, fileDirPermissions); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", tfvarsPath, err)
			}
		}
	}

	{ // IMPORT_GUIDE.md
		guideData := ImportGuideData{}
		for _, m := range generatedModules {
			guideData.ModuleTypes = append(guideData.ModuleTypes, m.ModuleType)
		}
		filePath := filepath.Join(args.OutputPath, "IMPORT_GUIDE.md")
		if err := os.WriteFile(filePath, RenderImportGuide(guideData), fileDirPermissions); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	_, _ = logger.Infof("Done! Output written to: %s\n", args.OutputPath)
	_, _ = logger.Infoln("See the IMPORT_GUIDE.md for next steps.")

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
		// Uncomment to see Atlas SDK debug logs when tool is run in debug mode.
		// admin.UseDebug(log.IsDebugLevel()) //nolint:gocritic
		admin.UseBaseURL(clientArgs.AtlasBaseURL),
		admin.UseHTTPClient(clientArgs.HTTPClient),
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
func fetchResources(
	ctx context.Context,
	clientArgs *AtlasClientArgs,
	input *Input,
	resourcesToFetch map[ResourceType]bool,
) (*ResourceStore, error) {
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
			_, _ = logger.Infof("Reading organization `%s` from MongoDB Atlas...\n", input.OrgID)
			org, _, err := clients.atlasClient.OrganizationsApi.GetOrg(ctx, input.OrgID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading organization `%s` from MongoDB Atlas: %w", input.OrgID, err)
			}
			resourceStore.Organization = org
		case ResourceTypeProject:
			_, _ = logger.Infof("Reading project `%s` from MongoDB Atlas...\n", input.ProjectID)
			project, _, err := clients.atlasClient.ProjectsApi.GetGroup(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.Project = project
		/* TODO@non-spike: See project_generator.go
		case ResourceTypeProjectLimits:
			_, _ = logger.Infof("Reading project limits for `%s` from MongoDB Atlas...\n", input.ProjectID)
			projectLimits, _, err := clients.atlasClient.ProjectsApi.ListGroupLimits(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project limits for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.ProjectLimits = projectLimits
		*/
		case ResourceTypeProjectSettings:
			_, _ = logger.Infof("Reading project settings for `%s` from MongoDB Atlas...\n", input.ProjectID)
			ps, _, err := clients.atlasClient.ProjectsApi.GetGroupSettings(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project settings for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.ProjectSettings = ps
		case ResourceTypeProjectIPAccessList:
			_, _ = logger.Infof("Reading project IP access list for `%s` from MongoDB Atlas...\n", input.ProjectID)
			list, _, err := clients.atlasClient.ProjectIPAccessListApi.ListAccessListEntries(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf("error reading project IP access list for `%s` from MongoDB Atlas: %w", input.ProjectID, err)
			}
			resourceStore.ProjectIPAccessList = list
		case ResourceTypeProjectMaintenanceWindow:
			_, _ = logger.Infof("Reading project maintenance window for `%s` from MongoDB Atlas...\n", input.ProjectID)
			mw, _, err := clients.atlasClient.MaintenanceWindowsApi.GetMaintenanceWindow(ctx, input.ProjectID).Execute()
			if err != nil {
				return nil, fmt.Errorf(
					"error reading project maintenance window for `%s` from MongoDB Atlas: %w", input.ProjectID, err,
				)
			}
			resourceStore.ProjectMaintenanceWindow = mw
		case ResourceTypeClusters:
			_, _ = logger.Infof("Reading clusters [`%s`] from MongoDB Atlas...\n", strings.Join(input.ClusterNames, "`, `"))
			clusters := make([]*admin.ClusterDescription20240805, len(input.ClusterNames))
			for i, clusterName := range input.ClusterNames {
				cluster, _, err := clients.atlasClient.ClustersApi.GetCluster(ctx, input.ProjectID, clusterName).Execute()
				if err != nil {
					return nil, fmt.Errorf("error reading cluster `%s` from MongoDB Atlas: %w", clusterName, err)
				}
				clusters[i] = cluster
			}
			resourceStore.Clusters = clusters
		}
	}
	return &resourceStore, nil
}

func generateVersions(generatedModules []*GenerateModuleResult) GenerateVersionsResult {
	// Highest required terraform version
	var tfVersion Version
	// Collect all provider requirements and deduplicate by type, taking the one with the highest required version.
	// Providers follow the order in which they were generated within a module, no extra sorting needed.
	var providers []ProviderRequirement
	providerMap := make(map[ProviderType]*ProviderRequirement) // Ptr to providers slice

	for _, generatedModule := range generatedModules {
		for _, provider := range generatedModule.Providers {
			if existing, ok := providerMap[provider.ProviderType]; ok {
				pMajor, pMinor := provider.Version.Major, provider.Version.Minor
				eMajor, eMinor := existing.Version.Major, existing.Version.Minor
				if pMajor > eMajor || (pMajor == eMajor && pMinor > eMinor) {
					*existing = provider
				}
				_, _ = logger.Debugf("[modulegen] Deduplicated provider: %s %s\n", provider.ProviderType, existing.Version)
			} else {
				providers = append(providers, provider)
				providerMap[provider.ProviderType] = &providers[len(providers)-1]
			}
		}

		tfMajor, tfMinor := generatedModule.TerraformVersion.Major, generatedModule.TerraformVersion.Minor
		if tfMajor > tfVersion.Major || (tfMajor == tfVersion.Major && tfMinor > tfVersion.Minor) {
			tfVersion = generatedModule.TerraformVersion
			_, _ = logger.Debugf("[modulegen] Required Terraform version updated to: %s\n", tfVersion)
		}
	}

	var blocks []*ProviderInfo
	var variables []*Variable
	for _, p := range providers {
		switch p.ProviderType { //nolint:gocritic
		case ProviderTypeAtlas:
			clientID := &Variable{
				Name:         "atlas_client_id",
				Description:  "Atlas Service Account Client ID",
				Type:         cty.String,
				Value:        cty.StringVal(""),
				DefaultValue: new(cty.StringVal("")),
			}
			clientSecret := &Variable{
				Name:         "atlas_client_secret",
				Description:  "Atlas Service Account Client Secret",
				Type:         cty.String,
				Value:        cty.StringVal(""),
				DefaultValue: new(cty.StringVal("")),
			}
			variables = append(variables, clientID, clientSecret)
			blocks = append(blocks, &ProviderInfo{
				Name:    "mongodbatlas",
				Source:  "mongodb/mongodbatlas",
				Version: p.Version,
				Attributes: []Attribute{
					{Name: "client_id", Value: AttributeValue{Variable: clientID}},
					{Name: "client_secret", Value: AttributeValue{Variable: clientSecret}},
				},
			})
		}
	}

	return GenerateVersionsResult{
		TFVersion: tfVersion,
		Blocks:    blocks,
		Variables: variables,
	}
}
