package moduleimport

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flags"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/log"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/modulegen"
	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-core/transport"
	"github.com/spf13/cobra"
)

const (
	CloudServiceURL    = "https://cloud.mongodb.com/"
	CloudGovServiceURL = "https://cloud.mongodbgov.com/"
)

// TODO@non-spike: Support tracking plugin versions, used in UserAgent header.
var Version = "dev"

type ModuleImportOpts struct {
	input        string
	output       string
	atlasBaseUrl string
	httpClient   *http.Client
}

func Builder() *cobra.Command {
	opts := &ModuleImportOpts{}
	cmd := &cobra.Command{
		Use:     "module-import",
		Short:   "Generate Terraform module configurations",
		Long:    "Generate Terraform module configurations to import existing infrastructure",
		PreRunE: opts.PreRun,
		RunE:    opts.Run,
	}

	cmd.Flags().StringVarP(&opts.input, flags.Input, flags.InputShort, "", "path to the input file")
	_ = cmd.MarkFlagRequired(flags.Input)
	cmd.Flags().StringVarP(&opts.output, flags.Output, flags.OutputShort, "", "path where to the directory where to generate the output files")
	_ = cmd.MarkFlagRequired(flags.Output)
	return cmd
}

func (opts *ModuleImportOpts) PreRun(cmd *cobra.Command, args []string) error {
	_, _ = log.Debugln("[module-import] PreRunE")

	profile, err := config.LoadAtlasCLIConfig()
	if err != nil {
		return err
	}

	// Use user-overridden url, otherwise if gov use gov url, otherwise use default.
	if opts.atlasBaseUrl = profile.OpsManagerURL(); opts.atlasBaseUrl == "" {
		if profile.Service() == config.CloudService {
			opts.atlasBaseUrl = CloudGovServiceURL
		} else {
			opts.atlasBaseUrl = CloudServiceURL
		}
	}

	// Check that Atlas credentials are configured.
	// IsAccessSet covers API Keys and Service Accounts. Token covers OAuth.
	if !profile.IsAccessSet() {
		token, _ := profile.Token()
		if token == nil {
			return errors.New("no Atlas credentials found")
		}
	}

	opts.httpClient, err = transport.HTTPClientFromProfile(profile, Version, transport.Default())
	if err != nil {
		return fmt.Errorf("failed to build HTTP client: %w", err)
	}
	return err
}

func (opts *ModuleImportOpts) Run(cmd *cobra.Command, args []string) error {
	_, _ = log.Debugln("[module-import] RunE")
	err := modulegen.Run(
		cmd.Context(),
		&modulegen.ModuleGenArgs{
			InputPath:  opts.input,
			OutputPath: opts.output,
		},
		&modulegen.AtlasClientArgs{
			AtlasBaseUrl: opts.atlasBaseUrl,
			UserAgent:    config.UserAgent(Version), // TODO@non-spike: Look into differentiating the plugin's UserAgent from the cli one
			HttpClient:   opts.httpClient,
		},
	)
	return err
}
