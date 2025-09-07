package version

import (
	"fmt"
	"runtime"
)

// Build-time variables - these will be injected during compilation
var (
	// Version is the semantic version (e.g., "1.0.0")
	Version = "dev"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is when the binary was built
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()
)

// BuildInfo returns comprehensive build information
func BuildInfo() string {
	return fmt.Sprintf(`simple-secrets %s
Git Commit: %s
Built: %s
Go Version: %s
Platform: %s/%s`,
		Version,
		GitCommit,
		BuildDate,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Short returns just the version string
func Short() string {
	if Version == "dev" {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}
