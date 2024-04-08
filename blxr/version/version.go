package version

import (
	"runtime/debug"
)

var commitHash = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}()

// CommitHash returns the commit hash of the current build
func CommitHash() string { return commitHash }
