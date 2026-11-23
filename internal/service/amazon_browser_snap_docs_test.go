package service

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"pricefollower.local/internal/amazon"
)

// The installation documentation of issue #226 must tell the operator to set PRICEFOLLOWER_BROWSER_PATH
// to Google Chrome when a Chromium snap is installed, without changing the documented lookup order.

const snapExamplePath = "/usr/bin/google-chrome-stable"

// snapParagraph is the snap sentence of the technical specification (TS-AMZ-BRET-008).
const snapParagraph = "When a Chromium snap is installed (usual on Ubuntu, where `chromium` and `chromium-browser` lead to it), " +
	"snap confinement can stop it from starting for the service and every Amazon check then fails with \"browser could not start\", " +
	"so set `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable, for example `/usr/bin/google-chrome-stable`."

// lookupParagraphEnd ends the lookup paragraph of specifiedServerSetup.
const lookupParagraphEnd = "found on the service's `PATH`.\n\n"

// specifiedSnapSetup is specifiedServerSetup with the snap sentence as a new paragraph after the
// lookup paragraph, as the technical specification describes it.
var specifiedSnapSetup = strings.Replace(specifiedServerSetup, lookupParagraphEnd, lookupParagraphEnd+snapParagraph+"\n\n", 1)

var sentenceEndPattern = regexp.MustCompile(`[.!?](\s+|$)`)

// browserSubsection returns the "### Browser for Amazon checks" subsection of a section, or "".
func browserSubsection(section string) string {
	const heading = "\n### Browser for Amazon checks\n"
	start := strings.Index(section, heading)
	if start < 0 {
		return ""
	}
	subsection := section[start+1:]
	rest := subsection[len(heading)-1:]
	end := len(rest)
	for _, next := range []string{"\n### ", "\n## "} {
		if index := strings.Index(rest, next); index >= 0 && index < end {
			end = index
		}
	}
	return subsection[:len(heading)-1+end]
}

// sentence is a sentence of the prose and its offset in the prose.
type sentence struct {
	text   string
	offset int
}

// prose returns text without its fenced code blocks.
func prose(text string) string {
	return fencedCodePattern.ReplaceAllString(text, "")
}

// sentences splits prose (fenced code already removed) on sentence ends and blank lines.
func sentences(text string) []sentence {
	var result []sentence
	offset := 0
	for _, paragraph := range strings.Split(text, "\n\n") {
		start := 0
		for _, end := range sentenceEndPattern.FindAllStringIndex(paragraph, -1) {
			if part := strings.TrimSpace(paragraph[start:end[0]]); part != "" {
				result = append(result, sentence{text: part, offset: offset + start})
			}
			start = end[1]
		}
		if part := strings.TrimSpace(paragraph[start:]); part != "" {
			result = append(result, sentence{text: part, offset: offset + start})
		}
		offset += len(paragraph) + len("\n\n")
	}
	return result
}

// snapSentenceError checks that the browser subsection of deployment has the snap sentence after the
// lookup list.
func snapSentenceError(deployment string) error {
	subsection := browserSubsection(serverSetupSection(deployment))
	if subsection == "" {
		return errors.New(`deploy/README.md has no "### Browser for Amazon checks" subsection in "## Server setup"`)
	}
	text := prose(subsection)
	listEnd := strings.Index(text, "`google-chrome`")
	if listEnd < 0 {
		return errors.New("the subsection does not list `google-chrome`, which ends the looked-up names")
	}
	for _, candidate := range sentences(text) {
		lower := strings.ToLower(candidate.text)
		if strings.Contains(lower, "snap") && strings.Contains(lower, "ubuntu") && strings.Contains(lower, "google chrome") &&
			strings.Contains(candidate.text, amazon.BrowserPathVariable) && strings.Contains(candidate.text, snapExamplePath) {
			if candidate.offset <= listEnd {
				return fmt.Errorf("the snap sentence comes before the looked-up names: %q", candidate.text)
			}
			return nil
		}
	}
	return fmt.Errorf("the subsection has no sentence naming a Chromium snap, Ubuntu, %s, Google Chrome and %s", amazon.BrowserPathVariable, snapExamplePath)
}

func assertSnapSentence(t *testing.T, deployment string) {
	t.Helper()
	if err := snapSentenceError(deployment); err != nil {
		t.Fatal(err)
	}
}

// TC-001
func TestAmazonBrowserSnapDocs_TC001_InstallationDocumentationHasTheSnapSentence(t *testing.T) {
	assertSnapSentence(t, readRepositoryFile(t, "deploy/README.md"))
}

// TC-001
func TestAmazonBrowserSnapDocs_TC001_ChecksAcceptTheSpecifiedSection(t *testing.T) {
	if specifiedSnapSetup == specifiedServerSetup {
		t.Fatal("the snap sentence was not inserted in the fixture")
	}
	assertSnapSentence(t, specifiedSnapSetup)
}

// TC-001
func TestAmazonBrowserSnapDocs_TC001_ChecksRejectAMissingOrMisplacedSentence(t *testing.T) {
	withParagraph := func(paragraph string) string {
		return strings.Replace(specifiedServerSetup, lookupParagraphEnd, lookupParagraphEnd+paragraph+"\n\n", 1)
	}
	fixtures := map[string]string{
		"no sentence": specifiedServerSetup,
		"no setting":  withParagraph(strings.Replace(snapParagraph, "`PRICEFOLLOWER_BROWSER_PATH`", "the browser path", 1)),
		"no path":     withParagraph(strings.Replace(snapParagraph, ", for example `/usr/bin/google-chrome-stable`", "", 1)),
		"words split over two sentences": withParagraph("When a Chromium snap is installed on Ubuntu, it can fail to start for the service. " +
			"Set `PRICEFOLLOWER_BROWSER_PATH` to the Google Chrome executable, for example `/usr/bin/google-chrome-stable`."),
		"sentence only in another section": strings.Replace(specifiedServerSetup, "## Administrator account\n\n",
			"## Administrator account\n\n"+snapParagraph+"\n\n", 1),
		"sentence only in a fenced block": withParagraph("```text\n" + snapParagraph + "\n```"),
		"sentence before the looked-up names": strings.Replace(specifiedServerSetup, "The application uses `PRICEFOLLOWER_BROWSER_PATH`",
			snapParagraph+"\n\nThe application uses `PRICEFOLLOWER_BROWSER_PATH`", 1),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if name != "no sentence" && fixture == specifiedServerSetup {
				t.Fatal("the fixture was not changed")
			}
			if err := snapSentenceError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}

// lookedUpNames returns the browser names in backticks of the prose of subsection, in reading order,
// without repeats.
func lookedUpNames(subsection string) []string {
	var names []string
	for _, match := range inlineCodePattern.FindAllStringSubmatch(prose(subsection), -1) {
		if browserNamePattern.MatchString(match[1]) && !slices.Contains(names, match[1]) {
			names = append(names, match[1])
		}
	}
	return names
}

// lookedUpNamesError checks that the browser subsection of deployment lists the names the code looks
// up, in the same order, and no other.
func lookedUpNamesError(deployment string) error {
	subsection := browserSubsection(serverSetupSection(deployment))
	if subsection == "" {
		return errors.New(`deploy/README.md has no "### Browser for Amazon checks" subsection in "## Server setup"`)
	}
	if names := lookedUpNames(subsection); !slices.Equal(names, amazon.BrowserNames) {
		return fmt.Errorf("the subsection lists the executable names %v, the code looks up %v", names, amazon.BrowserNames)
	}
	return nil
}

// TC-002
func TestAmazonBrowserSnapDocs_TC002_DocumentedNamesAreThoseOfTheCodeInOrder(t *testing.T) {
	if want := []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome"}; !slices.Equal(amazon.BrowserNames, want) {
		t.Fatalf("amazon.BrowserNames is %v, want %v", amazon.BrowserNames, want)
	}
	deployment := readRepositoryFile(t, "deploy/README.md")
	if err := lookedUpNamesError(deployment); err != nil {
		t.Fatal(err)
	}
	assertInstallationDocumentation(t, deployment, readRepositoryFile(t, "README.md"))
}

// TC-002
func TestAmazonBrowserSnapDocs_TC002_ChecksAcceptTheSpecifiedSection(t *testing.T) {
	if !strings.Contains(prose(browserSubsection(serverSetupSection(specifiedSnapSetup))), "`"+snapExamplePath+"`") {
		t.Fatal("the fixture has no backticked example path")
	}
	if err := lookedUpNamesError(specifiedSnapSetup); err != nil {
		t.Fatal(err)
	}
	assertInstallationDocumentation(t, specifiedSnapSetup, "- **Production deployment:** see [server setup](deploy/README.md#server-setup)\n")
}

// TC-002
func TestAmazonBrowserSnapDocs_TC002_ChecksRejectAnotherOrderOrAnExtraName(t *testing.T) {
	fixtures := map[string]string{
		"names in another order": strings.Replace(specifiedSnapSetup, "the first of `chromium`, `chromium-browser`,",
			"the first of `chromium-browser`, `chromium`,", 1),
		"extra name google-chrome-unstable": strings.Replace(specifiedSnapSetup, "for example `/usr/bin/google-chrome-stable`",
			"for example `/usr/bin/google-chrome-stable` or `google-chrome-unstable`", 1),
		"extra name chromium-snap": strings.Replace(specifiedSnapSetup, "When a Chromium snap is installed",
			"When a Chromium snap (`chromium-snap`) is installed", 1),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if fixture == specifiedSnapSetup {
				t.Fatal("the fixture was not changed")
			}
			if err := lookedUpNamesError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}

// TC-002
func TestAmazonBrowserSnapDocs_TC002_RepeatedNameAfterTheListIsAccepted(t *testing.T) {
	fixture := strings.Replace(specifiedSnapSetup, "the Google Chrome executable,", "the Google Chrome executable instead of `chromium`,", 1)
	if fixture == specifiedSnapSetup {
		t.Fatal("the fixture was not changed")
	}
	if err := lookedUpNamesError(fixture); err != nil {
		t.Fatal(err)
	}
}
