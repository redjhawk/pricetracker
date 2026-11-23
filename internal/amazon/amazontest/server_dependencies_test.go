package amazontest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #244: the server must not depend on the test helper. The check judges what
// go list -deps ./cmd/... reports instead of searching the source for the import path.
//
// Test cases of issue #244 proved outside this file (see the technical specification):
//   - TC-001: amazontest_test.go no longer walks cmd/ or internal/ and searches no source for the
//     helper's import path; TestAssertNoDownload_TC006_ChangeLeavesApplicationUntouched keeps only
//     the issue #208 check. Its tests run with go test -count=1 ./internal/amazon/amazontest/...
//   - TC-005: mutations of the real repository in a scratch worktree fail or pass through
//     TestServerDoesNotDependOnTestHelper; TC-003 below reproduces each mutation on fixture modules.
//   - TC-006: the diff outside doc/ is limited to internal/amazon/amazontest/ test code, and the
//     gates (go vet, go test, node --test, npm run build, npx playwright test) pass.
//
// Issue #251: the fixture modules compile. TestFixtureModulesBuild_Issue251_TC001 proves its TC-001;
// its other test cases are proved outside that test:
//   - TC-002: TestListServerDependencies_TC003_FixtureModules, rows a to e and the three corners of
//     fixtureCases, keep their subtest names and results.
//   - TC-003: rows f, g and h of the same test keep failing as listing failures with the same messages.
//   - TC-004: the diff outside doc/ is limited to this file, and the gates pass.

const (
	dependsOnHelper = "server depends on"
	toolError       = "exit status 1: cmd/pricefollower/main.go:3:2: package x is not in std"
)

// fixtureEnv lets go list run on a fixture module without network, workspace or toolchain download.
var fixtureEnv = []string{"GOFLAGS=-mod=mod", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off"}

func TestHelperImportPath(t *testing.T) {
	if helperImportPath != "pricefollower.local/internal/amazon/amazontest" {
		t.Errorf("helperImportPath = %q", helperImportPath)
	}
}

// assertDependsOnHelper fails unless err is the failure that names the helper as a server dependency.
func assertDependsOnHelper(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("the check passed, want a failure: the server depends on the test helper")
	}
	for _, want := range []string{"pricefollower.local/internal/amazon/amazontest", dependsOnHelper} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("failure %q does not hold %q", err, want)
		}
	}
}

// assertListingFailure fails unless err is a listing failure holding want, not a dependency failure.
func assertListingFailure(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the check passed, want a listing failure holding %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("failure %q does not hold %q", err, want)
	}
	if strings.Contains(err.Error(), dependsOnHelper) {
		t.Errorf("failure %q reports a dependency instead of the listing error", err)
	}
}

func assertPassesCheck(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("the check failed, want a pass: %v", err)
	}
}

func TestCheckServerDependencies_TC002_JudgesListing(t *testing.T) {
	std := []string{"errors", "fmt", "internal/abi", "net/http", "os"}
	server := []string{"pricefollower.local/cmd/pricefollower", "pricefollower.local/internal/httpapi"}
	withHelper := append(append(append([]string{}, std...), server...), "pricefollower.local/internal/amazon/amazontest")
	withoutHelper := append(append([]string{}, std...), server...)

	cases := []struct {
		name    string
		paths   []string
		listErr error
		fails   string // "" passes, "dependency" or "listing"
	}{
		{name: "TC-002 a: listing holds the helper", paths: withHelper, fails: "dependency"},
		{name: "TC-002 b: listing without the helper", paths: withoutHelper},
		{name: "TC-002 c: parent package only", paths: []string{"pricefollower.local/internal/amazon"}},
		{name: "TC-002 d: sub-package of the helper", paths: append(append([]string{}, withoutHelper...), "pricefollower.local/internal/amazon/amazontest/sub")},
		{name: "TC-002 d: path extending the helper's name", paths: append(append([]string{}, withoutHelper...), "pricefollower.local/internal/amazon/amazontestutil")},
		{name: "TC-002 e: listing error and no paths", listErr: errors.New(toolError), fails: "listing"},
		{name: "TC-002 f: listing error and a partial list without the helper", paths: withoutHelper, listErr: errors.New(toolError), fails: "listing"},
		{name: "TC-002 f: listing error and a partial list with the helper", paths: withHelper, listErr: errors.New(toolError), fails: "listing"},
		{name: "TC-002 corner: helper listed by several programs", paths: append(append([]string{}, withHelper...), "pricefollower.local/cmd/tool", "pricefollower.local/internal/amazon/amazontest"), fails: "dependency"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkServerDependencies(strings.Join(tc.paths, "\n"), tc.listErr)
			switch tc.fails {
			case "":
				assertPassesCheck(t, err)
			case "dependency":
				assertDependsOnHelper(t, err)
			case "listing":
				assertListingFailure(t, err, toolError)
			}
		})
	}

	t.Run("TC-002 corner: whitespace and a trailing empty line do not hide the helper", func(t *testing.T) {
		listing := "fmt\n  pricefollower.local/cmd/pricefollower\n\tpricefollower.local/internal/amazon/amazontest  \r\nos\n\n"
		assertDependsOnHelper(t, checkServerDependencies(listing, nil))
	})

	t.Run("TC-002 corner: the helper repeated gives one failure", func(t *testing.T) {
		listing := strings.Repeat("pricefollower.local/internal/amazon/amazontest\n", 3)
		err := checkServerDependencies(listing, nil)
		assertDependsOnHelper(t, err)
		if err != nil {
			if n := strings.Count(err.Error(), dependsOnHelper); n != 1 {
				t.Errorf("failure %q reports the dependency %d times, want once", err, n)
			}
		}
	})

	t.Run("TC-002 corner: the same listing judged twice gives the same result", func(t *testing.T) {
		for _, listing := range []string{strings.Join(withHelper, "\n"), strings.Join(withoutHelper, "\n")} {
			first := checkServerDependencies(listing, nil)
			second := checkServerDependencies(listing, nil)
			if (first == nil) != (second == nil) || (first != nil && first.Error() != second.Error()) {
				t.Errorf("results differ: %v, then %v", first, second)
			}
		}
		assertDependsOnHelper(t, checkServerDependencies(strings.Join(withHelper, "\n"), nil))
		assertPassesCheck(t, checkServerDependencies(strings.Join(withoutHelper, "\n"), nil))
	})
}

// writeModule writes a fixture module pricefollower.local with a stub test helper and the given
// files (paths relative to the module root) under a new temporary folder, and returns that folder.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	all := map[string]string{
		"go.mod": "module pricefollower.local\n\ngo 1.25\n",
		"internal/amazon/amazontest/amazontest.go": "package amazontest\n\nconst DownloadMarker = \"\"\n",
	}
	for name, content := range files {
		all[name] = content
	}
	for name, content := range all {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const (
	plainMain   = "package main\n\nfunc main() {}\n"
	helperMain  = "package main\n\nimport \"pricefollower.local/internal/amazon/amazontest\"\n\nvar _ = amazontest.DownloadMarker\n\nfunc main() {}\n"
	serviceMain = "package main\n\nimport \"pricefollower.local/internal/service\"\n\nfunc main() { service.Run() }\n"
	plainSvc    = "package service\n\nfunc Run() {}\n"
	helperSvc   = "package service\n\nimport _ \"pricefollower.local/internal/amazon/amazontest\"\n\nfunc Run() {}\n"
	helperTest  = "import _ \"pricefollower.local/internal/amazon/amazontest\"\n"
)

// fixtureCase is a fixture module of TC-003 (issue #244).
type fixtureCase struct {
	name  string
	files map[string]string
	fails string // "" passes, "dependency", or the text a listing failure must hold
}

// fixtureCases is shared by the listing test and the build test, so both run the same fixtures.
var fixtureCases = []fixtureCase{
	{
		name:  "TC-003 a: a program imports the helper directly",
		files: map[string]string{"cmd/app/main.go": helperMain},
		fails: "dependency",
	},
	{
		name: "TC-003 b: a program imports the helper through another package",
		files: map[string]string{
			"cmd/app/main.go":             serviceMain,
			"internal/service/service.go": helperSvc,
		},
		fails: "dependency",
	},
	{
		name: "TC-003 c: only the second program depends on the helper",
		files: map[string]string{
			"cmd/app/main.go":  plainMain,
			"cmd/tool/main.go": helperMain,
		},
		fails: "dependency",
	},
	{
		name: "TC-003 d: only test files import the helper",
		files: map[string]string{
			"cmd/app/main.go":                  serviceMain,
			"cmd/app/main_test.go":             "package main\n\n" + helperTest,
			"internal/service/service.go":      plainSvc,
			"internal/service/service_test.go": "package service\n\n" + helperTest,
		},
	},
	{
		name: "TC-003 e: a test-support package no program imports uses the helper",
		files: map[string]string{
			"cmd/app/main.go":                        plainMain,
			"internal/httpapi/httpapitest/helper.go": "package httpapitest\n\n" + helperTest,
		},
	},
	{
		name:  "TC-003 f: a program imports a package that does not exist",
		files: map[string]string{"cmd/app/main.go": "package main\n\nimport _ \"pricefollower.local/internal/nope\"\n\nfunc main() {}\n"},
		fails: "pricefollower.local/internal/nope",
	},
	{
		name:  "TC-003 g: a program has a syntax error",
		files: map[string]string{"cmd/app/main.go": "package main\n\nimport \"fmt\n\nfunc main() { fmt.Println() }\n"},
		fails: "main.go",
	},
	{
		name:  "TC-003 corner: no helper import at all",
		files: map[string]string{"cmd/app/main.go": serviceMain, "internal/service/service.go": plainSvc},
	},
	{
		name:  "TC-003 corner: blank import of the helper in a program",
		files: map[string]string{"cmd/app/main.go": "package main\n\n" + helperTest + "\nfunc main() {}\n"},
		fails: "dependency",
	},
	{
		name: "TC-003 corner: renamed import of the helper in a server dependency",
		files: map[string]string{
			"cmd/app/main.go":             serviceMain,
			"internal/service/service.go": "package service\n\nimport helper \"pricefollower.local/internal/amazon/amazontest\"\n\nvar _ = helper.DownloadMarker\n\nfunc Run() {}\n",
		},
		fails: "dependency",
	},
}

func TestListServerDependencies_TC003_FixtureModules(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeModule(t, tc.files)
			listing, listErr := listServerDependencies(dir, fixtureEnv)
			err := checkServerDependencies(listing, listErr)
			switch tc.fails {
			case "":
				if listErr != nil {
					t.Fatalf("go list failed on the fixture: %v", listErr)
				}
				if !strings.Contains(listing, "pricefollower.local/cmd/app") {
					t.Errorf("listing does not hold pricefollower.local/cmd/app:\n%s", listing)
				}
				assertPassesCheck(t, err)
			case "dependency":
				if listErr != nil {
					t.Fatalf("go list failed on the fixture: %v", listErr)
				}
				assertDependsOnHelper(t, err)
			default:
				if listErr == nil {
					t.Fatalf("go list succeeded, want an error holding %q; listing:\n%s", tc.fails, listing)
				}
				for _, want := range []string{"exit status", tc.fails} {
					if !strings.Contains(listErr.Error(), want) {
						t.Errorf("listing error %q does not hold %q", listErr, want)
					}
				}
				assertListingFailure(t, err, tc.fails)
			}
		})
	}

	t.Run("TC-003 h: the go command cannot be started", func(t *testing.T) {
		dir := writeModule(t, map[string]string{"cmd/app/main.go": plainMain})
		t.Setenv("PATH", "")
		listing, listErr := listServerDependencies(dir, fixtureEnv)
		if listErr == nil {
			t.Fatalf("go list started without PATH; listing:\n%s", listing)
		}
		if !strings.Contains(listErr.Error(), "executable file not found") {
			t.Errorf("listing error %q does not hold the start error", listErr)
		}
		assertListingFailure(t, checkServerDependencies(listing, listErr), "executable file not found")
	})
}

// buildModule runs go build ./... offline in the fixture module dir; a failure holds the tool's output.
func buildModule(dir string) error {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), fixtureEnv...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build ./...: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Issue #251 TC-001: every valid fixture module builds offline; the invalid ones and a program using a
// name the stub does not declare do not, so the check is not vacuous.
func TestFixtureModulesBuild_Issue251_TC001(t *testing.T) {
	for _, tc := range fixtureCases {
		t.Run(tc.name, func(t *testing.T) {
			err := buildModule(writeModule(t, tc.files))
			switch tc.fails {
			case "", "dependency":
				if err != nil {
					t.Errorf("valid fixture does not build: %v", err)
				}
			default:
				if err == nil {
					t.Error("invalid fixture builds, want a build failure")
				}
			}
		})
	}

	t.Run("TC-003 h fixture: plainMain", func(t *testing.T) {
		if err := buildModule(writeModule(t, map[string]string{"cmd/app/main.go": plainMain})); err != nil {
			t.Errorf("valid fixture does not build: %v", err)
		}
	})

	t.Run("control: a program uses a name the stub does not declare", func(t *testing.T) {
		dir := writeModule(t, map[string]string{
			"internal/amazon/amazontest/amazontest.go": "package amazontest\n",
			"cmd/app/main.go":                          helperMain,
		})
		err := buildModule(dir)
		if err == nil {
			t.Fatal("the module builds, want a failure on the undeclared DownloadMarker")
		}
		if !strings.Contains(err.Error(), "DownloadMarker") {
			t.Errorf("build failure %q does not name DownloadMarker", err)
		}
	})
}

// TC-004: the check runs on the repository itself in every go test ./... run, and passes.
func TestServerDoesNotDependOnTestHelper(t *testing.T) {
	listing, listErr := listServerDependencies(filepath.Join("..", "..", ".."), nil)
	if err := checkServerDependencies(listing, listErr); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range strings.Split(listing, "\n") {
		if strings.TrimSpace(line) == "pricefollower.local/cmd/pricefollower" {
			found = true
		}
	}
	if !found {
		t.Errorf("the listing does not hold pricefollower.local/cmd/pricefollower, so the server was not listed:\n%s", listing)
	}
}
