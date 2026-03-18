package modulegen

import (
	"net/http"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/log"
	"go.mongodb.org/atlas-sdk/v20250312014/admin"
)

func PreRun() error {
	log.Debug("[modulegen] PreRun\n")

	// Delete if prerun is not required

	return nil
}

type RunState struct {
	atlasClient *admin.APIClient
	// awsClient, gcpClient, azureClient
}

func Run(httpClient *http.Client, userAgent string) error {
	log.Debug("[modulegen] Run\n")

	runState := &RunState{}
	var err error

	// Note: Assuming that we always build the Atlas client. For other SDKs, we'll be lazy. - Manu
	runState.atlasClient, err = newAtlasClient(httpClient, userAgent)
	if err != nil {
		return err
	}

	return nil
}

func newAtlasClient(httpClient *http.Client, userAgent string) (*admin.APIClient, error) {
	return admin.NewClient(
		admin.UseHTTPClient(httpClient),
		admin.UseUserAgent(userAgent),
	)
}
