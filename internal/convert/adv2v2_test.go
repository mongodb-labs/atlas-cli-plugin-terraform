package convert_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/convert"
)

func TestAdvancedClusterToV2(t *testing.T) {
	runConvertTests(t, "adv2v2", func(testName string, inConfig []byte) ([]byte, error) {
		return convert.AdvancedClusterToV2(inConfig)
	})
}
