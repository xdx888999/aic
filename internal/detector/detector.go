package detector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xdx888999/aic/internal/registry"
)

const (
	commandTimeout       = 5 * time.Second
	latestVersionTimeout = 8 * time.Second
	plistBuddyPath       = "/usr/libexec/PlistBuddy"
	plistVersionKey      = "CFBundleShortVersionString"
	plistBuildKey        = "CFBundleVersion"
)

var versionTokenRe = regexp.MustCompile(`\d+|[A-Za-z]+`)

type Status struct {
	Tool          registry.Tool
	Installed     bool
	BinaryPath    string
	Version       string
	LatestVersion string
	HasConfig     bool
	ConfigPath    string
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func Detect(tool registry.Tool) Status {
	status := Status{Tool: tool}

	binaryPath, err := exec.LookPath(tool.Binary)
	if err != nil {
		return status
	}

	status.Installed = true
	status.BinaryPath = binaryPath
	status.Version = detectCurrentVersion(tool)
	status.HasConfig, status.ConfigPath = detectConfig(tool.ConfigPaths)
	status.LatestVersion = fetchLatest(tool)

	return status
}

func detectCurrentVersion(tool registry.Tool) string {
	switch tool.CurrentVersion.Provider {
	case registry.CurrentVersionProviderAppBundle:
		version := readAppBundleVersion(tool.CurrentVersion.Paths)
		if version != "" {
			return registry.ParseVersion(version, tool.CurrentVersion.ParseStrategy)
		}
		if len(tool.CurrentVersion.Args) > 0 {
			return readCommandVersion(tool.Binary, tool.CurrentVersion.Args, tool.CurrentVersion.ParseStrategy)
		}
	case registry.CurrentVersionProviderNPMGlobal:
		return readNPMGlobalVersion(tool.CurrentVersion)
	case "", registry.CurrentVersionProviderCommand:
		return readCommandVersion(tool.Binary, tool.CurrentVersion.Args, tool.CurrentVersion.ParseStrategy)
	}
	return ""
}

func readCommandVersion(binary string, args []string, parseStrategy string) string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, binary, args...)
	output, err := command.CombinedOutput()
	if err != nil || len(output) == 0 {
		return ""
	}
	return registry.ParseVersion(string(output), parseStrategy)
}

func readAppBundleVersion(paths []string) string {
	if runtime.GOOS != "darwin" {
		return ""
	}

	for _, path := range paths {
		appPath := expandHome(path)
		if _, err := os.Stat(appPath); err != nil {
			continue
		}

		infoPlistPath := filepath.Join(appPath, "Contents", "Info.plist")
		if version := readPlistValue(infoPlistPath, plistVersionKey); version != "" {
			return version
		}
		if buildVersion := readPlistValue(infoPlistPath, plistBuildKey); buildVersion != "" {
			return buildVersion
		}
	}

	return ""
}

func readPlistValue(plistPath string, key string) string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, plistBuddyPath, "-c", "Print :"+key, plistPath)
	output, err := command.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func detectConfig(paths []string) (bool, string) {
	for _, rawPath := range paths {
		path := expandHome(rawPath)
		if _, err := os.Stat(path); err == nil {
			return true, path
		}
	}
	return false, ""
}

func fetchLatest(tool registry.Tool) string {
	switch tool.LatestVersion.Provider {
	case registry.LatestVersionProviderNPM:
		return fetchNpmLatest(tool.LatestVersion)
	case registry.LatestVersionProviderNPMDistTag:
		return fetchNPMDistTagLatest(tool.LatestVersion)
	case registry.LatestVersionProviderPyPI:
		return fetchPyPILatest(tool.LatestVersion)
	case registry.LatestVersionProviderGitHubRelease:
		return fetchGitHubLatest(tool.LatestVersion)
	case registry.LatestVersionProviderHomebrewCask:
		return fetchHomebrewCaskLatest(tool.LatestVersion)
	default:
		return ""
	}
}

func readNPMGlobalVersion(source registry.VersionSource) string {
	if source.Target == "" {
		return ""
	}

	output := runCommand("npm", []string{"list", "-g", "--depth=0", source.Target})
	if output == "" {
		return ""
	}

	return parseNPMGlobalVersion(output, source.Target, source.ParseStrategy)
}

func fetchNpmLatest(source registry.VersionSource) string {
	url := fmt.Sprintf("https://registry.npmjs.org/%s/latest", source.Target)

	var data struct {
		Version string `json:"version"`
	}
	if err := fetchJSON(url, source.Headers, &data); err != nil {
		return ""
	}
	return registry.ParseVersion(data.Version, source.ParseStrategy)
}

func fetchNPMDistTagLatest(source registry.VersionSource) string {
	if source.Target == "" {
		return ""
	}

	output := runCommand("npm", []string{"view", source.Target, "dist-tags.latest", "--json"})
	if output == "" {
		return ""
	}

	return registry.ParseVersion(normalizeJSONCommandValue(output), source.ParseStrategy)
}

func fetchPyPILatest(source registry.VersionSource) string {
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", source.Target)

	var data struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := fetchJSON(url, source.Headers, &data); err != nil {
		return ""
	}
	return registry.ParseVersion(data.Info.Version, source.ParseStrategy)
}

func fetchGitHubLatest(source registry.VersionSource) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", source.Target)

	headers := cloneHeaders(source.Headers)
	headers["Accept"] = "application/vnd.github+json"
	headers["User-Agent"] = "aic"

	var data struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
	}
	if err := fetchJSON(url, headers, &data); err != nil {
		return ""
	}
	if data.TagName != "" {
		return registry.ParseVersion(data.TagName, source.ParseStrategy)
	}
	return registry.ParseVersion(data.Name, source.ParseStrategy)
}

func fetchHomebrewCaskLatest(source registry.VersionSource) string {
	url := fmt.Sprintf("https://formulae.brew.sh/api/cask/%s.json", source.Target)

	var data struct {
		Version string `json:"version"`
	}
	if err := fetchJSON(url, source.Headers, &data); err != nil {
		return ""
	}
	return registry.ParseVersion(data.Version, source.ParseStrategy)
}

func runCommand(binary string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), latestVersionTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, binary, args...)
	output, err := command.CombinedOutput()
	if err != nil || len(output) == 0 {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func parseNPMGlobalVersion(output string, packageName string, parseStrategy string) string {
	pattern := regexp.MustCompile(regexp.QuoteMeta(packageName) + `@([0-9A-Za-z.\-_]+)`)
	matches := pattern.FindStringSubmatch(output)
	if len(matches) < 2 {
		return ""
	}
	return registry.ParseVersion(matches[1], parseStrategy)
}

func normalizeJSONCommandValue(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.Trim(trimmed, "\"")
}

func fetchJSON(url string, headers map[string]string, target any) error {
	ctx, cancel := context.WithTimeout(context.Background(), latestVersionTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", response.StatusCode)
	}

	return json.NewDecoder(response.Body).Decode(target)
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return make(map[string]string)
	}

	cloned := make(map[string]string, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}

func CompareVersions(current string, latest string) int {
	currentTokens := versionTokenRe.FindAllString(current, -1)
	latestTokens := versionTokenRe.FindAllString(latest, -1)

	if len(currentTokens) == 0 || len(latestTokens) == 0 {
		return strings.Compare(current, latest)
	}

	limit := len(currentTokens)
	if len(latestTokens) > limit {
		limit = len(latestTokens)
	}

	for index := 0; index < limit; index++ {
		if index >= len(currentTokens) {
			return -1
		}
		if index >= len(latestTokens) {
			return 1
		}

		currentToken := currentTokens[index]
		latestToken := latestTokens[index]

		currentNumber, currentErr := strconv.Atoi(currentToken)
		latestNumber, latestErr := strconv.Atoi(latestToken)
		if currentErr == nil && latestErr == nil {
			switch {
			case currentNumber < latestNumber:
				return -1
			case currentNumber > latestNumber:
				return 1
			default:
				continue
			}
		}

		compareResult := strings.Compare(strings.ToLower(currentToken), strings.ToLower(latestToken))
		if compareResult != 0 {
			return compareResult
		}
	}

	return 0
}

func DetectAll(tools []registry.Tool) []Status {
	results := make([]Status, len(tools))
	var waitGroup sync.WaitGroup

	for index, tool := range tools {
		waitGroup.Add(1)
		go func(toolIndex int, currentTool registry.Tool) {
			defer waitGroup.Done()
			results[toolIndex] = Detect(currentTool)
		}(index, tool)
	}

	waitGroup.Wait()
	return results
}
