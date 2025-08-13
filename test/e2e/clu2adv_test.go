package e2e_test

import (
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/test/e2e"
)

func TestClusterToAdvancedCluster(t *testing.T) {
	files := e2e.GetTestFiles(t, "clu2adv")
	fileExpectedMoved := files.GetCustomFilePath("expected_moved.tf")
	extraTests := map[string]e2e.TestCase{
		"include moved": {
			Args:   []string{"--file", files.FileIn, "--output", files.FileOut, "--includeMoved"},
			Assert: func(t *testing.T) { t.Helper(); e2e.CompareFiles(t, files.Fs, files.FileOut, fileExpectedMoved) },
		},
	}
	e2e.RunTests(t, "clu2adv", extraTests)
}
