package context

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// The semantic version of the application.
// This can be overriden by vcsVersion.
const version = "0.0.0"

var (
	vcsVersion string // version from VCS set at build time
	// Simplified semver regex. A more complete one can be found on https://semver.org/
	versionRx = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*).*`)
	shaRx     = regexp.MustCompile(`^g[0-9a-f]{6,}`)
)

// VersionInfo stores app version information.
type VersionInfo struct {
	Commit      string
	Semantic    string
	TagDistance int // number of commits since latest tag
	Dirty       bool
	goInfo      string
}

// GetVersion returns the app version including VCS and Go runtime information.
// It first reads the VCS version set at build time, and falls back to the VCS
// information provided by the Go runtime.
// The information in the Go runtime is not as extensive as the one extracted
// from Git, and we use the Go runtime information in case the binary is built
// without setting -ldflags (e.g. Homebrew), so we still need both.
// See https://github.com/golang/go/issues/50603
func GetVersion() (*VersionInfo, error) {
	vi := &VersionInfo{}
	vi.goInfo = fmt.Sprintf("%s, %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if vcsVersion != "" {
		if err := vi.UnmarshalText([]byte(vcsVersion)); err != nil {
			return nil, fmt.Errorf("failed reading VCS version '%s': %w", vcsVersion, err)
		}
	}

	vi.extractVersionFromRuntime()

	if vi.Semantic == "" {
		vi.Semantic = version
	}

	return vi, nil
}

// String returns the full version information.
func (vi *VersionInfo) String() string {
	if vi.Commit == "" {
		return fmt.Sprintf("v%s (%s)", vi.Semantic, vi.goInfo)
	}

	var distance string
	if vi.TagDistance > 0 {
		distance = fmt.Sprintf("-%d", vi.TagDistance)
	}

	var dirty string
	if vi.Dirty {
		dirty = "-dirty"
	}

	return fmt.Sprintf("v%s (commit/%s%s%s, %s)",
		vi.Semantic, vi.Commit, distance, dirty, vi.goInfo)
}

// UnmarshalText parses the output of `git describe`.
func (vi *VersionInfo) UnmarshalText(data []byte) error {
	parts := strings.Split(string(data), "-")
	verParts := []string{}

	for _, part := range parts {
		if shaRx.Match([]byte(part)) {
			vi.Commit = strings.TrimPrefix(part, "g")
			continue
		}
		if part == "dirty" {
			vi.Dirty = true
			continue
		}
		if distance, err := strconv.Atoi(part); err == nil {
			vi.TagDistance = distance
			continue
		}
		verParts = append(verParts, part)
	}

	ver := strings.Join(verParts, "-")
	if versionRx.Match([]byte(ver)) {
		vi.Semantic = strings.TrimPrefix(ver, "v")
	} else if vi.Commit == "" {
		vi.Commit = ver
	}

	return nil
}

func (vi *VersionInfo) extractVersionFromRuntime() {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, s := range buildInfo.Settings {
		switch s.Key {
		case "vcs.revision":
			commitLen := 10
			if len(s.Value) < commitLen {
				commitLen = len(s.Value)
			}
			if vi.Commit == "" {
				vi.Commit = s.Value[:commitLen]
			}
		case "vcs.modified":
			if s.Value == "true" {
				vi.Dirty = true
			}
		}
	}
}
