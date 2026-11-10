// Package version reports the running application version.
package version

import (
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Version is the release tag embedded at build time with
// -ldflags "-X pricefollower.local/internal/version.Version=<tag>".
var Version string

var (
	once    sync.Once
	current string
)

// Current returns the embedded release tag or, for development builds,
// the latest git tag followed by "-dev", or "dev" when no tag is available.
func Current() string {
	once.Do(func() { current = resolve(Version, latestTag) })
	return current
}

func resolve(embedded string, latestTag func() (string, error)) string {
	if embedded != "" {
		return embedded
	}
	tag, err := latestTag()
	if err != nil || tag == "" {
		return "dev"
	}
	return tag + "-dev"
}

func latestTag() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "git", "describe", "--tags", "--abbrev=0").Output()
	return strings.TrimSpace(string(output)), err
}
