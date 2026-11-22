package service

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"pricefollower.local/internal/amazon"
)

// The installation documentation of issue #204 must name what the code reads to find and start the
// browser, and nothing else.

var (
	fencedCodePattern    = regexp.MustCompile("(?s)```.*?```")
	inlineCodePattern    = regexp.MustCompile("`([^`\n]+)`")
	browserNamePattern   = regexp.MustCompile(`^(chromium|google-chrome)[a-z-]*$`)
	browserVariableToken = regexp.MustCompile(`PRICEFOLLOWER_BROWSER_[A-Z_]+`)
	pathWordPattern      = regexp.MustCompile(`\bPATH\b`)
)

// serverSetupSection returns the "## Server setup" section of a Markdown document, or "".
func serverSetupSection(document string) string {
	start := strings.Index(document, "\n## Server setup\n")
	if start < 0 {
		return ""
	}
	section := document[start+1:]
	if end := strings.Index(section[len("## Server setup\n"):], "\n## "); end >= 0 {
		section = section[:len("## Server setup\n")+end]
	}
	return section
}

func readRepositoryFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name)) // #nosec G304 -- documentation file of the repository
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TC-011
func TestAmazonBrowserRetrieval_TC011_InstallationDocumentationStatesTheBrowserNeeds(t *testing.T) {
	assertInstallationDocumentation(t, readRepositoryFile(t, "deploy/README.md"), readRepositoryFile(t, "README.md"))
}

// specifiedServerSetup is a server setup section written as the technical specification describes it
// (TS-AMZ-BRET-006), followed by another section.
const specifiedServerSetup = "# Build and release\n\n## Publish\n\ntext\n\n## Server setup\n\n" +
	"The data directory (`PRICEFOLLOWER_DATA_DIR`) holds the SQLite database and the browser profile `amazon-browser/`.\n\n" +
	"### Browser for Amazon checks\n\nInstall Chromium or Google Chrome. On Debian:\n\n```bash\nsudo apt-get install -y chromium\n```\n\n" +
	"On Ubuntu and other amd64 systems: `wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb && sudo apt-get install -y ./google-chrome-stable_current_amd64.deb`.\n\n" +
	"The application uses `PRICEFOLLOWER_BROWSER_PATH` (full path to the executable), otherwise the first of `chromium`, `chromium-browser`, " +
	"`google-chrome-stable`, `google-chrome` found on the service's `PATH`.\n\n" +
	"In a container or for a service running as root, set `PRICEFOLLOWER_BROWSER_NO_SANDBOX=true`. In Docker, also give the container `--shm-size=1g`.\n\n" +
	"## Administrator account\n\nNot part of the section: `PRICEFOLLOWER_BROWSER_OTHER`, `chromium-dev`.\n"

// The TC-011 checks accept the section that the technical specification describes.
// TC-011
func TestAmazonBrowserRetrieval_TC011_ChecksAcceptTheSpecifiedSection(t *testing.T) {
	assertInstallationDocumentation(t, specifiedServerSetup, "- **Production deployment:** see [server setup](deploy/README.md#server-setup)\n")
}

// assertInstallationDocumentation checks the server setup section of deploy/README.md (deployment)
// and the link to it in README.md (readme).
func assertInstallationDocumentation(t *testing.T, deployment, readme string) {
	t.Helper()
	section := serverSetupSection(deployment)
	if section == "" {
		t.Fatal(`deploy/README.md has no "## Server setup" section`)
	}
	lower := strings.ToLower(section)

	t.Run("browser to install", func(t *testing.T) {
		if !strings.Contains(section, "Chromium") && !strings.Contains(section, "Google Chrome") {
			t.Fatal("the section does not name Chromium or Google Chrome")
		}
		if !strings.Contains(section, "apt-get install") {
			t.Fatal("the section gives no install command")
		}
	})

	t.Run("how the application finds the browser", func(t *testing.T) {
		if !strings.Contains(section, amazon.BrowserPathVariable) || !pathWordPattern.MatchString(section) {
			t.Fatalf("the section does not say the browser is %s or found on PATH", amazon.BrowserPathVariable)
		}
	})

	t.Run("container or root service", func(t *testing.T) {
		if !strings.Contains(section, browserNoSandboxVariable+"=true") || !strings.Contains(lower, "container") || !strings.Contains(lower, "root") {
			t.Fatalf("the section does not give %s=true for a container or a service running as root", browserNoSandboxVariable)
		}
	})

	t.Run("environment variables match the code", func(t *testing.T) {
		documented := map[string]bool{}
		for _, token := range browserVariableToken.FindAllString(section, -1) {
			documented[token] = true
		}
		read := []string{amazon.BrowserPathVariable, browserNoSandboxVariable}
		for _, name := range read {
			if !documented[name] {
				t.Errorf("the code reads %s, which the section does not document", name)
			}
		}
		for name := range documented {
			if !slices.Contains(read, name) {
				t.Errorf("the section documents %s, which the code does not read", name)
			}
		}
	})

	t.Run("executable names match the code", func(t *testing.T) {
		prose := fencedCodePattern.ReplaceAllString(section, "")
		documented := map[string]bool{}
		for _, match := range inlineCodePattern.FindAllStringSubmatch(prose, -1) {
			if browserNamePattern.MatchString(match[1]) {
				documented[match[1]] = true
			}
		}
		if len(amazon.BrowserNames) == 0 {
			t.Fatal("the code looks up no executable name")
		}
		for _, name := range amazon.BrowserNames {
			if !documented[name] {
				t.Errorf("the code looks up %q on PATH, which the section does not list in backticks", name)
			}
		}
		var extra []string
		for name := range documented {
			if !slices.Contains(amazon.BrowserNames, name) {
				extra = append(extra, name)
			}
		}
		sort.Strings(extra)
		if len(extra) > 0 {
			t.Errorf("the section lists executable names the code does not look up: %v", extra)
		}
	})

	t.Run("README links to the section", func(t *testing.T) {
		if !strings.Contains(readme, "deploy/README.md#server-setup") {
			t.Fatal("README.md does not link to deploy/README.md#server-setup")
		}
	})
}
