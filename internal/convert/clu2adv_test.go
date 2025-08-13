package convert_test

import (
	"strings"
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
)

func TestClusterToAdvancedCluster(t *testing.T) {
	runConvertTests(t, "clu2adv", func(testName string, inConfig []byte) ([]byte, error) {
		includeMoved := strings.Contains(testName, "includeMoved")
		return convert.ClusterToAdvancedCluster(inConfig, includeMoved)
	})
}
