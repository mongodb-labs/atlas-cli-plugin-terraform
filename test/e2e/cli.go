package e2e

import "os/exec"

func RunPlugin(args ...string) (string, error) {
	cmd := exec.Command("atlas", args...)
	resp, err := cmd.CombinedOutput()
	return string(resp), err
}
