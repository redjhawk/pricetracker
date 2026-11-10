package version

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	tag := func() (string, error) { return "v20261109200945", nil }
	noTag := func() (string, error) { return "", errors.New("no tags") }
	cases := []struct {
		name     string
		embedded string
		latest   func() (string, error)
		want     string
	}{
		{"release build", "v20261109200945", noTag, "v20261109200945"},
		{"development build", "", tag, "v20261109200945-dev"},
		{"no tag available", "", noTag, "dev"},
	}
	for _, c := range cases {
		if got := resolve(c.embedded, c.latest); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCurrentIsNeverEmpty(t *testing.T) {
	if Current() == "" {
		t.Fatal("empty version")
	}
}
