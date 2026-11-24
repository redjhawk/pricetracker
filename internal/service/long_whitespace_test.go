package service

import (
	"errors"
	"strings"
	"testing"
)

// Issue #286: a whitespace-only LeBoncoin session input of any length leaves the
// session untouched; the 8192-character limit applies only to other input.

// longBlank returns n characters mixing spaces, tabs, "\n" and "\r\n".
func longBlank(n int) string {
	var builder strings.Builder
	for builder.Len() < n {
		builder.WriteString(" \t\n\r\n")
	}
	return builder.String()[:n]
}

func TestLongWhitespace_TC001_BlankSessionInputOfAnyLengthIsNoChange(t *testing.T) {
	for _, test := range []struct{ name, value string }{
		{"empty", ""},
		{"three spaces", "   "},
		{"8192 spaces", strings.Repeat(" ", 8192)},
		{"8193 spaces", strings.Repeat(" ", 8193)},
		{"9000 spaces", strings.Repeat(" ", 9000)},
		{"9000 mixed blanks", longBlank(9000)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if strings.TrimSpace(test.value) != "" {
				t.Fatalf("fixture %q is not blank", test.name)
			}
			change, err := sessionChangeFor(&SessionInput{Value: test.value, Revision: 3})
			if err != nil || change != nil {
				t.Fatalf("got change %+v, error %v; want no session change and no error", change, err)
			}
		})
	}
}

func TestLongWhitespace_TC001_LengthRuleStillAppliesToNonBlankInput(t *testing.T) {
	const tooLong = "The pasted text is too long (maximum 8192 characters)."
	w9000 := strings.Repeat(" ", 9000)
	for _, test := range []struct{ name, value string }{
		{"8193 characters", strings.Repeat("v", 8193)},
		{"blanks then abc", w9000 + "abc"},
		{"abc then blanks", "abc" + w9000},
	} {
		t.Run(test.name, func(t *testing.T) {
			change, err := sessionChangeFor(&SessionInput{Value: test.value, Revision: 3})
			var serviceError *Error
			if change != nil || !errors.As(err, &serviceError) || serviceError.Status != 400 || serviceError.Code != "INVALID_SESSION" || serviceError.Message != tooLong {
				t.Fatalf("got change %+v, error %v; want 400 INVALID_SESSION %q", change, err, tooLong)
			}
		})
	}
}

func TestLongWhitespace_TC001_PaddedValueAndRemovalAreUnchanged(t *testing.T) {
	padding := strings.Repeat(" ", 100)
	change, err := sessionChangeFor(&SessionInput{Value: padding + "datadome=synthetic-renewed-or-new-n0p1" + padding, Revision: 3})
	if err != nil || change == nil || change.Value == nil || *change.Value != "synthetic-renewed-or-new-n0p1" || change.ExpectedRevision != 3 {
		t.Fatalf("padded value: got change %+v, error %v; want a replacement with N at revision 3", change, err)
	}

	change, err = sessionChangeFor(&SessionInput{Remove: true, Revision: 3})
	if err != nil || change == nil || change.Value != nil || change.ExpectedRevision != 3 {
		t.Fatalf("removal: got change %+v, error %v; want a removal at revision 3", change, err)
	}
}
