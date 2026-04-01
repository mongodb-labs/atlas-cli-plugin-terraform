package moduleimport

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/flags"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/logger"
	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/modulegen"
	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-core/transport"
	"github.com/spf13/cobra"
)

// TODO@non-spike: Support tracking plugin versions, used in UserAgent header.
var Version = "dev"

type Opts struct {
	httpClient   *http.Client
	input        string
	output       string
	atlasBaseURL string
}

func Builder() *cobra.Command {
	opts := &Opts{}
	cmd := &cobra.Command{
		Use:     "module-import",
		Short:   "Generate Terraform module configurations",
		Long:    "Generate Terraform module configurations to import existing infrastructure",
		PreRunE: opts.PreRun,
		RunE:    opts.Run,
	}

	cmd.Flags().StringVarP(
		&opts.input, flags.Input, flags.InputShort, "",
		"path to the input file",
	)
	_ = cmd.MarkFlagRequired(flags.Input)
	cmd.Flags().StringVarP(
		&opts.output, flags.Output, flags.OutputShort, "",
		"path where to the directory where to generate the output files",
	)
	_ = cmd.MarkFlagRequired(flags.Output)
	return cmd
}

func (opts *Opts) PreRun(cmd *cobra.Command, args []string) error {
	logger.Debugln("[module-import] PreRunE")

	profile, err := config.LoadAtlasCLIConfig()
	if err != nil {
		return err
	}

	// Use user-overridden url, otherwise if gov use gov url, otherwise use default.
	if opts.atlasBaseURL = profile.OpsManagerURL(); opts.atlasBaseURL == "" {
		if profile.Service() == config.CloudGovService {
			opts.atlasBaseURL = modulegen.CloudGovServiceURL
		} else {
			opts.atlasBaseURL = modulegen.CloudServiceURL
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

func (opts *Opts) Run(cmd *cobra.Command, args []string) error {
	logger.Debugln("[module-import] RunE")
	err := modulegen.Run(
		cmd.Context(),
		&modulegen.GenArgs{
			InputPath:    opts.input,
			OutputPath:   opts.output,
			AtlasBaseURL: opts.atlasBaseURL,
		},
		&modulegen.DefaultClient{
			AtlasBaseURL: opts.atlasBaseURL,
			// TODO@non-spike: Look into differentiating the plugin's UserAgent from the cli one
			UserAgent:  config.UserAgent(Version),
			HTTPClient: opts.httpClient,
		},
	)
	return err
}
