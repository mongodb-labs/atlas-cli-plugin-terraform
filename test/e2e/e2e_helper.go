package e2e

import (
	"context"
	"os/exec"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunTF(args ...string) (string, error) {
	args = append([]string{"tf"}, args...)
	cmd := exec.CommandContext(context.Background(), "atlas", args...)
	resp, err := cmd.CombinedOutput()
	return string(resp), err
}

func RunClu2Adv(args ...string) (string, error) {
	args = append([]string{"clu2adv"}, args...)
	return RunTF(args...)
}

func CompareFiles(t *testing.T, fs afero.Fs, file1, file2 string) {
	t.Helper()
	data1, err1 := afero.ReadFile(fs, file1)
	require.NoError(t, err1)
	data2, err2 := afero.ReadFile(fs, file2)
	require.NoError(t, err2)
	assert.Equal(t, string(data1), string(data2))
}
