package amazontest

import "errors"

// helperImportPath is the import path of this test helper package.
const helperImportPath = "pricefollower.local/internal/amazon/amazontest"

// listServerDependencies lists the dependencies of the programs under dir/cmd. Not implemented yet.
func listServerDependencies(dir string, env []string) (string, error) {
	return "", nil
}

// checkServerDependencies judges a dependency listing. Not implemented yet.
func checkServerDependencies(listing string, listErr error) error {
	return errors.New("not implemented")
}
