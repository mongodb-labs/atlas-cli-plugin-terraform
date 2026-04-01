package modulegen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"github.com/zclconf/go-cty/cty"
)

type GenArgs struct {
	InputPath    string
	OutputPath   string
	AtlasBaseURL string
}

func Run(ctx context.Context, args *GenArgs, client Client) error {
	logger.Debugln("[modulegen] Run")

	// == Parse input ==
	logger.Infof("Reading input from %s...\n", args.InputPath)
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
		logger.Warningln("invalid input:")
		for field, moduleNames := range invalidFieldsPerModule {
			logger.Warningf("\t[%s] `%s` missing or invalid\n", strings.Join(moduleNames, ", "), field)
		}
		return errors.New("invalid input")
	}

	// == Gather resources to fetch ==
	// It is currently safe to assume that there is only one resource of each type (except clusters), so gathering
	// the resource type is enough in this step. If this assumption is no longer true, we can collect resource type &
	// id instead.
	resourcesToFetch := NewResourcesToFetch()
	for _, generator := range generators {
		generator.GetResourcesToFetch(&input, resourcesToFetch)
	}

	// == Fetch resources ==
	// Fetch all resources needed for generating the requested modules.
	// No network calls are made outside of this step.
	resourceStore, err := client.FetchResources(ctx, &input, resourcesToFetch)
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
	generatedVersions := generateVersions(args.AtlasBaseURL, &input, generatedModules)

	// == Render and write output files ==
	logger.Infoln("Generating output...")

	const dirPermissions, filePermissions = 0o700, 0o600 // owner: read/write/execute for dir, read/write for file.
	if err = os.MkdirAll(args.OutputPath, dirPermissions); err != nil {
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
		if err := os.WriteFile(filePath, buffer.Bytes(), filePermissions); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	{ // versions.tf
		// Render and write
		filePath := filepath.Join(args.OutputPath, "versions.tf")
		rendered := RenderVersionsAndProviders(generatedVersions.TFVersion, generatedVersions.Blocks)
		if err := os.WriteFile(filePath, rendered, filePermissions); err != nil {
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
				logger.Debugf("[modulegen] Deduplicated variable: %s\n", variable.Name)
				continue
			}
			variableNameSet[variable.Name] = true
			variables = append(variables, variable)
		}

		for _, generatedModule := range generatedModules {
			for _, variable := range generatedModule.Variables {
				if _, ok := variableNameSet[variable.Name]; ok {
					logger.Debugf("[modulegen] Deduplicated variable: %s\n", variable.Name)
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
			if err := os.WriteFile(variablesPath, rendered.Blocks, filePermissions); err != nil {
				return fmt.Errorf("failed to write to file %s: %w", variablesPath, err)
			}

			tfvarsPath := filepath.Join(args.OutputPath, "terraform.tfvars")
			if err := os.WriteFile(tfvarsPath, rendered.Definitions, filePermissions); err != nil {
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
		if err := os.WriteFile(filePath, RenderImportGuide(guideData), filePermissions); err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, err)
		}
	}

	logger.Infof("Done! Output written to: %s\n", args.OutputPath)
	logger.Infoln("See the IMPORT_GUIDE.md for next steps.")

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

func generateVersions(
	atlasBaseURL string, input *Input, generatedModules []*GenerateModuleResult,
) GenerateVersionsResult {
	// Highest required terraform version
	var tfVersion Version
	// Collect all provider requirements and deduplicate by type, taking the one with the highest required version.
	// Providers follow the order in which they were generated within a module, no extra sorting needed.
	var providers []ProviderRequirement
	providerMap := make(map[ProviderType]*ProviderRequirement) // Ptr to providers slice

	for _, generatedModule := range generatedModules {
		for _, provider := range generatedModule.Providers {
			if existing, ok := providerMap[provider.ProviderType]; ok {
				if provider.Version.GreaterThan(existing.Version) {
					*existing = provider
				}
				logger.Debugf("[modulegen] Deduplicated provider: %s %s\n", provider.ProviderType, existing.Version)
			} else {
				providers = append(providers, provider)
				providerMap[provider.ProviderType] = &providers[len(providers)-1]
			}
		}

		if generatedModule.TerraformVersion.GreaterThan(tfVersion) {
			tfVersion = generatedModule.TerraformVersion
			logger.Debugf("[modulegen] Required Terraform version updated to: %s\n", tfVersion)
		}
	}

	var blocks []*ProviderInfo
	var variables []*Variable
	for _, p := range providers {
		switch p.ProviderType {
		case ProviderTypeAtlas:
			var attributes []Attribute
			if atlasBaseURL != CloudServiceURL {
				if atlasBaseURL == CloudGovServiceURL {
					attributes = append(attributes, BoolAttr("is_mongodbgov_cloud", true))
				} else {
					attributes = append(attributes, StringAttr("base_url", atlasBaseURL))
				}
			}
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
			attributes = append(attributes, VarAttr("client_id", clientID), VarAttr("client_secret", clientSecret))
			blocks = append(blocks, &ProviderInfo{
				Name:       "mongodbatlas",
				Source:     "mongodb/mongodbatlas",
				Version:    p.Version,
				Attributes: attributes,
			})
		case ProviderTypeAWS:
			blocks = append(blocks, &ProviderInfo{
				Name:       "aws",
				Source:     "hashicorp/aws",
				Version:    p.Version,
				Attributes: []Attribute{StringAttr("region", input.MultiSDK.AWSRegion)},
			})
		case ProviderTypeAzureRM:
			subscriptionID := &Variable{
				Name:        "azure_subscription_id",
				Description: "Azure subscription ID",
				Type:        cty.String,
				Value:       cty.StringVal(input.MultiSDK.AzureSubscriptionID),
			}
			variables = append(variables, subscriptionID)
			blocks = append(blocks, &ProviderInfo{
				Name:    "azurerm",
				Source:  "hashicorp/azurerm",
				Version: p.Version,
				Attributes: []Attribute{
					VarAttr("subscription_id", subscriptionID),
					BlockAttr("features", nil),
				},
			})
		case ProviderTypeAzureAD:
			blocks = append(blocks, &ProviderInfo{
				Name:    "azuread",
				Source:  "hashicorp/azuread",
				Version: p.Version,
			})
		case ProviderTypeGoogle:
			projectID := &Variable{
				Name:        "gcp_project_id",
				Description: "GCP project ID",
				Type:        cty.String,
				Value:       cty.StringVal(input.MultiSDK.GCPProjectID),
			}
			variables = append(variables, projectID)
			blocks = append(blocks, &ProviderInfo{
				Name:    "google",
				Source:  "hashicorp/google",
				Version: p.Version,
				Attributes: []Attribute{
					VarAttr("project", projectID),
					StringAttr("region", input.MultiSDK.GCPRegion),
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
