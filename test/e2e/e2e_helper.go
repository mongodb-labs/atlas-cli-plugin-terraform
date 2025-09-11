package e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func RunTF(args ...string) (string, error) {
	// Ensure Atlas CLI storage warning is silenced before running tests as it is not enabled in GitHub Actions
	exec.Command("atlas", "config", "set", "silence_storage_warning", "true").Run()

	args = append([]string{"tf"}, args...)
	cmd := exec.CommandContext(context.Background(), "atlas", args...)
	resp, err := cmd.CombinedOutput()
	return string(resp), err
}

func RunTFCommand(command string, args ...string) (string, error) {
	args = append([]string{command}, args...)
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

type TestFiles struct {
	Fs             afero.Fs
	Prefix         string
	FileIn         string
	FileOut        string
	FileExpected   string
	FileUnexisting string
	CmdName        string
}

// GetTestFiles creates a TestFiles struct with standard file paths for the given command name
func GetTestFiles(t *testing.T, cmdName string) *TestFiles {
	t.Helper()
	cwd, err := os.Getwd()
	require.NoError(t, err)

	prefix := cwd + "/testdata/"
	files := &TestFiles{
		Fs:      afero.NewOsFs(),
		CmdName: cmdName,
		Prefix:  prefix,
	}
	files.FileIn = files.GetCustomFilePath("in.tf")
	files.FileOut = files.GetCustomFilePath("out.tf")
	files.FileExpected = files.GetCustomFilePath("expected.tf")
	files.FileUnexisting = files.GetCustomFilePath("unexisting.tf")
	return files
}

// GetCustomFilePath returns a file path using the testdata prefix
func (tf *TestFiles) GetCustomFilePath(suffix string) string {
	return tf.Prefix + tf.CmdName + "." + suffix
}

type TestCase struct {
	ExpectedErrContains string
	Assert              func(t *testing.T)
	Args                []string
}

// RunTests runs common parameter validation tests for both commands. Specific tests can be provided in extraTests.
func RunTests(t *testing.T, cmdName string, extraTests map[string]TestCase) {
	t.Helper()
	files := GetTestFiles(t, cmdName)
	commonTests := map[string]TestCase{
		"no params": {
			ExpectedErrContains: "required flag(s) \"file\", \"output\" not set",
		},
		"no input file": {
			Args:                []string{"--output", files.FileOut},
			ExpectedErrContains: "required flag(s) \"file\" not set",
		},
		"no output file": {
			Args:                []string{"--file", files.FileIn},
			ExpectedErrContains: "required flag(s) \"output\" not set",
		},
		"unexisting input file": {
			Args:                []string{"--file", files.FileUnexisting, "--output", files.FileOut},
			ExpectedErrContains: "file must exist: " + files.FileUnexisting,
		},
		"existing output file without replaceOutput flag": {
			Args:                []string{"--file", files.FileIn, "--output", files.FileExpected},
			ExpectedErrContains: "file must not exist: " + files.FileExpected,
		},
		"basic use": {
			Args:   []string{"--file", files.FileIn, "--output", files.FileOut},
			Assert: func(t *testing.T) { t.Helper(); CompareFiles(t, files.Fs, files.FileOut, files.FileExpected) },
		},
	}

	allTests := make(map[string]TestCase)
	for name, test := range commonTests {
		allTests[name] = test
	}
	for name, test := range extraTests {
		allTests[name] = test
	}

	for name, tc := range allTests {
		t.Run(name, func(t *testing.T) {
			resp, err := RunTFCommand(cmdName, tc.Args...)
			assert.Equal(t, tc.ExpectedErrContains == "", err == nil)
			if err == nil {
				assert.Empty(t, resp)
				if tc.Assert != nil {
					tc.Assert(t)
				}
			} else {
				assert.Contains(t, resp, tc.ExpectedErrContains)
			}
			_ = files.Fs.Remove(files.FileOut) // Ensure output file does not exist in case it was generated in some test case
		})
	}
}
