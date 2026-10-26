package thisvid_downloader

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BRUHItsABunny/go-ghvu"
)

// Set with -ldflags -X <module>.<var>=... (this file is the module-root package).
var (
	AppVersion      = "v0.0.1"
	BuildTime       = ""
	GitCommit       = ""
	GitRef          = ""
	GitRepo         = "https://github.com/RoyalFlyBy/ThisVidDownloader"
	CurrentVersion  *ghvu.Version
	CurrentCodeBase *ghvu.CodeBase
)

func init() {
	CurrentVersion = ghvu.NewVersionOrDefault(AppVersion, GitCommit, GitRef, BuildTime)
	repoSplit := strings.Split(strings.TrimSuffix(GitRepo, "/"), "/")
	CurrentCodeBase = ghvu.NewCodeBase(nil, repoSplit[len(repoSplit)-2], repoSplit[len(repoSplit)-1])
}

const versionCacheFile = "thisvid_downloader.version_check"

// VersionRoutine prints the running version and, at most once a day, hints at a
// newer release. A failed check never blocks a download.
func VersionRoutine() string {
	result := strings.Builder{}
	result.WriteString(CurrentCodeBase.PromptCurrentVersion(CurrentVersion).Output)

	if data, err := os.ReadFile(versionCacheFile); err == nil &&
		strings.TrimSpace(string(data)) == time.Now().Format("2006-01-02") {
		return result.String()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	latestVersion, err := CurrentCodeBase.GetLatestVersion(ctx, nil)
	if err != nil {
		return result.String() // offline, rate-limited or untagged: skip the hint
	}
	_ = os.WriteFile(versionCacheFile, []byte(time.Now().Format("2006-01-02")), 0666)

	if isOutdated, latestPrompt := CurrentCodeBase.PromptLatestVersion(CurrentVersion, latestVersion); isOutdated {
		result.WriteString("\n")
		result.WriteString(latestPrompt.Output)
		result.WriteString(fmt.Sprintf("You can find more here:\n%s\n", latestPrompt.UpdateURL))
	}
	return result.String()
}
