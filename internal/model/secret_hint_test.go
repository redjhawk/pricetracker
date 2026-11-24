package model

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TC-011: the hint shows the last 4 characters of a secret of at least
// SecretHintMinLength characters, and nothing for a shorter or missing secret.
func TestSecretHint_TC011_LastFourCharactersFromTheThreshold(t *testing.T) {
	if SecretHintMinLength != 16 {
		t.Fatalf("SecretHintMinLength is %d, want 16", SecretHintMinLength)
	}
	text := func(value string) *string { return &value }
	for _, test := range []struct {
		name  string
		value *string
		want  *string
	}{
		{"saved token T", text("sk-ant-oat01-synthetic-token-wxyz"), text("wxyz")},
		{"saved session S", text("synthetic-session-value-9f3a"), text("9f3a")},
		{"exactly at the threshold", text("abcdefghijkl1234"), text("1234")},
		{"one below the threshold", text("abcdefghijk1234"), nil},
		{"one character", text("x"), nil},
		{"empty", text(""), nil},
		{"not saved", nil, nil},
		{"threshold counted in characters", text(strings.Repeat("é", 12) + "wxyz"), text("wxyz")},
		{"multi-byte value below the threshold", text(strings.Repeat("é", 11) + "wxyz"), nil},
		{"multi-byte last characters", text("abcdefghijkl" + "éèàü"), text("éèàü")},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := SecretHint(test.value)
			switch {
			case test.want == nil && got != nil:
				t.Fatalf("SecretHint = %q, want nil", *got)
			case test.want != nil && got == nil:
				t.Fatalf("SecretHint = nil, want %q", *test.want)
			case test.want != nil && *got != *test.want:
				t.Fatalf("SecretHint = %q, want %q", *got, *test.want)
			}
			if got != nil && utf8.RuneCountInString(*got) > 4 {
				t.Fatalf("SecretHint returned more than 4 characters: %q", *got)
			}
		})
	}
}

// TC-011: the hint is a new string, so changing it never changes the secret.
func TestSecretHint_TC011_DoesNotAliasTheSecret(t *testing.T) {
	secret := "sk-ant-oat01-synthetic-token-wxyz"
	hint := SecretHint(&secret)
	if hint == nil || hint == &secret {
		t.Fatalf("SecretHint(%q) = %v, want a new string", secret, hint)
	}
	*hint = "zzzz"
	if secret != "sk-ant-oat01-synthetic-token-wxyz" {
		t.Fatalf("changing the hint changed the secret to %q", secret)
	}
}
