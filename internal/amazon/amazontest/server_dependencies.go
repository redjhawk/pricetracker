package amazontest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// helperImportPath is the import path of this test helper package.
const helperImportPath = "pricefollower.local/internal/amazon/amazontest"

// listServerDependencies runs go list -deps ./cmd/... in dir, with env added to the process
// environment, and returns its standard output. Its error holds the Go tool's message.
func listServerDependencies(dir string, env []string) (string, error) {
	cmd := exec.Command("go", "list", "-deps", "./cmd/...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), fmt.Errorf("go list -deps ./cmd/...: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	if err != nil {
		return string(out), fmt.Errorf("go list -deps ./cmd/...: %w", err)
	}
	return string(out), nil
}

// checkServerDependencies fails when the listing could not be made or holds the test helper's
// import path as one of its lines.
func checkServerDependencies(listing string, listErr error) error {
	if listErr != nil {
		return fmt.Errorf("list the server dependencies: %w", listErr)
	}
	for _, line := range strings.Split(listing, "\n") {
		if strings.TrimSpace(line) == helperImportPath {
			return fmt.Errorf("the server depends on the test helper %s: a program under cmd/ imports it, directly or through another package (go list -deps ./cmd/...); only _test.go files may import it", helperImportPath)
		}
	}
	return nil
}
