package hcl_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mongodb-labs/atlas-cli-plugin-terraform/internal/hcl"
	"github.com/sebdah/goldie/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusterToAdvancedCluster(t *testing.T) {
	const (
		root        = "testdata/clu2adv"
		inSuffix    = ".in.tf"
		outSuffix   = ".out.tf"
		errFilename = "errors.json"
	)
	fs := afero.NewOsFs()
	errMap := make(map[string]string)
	errContent, err := afero.ReadFile(fs, filepath.Join(root, errFilename))
	require.NoError(t, err)
	err = json.Unmarshal(errContent, &errMap)
	require.NoError(t, err)
	g := goldie.New(t,
		goldie.WithFixtureDir(root),
		goldie.WithNameSuffix(outSuffix))
	pattern := filepath.Join(root, "*"+inSuffix)
	inputFiles, err := afero.Glob(fs, pattern)
	require.NoError(t, err)
	assert.NotEmpty(t, inputFiles)
	for _, inputFile := range inputFiles {
		testName := strings.TrimSuffix(filepath.Base(inputFile), inSuffix)
		inConfig, err := afero.ReadFile(fs, inputFile)
		require.NoError(t, err)
		outConfig, err := hcl.ClusterToAdvancedCluster(inConfig)
		if err == nil {
			g.Assert(t, testName, outConfig)
		} else {
			errMsg, found := errMap[testName]
			assert.True(t, found, "error not found for test %s", testName)
			assert.Contains(t, err.Error(), errMsg)
		}
	}
}
