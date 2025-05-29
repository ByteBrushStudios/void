package extras

import (
	"runtime/debug"
)

type VoidInfo struct {
	Version string
	Commit  string
}

var (
	version = "2.0.0-alpha.1"
	commit  = ""
)

func init() {
	// Try to get commit from build info
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
			}
		}
	}
	if commit == "" {
		commit = "unknown"
	}
}

// SetVersion allows main.go to set the version at startup/build time
func SetVersion(v string) {
	version = v
}

// GetVoidInfo returns the current version and commit info
func GetVoidInfo() VoidInfo {
	return VoidInfo{
		Version: version,
		Commit:  commit,
	}
}
