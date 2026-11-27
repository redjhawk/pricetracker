package service

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The installation documentation of issue #210 must show the Amazon failure log line with its
// response field, say what the field holds, and leave the rest of the log list as it is.

const (
	failedLinePrefix  = "Amazon request failed"
	noPriceLinePrefix = "Amazon check found no price"
	formerFailedLine  = "Amazon request failed url=… at=… status=… cause=…"
	failedLine        = "Amazon request failed url=… at=… status=… cause=… response=…"
	noPriceLine       = "Amazon check found no price url=… at=… status=… cause=…"
	startLine         = "Amazon browser not found; Amazon checks will fail until Chromium is installed or …"

	startItemLabel   = "At start, when no browser is found:"
	failedItemLabel  = "For each failed Amazon check:"
	noPriceItemLabel = "For a page without a detectable price:"

	causesSentence = "The cause says whether the browser could not start (with the browser's own reason), the page could not be opened " +
		"or did not finish loading, Amazon returned a non-success HTTP status, or Amazon showed a robot check."
	responseSentence = "`response` is the start of the page Amazon returned; it can be empty, for example when the browser could not start."
	notCountedText   = "It is not counted as a failed request."
)

// specifiedLogList is the log list of the browser subsection as the technical specification
// describes it (TS-AMZ-BRET-009), followed by a blank line.
const specifiedLogList = "What the server log shows:\n\n" +
	"- " + startItemLabel + " `" + startLine + "`.\n" +
	"- " + failedItemLabel + " `" + failedLine + "`. " + causesSentence + " " + responseSentence + "\n" +
	"- " + noPriceItemLabel + " `" + noPriceLine + "`. " + notCountedText + "\n\n"

const administratorHeading = "## Administrator account\n"

// specifiedFailureLogSetup is specifiedSnapSetup with the log list at the end of the browser subsection.
var specifiedFailureLogSetup = strings.Replace(specifiedSnapSetup, administratorHeading, specifiedLogList+administratorHeading, 1)

var fieldNamePattern = regexp.MustCompile(`(?:^| )([a-z]+)=`)

// logListItems returns the items of the "What the server log shows:" list of the browser subsection
// of deployment, without their "- ".
func logListItems(deployment string) ([]string, error) {
	subsection := browserSubsection(serverSetupSection(deployment))
	if subsection == "" {
		return nil, errors.New(`deploy/README.md has no "### Browser for Amazon checks" subsection in "## Server setup"`)
	}
	lines := strings.Split(prose(subsection), "\n")
	start := slices.Index(lines, "What the server log shows:")
	if start < 0 {
		return nil, errors.New(`the browser subsection has no "What the server log shows:" list`)
	}
	index := start + 1
	for index < len(lines) && strings.TrimSpace(lines[index]) == "" {
		index++
	}
	var items []string
	for ; index < len(lines) && strings.HasPrefix(lines[index], "- "); index++ {
		items = append(items, strings.TrimPrefix(lines[index], "- "))
	}
	if len(items) == 0 {
		return nil, errors.New(`the "What the server log shows:" list has no item`)
	}
	return items, nil
}

// logItem returns the item of items that starts with label, or "".
func logItem(items []string, label string) string {
	for _, item := range items {
		if strings.HasPrefix(item, label) {
			return item
		}
	}
	return ""
}

// documentedLine returns the only inline code token of the browser subsection prose of deployment that
// starts with prefix, the log list item holding it, and the offset in the item where the token ends.
func documentedLine(deployment, prefix string) (line, item string, end int, err error) {
	subsection := browserSubsection(serverSetupSection(deployment))
	if subsection == "" {
		return "", "", 0, errors.New(`deploy/README.md has no "### Browser for Amazon checks" subsection in "## Server setup"`)
	}
	var tokens []string
	for _, match := range inlineCodePattern.FindAllStringSubmatch(prose(subsection), -1) {
		if strings.HasPrefix(match[1], prefix) {
			tokens = append(tokens, match[1])
		}
	}
	if len(tokens) != 1 {
		return "", "", 0, fmt.Errorf("the browser subsection has %d inline code lines starting with %q, want 1: %q", len(tokens), prefix, tokens)
	}
	items, err := logListItems(deployment)
	if err != nil {
		return "", "", 0, err
	}
	quoted := "`" + tokens[0] + "`"
	for _, candidate := range items {
		if index := strings.Index(candidate, quoted); index >= 0 {
			return tokens[0], candidate, index + len(quoted), nil
		}
	}
	return "", "", 0, fmt.Errorf("the line %q is not in an item of the log list", tokens[0])
}

// fieldNames returns the name= field names of a log line, in reading order.
func fieldNames(line string) []string {
	var names []string
	for _, match := range fieldNamePattern.FindAllStringSubmatch(line, -1) {
		names = append(names, match[1])
	}
	return names
}

// documentedPattern returns the pattern of a documented log line: its text quoted, each "…" a lazy
// capture except the last, which captures to the end of the line.
func documentedPattern(line string) *regexp.Regexp {
	parts := strings.Split(line, "…")
	var pattern strings.Builder
	pattern.WriteString("^")
	for index, part := range parts {
		pattern.WriteString(regexp.QuoteMeta(part))
		switch {
		case index == len(parts)-2:
			pattern.WriteString("(.*)")
		case index < len(parts)-2:
			pattern.WriteString("(.*?)")
		}
	}
	pattern.WriteString("$")
	return regexp.MustCompile(pattern.String())
}

// failedLineError checks that the browser subsection of deployment documents the failure line with
// the response field after the cause, in the failed-check item.
func failedLineError(deployment string) error {
	line, item, _, err := documentedLine(deployment, failedLinePrefix)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(item, failedItemLabel) {
		return fmt.Errorf("the failure line is not in the %q item: %q", failedItemLabel, item)
	}
	if line != failedLine {
		return fmt.Errorf("the documented failure line is %q, want %q", line, failedLine)
	}
	if names, want := fieldNames(line), []string{"url", "at", "status", "cause", "response"}; !slices.Equal(names, want) {
		return fmt.Errorf("the documented failure line has the fields %v, want %v", names, want)
	}
	return nil
}

// responseSentenceError checks that the failed-check item of deployment has, after the failure line,
// one sentence saying that response is the start of the page Amazon returned and that it can be empty.
func responseSentenceError(deployment string) error {
	_, item, end, err := documentedLine(deployment, failedLinePrefix)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(item, failedItemLabel) {
		return fmt.Errorf("the failure line is not in the %q item: %q", failedItemLabel, item)
	}
	for _, candidate := range sentences(item) {
		if candidate.offset < end {
			continue
		}
		lower := strings.ToLower(candidate.text)
		if strings.Contains(lower, "response") && strings.Contains(lower, "start") && strings.Contains(lower, "page") && strings.Contains(lower, "empty") {
			return nil
		}
	}
	return fmt.Errorf("the failed-check item has no sentence after the failure line saying that response is the start of the page and can be empty: %q", item)
}

// logListError checks that the log list of deployment keeps its start item, the causes sentence and
// the no-price item without a response field, and has exactly three items.
func logListError(deployment string) error {
	items, err := logListItems(deployment)
	if err != nil {
		return err
	}
	if len(items) != 3 {
		return fmt.Errorf("the log list has %d items, want 3: %q", len(items), items)
	}
	start := logItem(items, startItemLabel)
	if match := inlineCodePattern.FindStringSubmatch(start); match == nil || match[1] != startLine {
		return fmt.Errorf("the start item is %q, want the line %q", start, startLine)
	}
	if failed := logItem(items, failedItemLabel); !strings.Contains(failed, causesSentence) {
		return fmt.Errorf("the failed-check item lost the causes sentence: %q", failed)
	}
	line, item, end, err := documentedLine(deployment, noPriceLinePrefix)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(item, noPriceItemLabel) {
		return fmt.Errorf("the no-price line is not in the %q item: %q", noPriceItemLabel, item)
	}
	if line != noPriceLine {
		return fmt.Errorf("the documented no-price line is %q, want %q", line, noPriceLine)
	}
	if !strings.Contains(item[end:], notCountedText) {
		return fmt.Errorf("the no-price item does not say %q after the line: %q", notCountedText, item)
	}
	return nil
}

// TC-001
func TestAmazonFailureLogDocs_TC001_DocumentedFailureLineEndsWithResponse(t *testing.T) {
	if err := failedLineError(readRepositoryFile(t, "deploy/README.md")); err != nil {
		t.Fatal(err)
	}
}

// TC-001
func TestAmazonFailureLogDocs_TC001_ChecksAcceptTheSpecifiedList(t *testing.T) {
	if specifiedFailureLogSetup == specifiedSnapSetup {
		t.Fatal("the log list was not inserted in the fixture")
	}
	if err := failedLineError(specifiedFailureLogSetup); err != nil {
		t.Fatal(err)
	}
}

// TC-001
func TestAmazonFailureLogDocs_TC001_ChecksRejectAWrongOrMisplacedLine(t *testing.T) {
	fixtures := map[string]string{
		"former line": strings.Replace(specifiedFailureLogSetup, "`"+failedLine+"`", "`"+formerFailedLine+"`", 1),
		"response before cause": strings.Replace(specifiedFailureLogSetup, "`"+failedLine+"`",
			"`Amazon request failed url=… at=… status=… response=… cause=…`", 1),
		"line given twice": strings.Replace(specifiedFailureLogSetup, "`"+failedLine+"`.",
			"`"+failedLine+"` (formerly `"+formerFailedLine+"`).", 1),
		"list only in another section": strings.Replace(specifiedSnapSetup, administratorHeading+"\n",
			administratorHeading+"\n"+specifiedLogList, 1),
		"list only in a fenced block": strings.Replace(specifiedSnapSetup, administratorHeading,
			"```text\n"+specifiedLogList+"```\n\n"+administratorHeading, 1),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if fixture == specifiedFailureLogSetup || fixture == specifiedSnapSetup {
				t.Fatal("the fixture was not changed")
			}
			if err := failedLineError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}

// TC-002
func TestAmazonFailureLogDocs_TC002_FailedItemSaysWhatResponseHolds(t *testing.T) {
	if err := responseSentenceError(readRepositoryFile(t, "deploy/README.md")); err != nil {
		t.Fatal(err)
	}
}

// TC-002
func TestAmazonFailureLogDocs_TC002_ChecksAcceptTheSpecifiedList(t *testing.T) {
	if err := responseSentenceError(specifiedFailureLogSetup); err != nil {
		t.Fatal(err)
	}
}

// TC-002
func TestAmazonFailureLogDocs_TC002_ChecksRejectAMissingOrIncompleteSentence(t *testing.T) {
	withSentence := func(sentence string) string {
		return strings.Replace(specifiedFailureLogSetup, " "+responseSentence, sentence, 1)
	}
	withoutSentence := withSentence("")
	fixtures := map[string]string{
		"no sentence":       withoutSentence,
		"no empty":          withSentence(" `response` is the start of the page Amazon returned."),
		"no start of page":  withSentence(" `response` is what Amazon returned; it can be empty."),
		"words split apart": withSentence(" `response` is the start of the page Amazon returned. It can be empty, for example when the browser could not start."),
		"sentence only in the no-price item": strings.Replace(withoutSentence, notCountedText,
			notCountedText+" "+responseSentence, 1),
		"sentence only in another section": strings.Replace(withoutSentence, administratorHeading+"\n",
			administratorHeading+"\n"+responseSentence+"\n\n", 1),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if fixture == specifiedFailureLogSetup {
				t.Fatal("the fixture was not changed")
			}
			if err := failedLineError(fixture); err != nil {
				t.Fatalf("the fixture changed the failure line: %v", err)
			}
			if err := responseSentenceError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}

// TC-003; the #204 TC-011 and #226 TC-001 – TC-002 tests check the real file.
func TestAmazonFailureLogDocs_TC003_RestOfTheLogListIsUnchanged(t *testing.T) {
	if err := logListError(readRepositoryFile(t, "deploy/README.md")); err != nil {
		t.Fatal(err)
	}
}

// TC-003
func TestAmazonFailureLogDocs_TC003_ChecksAcceptTheSpecifiedList(t *testing.T) {
	if err := logListError(specifiedFailureLogSetup); err != nil {
		t.Fatal(err)
	}
	if err := snapSentenceError(specifiedFailureLogSetup); err != nil {
		t.Fatal(err)
	}
	if err := lookedUpNamesError(specifiedFailureLogSetup); err != nil {
		t.Fatal(err)
	}
	assertInstallationDocumentation(t, specifiedFailureLogSetup, "- **Production deployment:** see [server setup](deploy/README.md#server-setup)\n")
}

// TC-003
func TestAmazonFailureLogDocs_TC003_ChecksRejectAChangedList(t *testing.T) {
	noPriceItem := "- " + noPriceItemLabel + " `" + noPriceLine + "`. " + notCountedText + "\n"
	fixtures := map[string]string{
		"response added to the no-price line": strings.Replace(specifiedFailureLogSetup, "`"+noPriceLine+"`",
			"`"+noPriceLine+" response=…`", 1),
		"causes sentence removed": strings.Replace(specifiedFailureLogSetup, " "+causesSentence, "", 1),
		"no-price item removed":   strings.Replace(specifiedFailureLogSetup, noPriceItem, "", 1),
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if fixture == specifiedFailureLogSetup {
				t.Fatal("the fixture was not changed")
			}
			if err := logListError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}
