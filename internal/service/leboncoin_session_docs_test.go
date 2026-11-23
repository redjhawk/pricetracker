package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// Issue #265 (FR-SECRET-012, FR-LBC-CAP-013): the operator guide describes the masked hint,
// keeping the saved session with an empty input, replacing it, and removing it with
// "Remove session" then "Save".

const sessionGuidePath = "doc/leboncoin-session.md"

// specifiedSessionGuide is the Settings part of the guide as the technical specification
// describes it (TS-SECRET-016).
const specifiedSessionGuide = "# LeBoncoin session from Settings\n\n" +
	"Paste it into **Settings › LeBonCoin session** in the PriceFollower interface. No file, server shell access, environment variable or restart is needed.\n\n" +
	"## Paste it into Settings\n\n" +
	"1. Copy the whole `datadome=…` line.\n" +
	"2. Paste the line into **LeBonCoin session** and save. Pasting a new value replaces the saved session.\n\n" +
	"Settings never shows the saved session again: it shows a masked hint with only the last 4 characters, and the input is always empty. " +
	"Leaving the input empty keeps the saved session when you save.\n\n" +
	"## Check the result\n\n" +
	"Refresh a LeBoncoin item, then reopen **Settings**.\n\n" +
	"## Renew or clear\n\n" +
	"- **Renew** by running the helper again and pasting the new line into Settings.\n" +
	"- **Clear** by selecting **Remove session**, then **Save**.\n\n" +
	"## Keep the value secret\n\n" +
	"Do not paste it into chats, tickets, URLs or shared logs. Once saved, PriceFollower never shows or returns it in full, not even in Settings.\n"

var (
	emptyFieldClearPattern = regexp.MustCompile(`(?i)sav(e|ing) an empty`)
	readInSettingsPattern  = regexp.MustCompile(`(?i)can read (it|the value|the session) in settings`)
	oldWorkflowPattern     = regexp.MustCompile("(?i)\\bscp\\b|drop-in|systemctl|Environment=|LEBONCOIN_SESSION_FILE|session\\.json")
)

// guideSentence reports whether one sentence of the guide's prose contains every word.
func guideSentence(text string, words ...string) bool {
	for _, candidate := range sentences(text) {
		lower := strings.ToLower(candidate.text)
		found := true
		for _, word := range words {
			if !strings.Contains(lower, strings.ToLower(word)) {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

// sessionGuideError checks the Settings instructions of the operator guide.
func sessionGuideError(guide string) error {
	text := prose(guide)
	for _, heading := range []string{"## Paste it into Settings", "## Check the result", "## Renew or clear", "## Keep the value secret"} {
		if !strings.Contains(text, "\n"+heading+"\n") {
			return fmt.Errorf("the guide has no %q section", heading)
		}
	}
	switch {
	case !strings.Contains(text, "Settings › LeBonCoin session"):
		return errors.New("the guide does not describe pasting the session into Settings › LeBonCoin session")
	case !guideSentence(text, "masked hint", "last 4 characters"):
		return errors.New("no sentence says that Settings shows a masked hint with the last 4 characters")
	case !guideSentence(text, "empty", "keeps the saved session"):
		return errors.New("no sentence says that leaving the input empty keeps the saved session")
	case !guideSentence(text, "new value", "replaces"):
		return errors.New("no sentence says that pasting a new value replaces the saved session")
	case !guideSentence(text, "clear", "**Remove session**", "**Save**"):
		return errors.New("no sentence says to clear the session with **Remove session** then **Save**")
	case !guideSentence(text, "do not paste it into chats"):
		return errors.New("the secret-handling warning is missing")
	}
	if match := emptyFieldClearPattern.FindString(text); match != "" {
		return fmt.Errorf("the guide still says to clear by %q field", match)
	}
	if match := readInSettingsPattern.FindString(text); match != "" {
		return fmt.Errorf("the guide still says that anyone %q", match)
	}
	if match := oldWorkflowPattern.FindString(guide); match != "" {
		return fmt.Errorf("the guide has file-based workflow instructions (%q)", match)
	}
	return nil
}

// TC-019
func TestLeboncoinSessionDocs_TC019_GuideDescribesHintsKeepingAndRemoving(t *testing.T) {
	if err := sessionGuideError(readRepositoryFile(t, sessionGuidePath)); err != nil {
		t.Fatal(err)
	}
}

// TC-019
func TestLeboncoinSessionDocs_TC019_ChecksAcceptTheSpecifiedGuide(t *testing.T) {
	if err := sessionGuideError(specifiedSessionGuide); err != nil {
		t.Fatal(err)
	}
}

// TC-019
func TestLeboncoinSessionDocs_TC019_ChecksRejectOutdatedGuides(t *testing.T) {
	const removeSentence = "- **Clear** by selecting **Remove session**, then **Save**.\n"
	fixtures := map[string]string{
		"clears by saving an empty field": strings.Replace(specifiedSessionGuide, removeSentence,
			removeSentence+"- You can also clear it by saving an empty **LeBonCoin session** field.\n", 1),
		"no Remove session": strings.Replace(specifiedSessionGuide, removeSentence, "- **Clear** it in Settings.\n", 1),
		"remove instruction only in a fenced block": strings.Replace(specifiedSessionGuide, removeSentence,
			"```text\nClear by selecting **Remove session**, then **Save**.\n```\n", 1),
		"no masked hint":    strings.Replace(specifiedSessionGuide, "it shows a masked hint with only the last 4 characters, and ", "", 1),
		"no keeping rule":   strings.Replace(specifiedSessionGuide, "Leaving the input empty keeps the saved session when you save.", "", 1),
		"no replacing rule": strings.Replace(specifiedSessionGuide, " Pasting a new value replaces the saved session.", "", 1),
		"value readable in Settings": strings.Replace(specifiedSessionGuide, "Once saved, PriceFollower never shows or returns it in full, not even in Settings.",
			"Anyone who can open the PriceFollower interface can read it in Settings.", 1),
		"no secret warning": strings.Replace(specifiedSessionGuide, "Do not paste it into chats, tickets, URLs or shared logs. ", "", 1),
		"no check section":  strings.Replace(specifiedSessionGuide, "## Check the result\n", "## Results\n", 1),
		"scp instructions":  specifiedSessionGuide + "\nCopy the file with `scp session.json pi:` to the server.\n",
		"systemd drop-in":   specifiedSessionGuide + "\nAdd a systemd drop-in with `Environment=LEBONCOIN_SESSION_FILE=/var/lib/pricefollower/leboncoin/session.json`.\n",
	}
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if fixture == specifiedSessionGuide {
				t.Fatal("the fixture was not changed")
			}
			if err := sessionGuideError(fixture); err == nil {
				t.Fatal("the check accepted the fixture")
			}
		})
	}
}
