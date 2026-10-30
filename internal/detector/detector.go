package detector

import (
	"bytes"
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

type InstallSource string

const (
	InstallSourceUnknown         InstallSource = ""
	InstallSourceNPMGlobal       InstallSource = "npm_global"
	InstallSourceOfficialScript  InstallSource = "official_script"
	InstallSourceHomebrewFormula InstallSource = "homebrew_formula"
	InstallSourceHomebrewCask    InstallSource = "homebrew_cask"
)

type Status struct {
	Tool            registry.Tool
	Installed       bool
	BinaryPath      string
	InstallSource   InstallSource
	HomebrewPackage string
	HomebrewPrefix  string
	Version         string
	LatestVersion   string
	HasConfig       bool
	ConfigPath      string
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
	status.InstallSource = detectInstallSource(tool, binaryPath)
	_, status.HomebrewPackage, status.HomebrewPrefix = detectHomebrewInstall(binaryPath)
	status.Version = detectCurrentVersion(tool, binaryPath)
	status.HasConfig, status.ConfigPath = detectConfig(tool.ConfigPaths)
	status.Tool = resolveVersionSource(status)
	status.LatestVersion = fetchLatest(status.Tool)

	return status
}

func detectCurrentVersion(tool registry.Tool, binaryPath string) string {
	switch tool.CurrentVersion.Provider {
	case registry.CurrentVersionProviderAppBundle:
		version := readAppBundleVersion(tool.CurrentVersion.Paths)
		if version != "" {
			return registry.ParseVersion(version, tool.CurrentVersion.ParseStrategy)
		}
		if len(tool.CurrentVersion.Args) > 0 {
			return readCommandVersion(binaryPath, tool.CurrentVersion.Args, tool.CurrentVersion.ParseStrategy)
		}
	case registry.CurrentVersionProviderNPMGlobal:
		if version := readInstalledNPMVersion(binaryPath, tool.CurrentVersion); version != "" {
			return version
		}
		return readCommandVersion(binaryPath, []string{"--version"}, tool.CurrentVersion.ParseStrategy)
	case "", registry.CurrentVersionProviderCommand:
		return readCommandVersion(binaryPath, tool.CurrentVersion.Args, tool.CurrentVersion.ParseStrategy)
	}
	return ""
}

func readCommandVersion(binary string, args []string, parseStrategy string) string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, binary, args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return ""
	}
	// 启动警告可能先于版本写入 stderr，不能把两条输出流混为版本号。
	if len(bytes.TrimSpace(output)) == 0 {
		output = stderr.Bytes()
	}
	return registry.ParseVersion(string(output), parseStrategy)
}

func readInstalledNPMVersion(binaryPath string, source registry.VersionSource) string {
	resolved, err := filepath.EvalSymlinks(binaryPath)
	if err != nil || source.Target == "" {
		return ""
	}
	for dir := filepath.Dir(resolved); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		content, err := os.ReadFile(filepath.Join(dir, "package.json"))
		if err != nil {
			continue
		}
		var manifest struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if json.Unmarshal(content, &manifest) == nil && manifest.Name == source.Target {
			return registry.ParseVersion(manifest.Version, source.ParseStrategy)
		}
	}
	return ""
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

func detectInstallSource(tool registry.Tool, binaryPath string) InstallSource {
	// Homebrew 的 Node 工具也包含 node_modules，必须先判断实际所属的包。
	if source, _, _ := detectHomebrewInstall(binaryPath); source != InstallSourceUnknown {
		return source
	}
	switch tool.Name {
	case "OpenCode":
		return detectOpenCodeInstallSource(binaryPath)
	case "Kimi Code":
		return detectKimiCodeInstallSource(binaryPath)
	default:
		return detectNPMGlobalInstallSource(binaryPath)
	}
}

func detectHomebrewInstall(binaryPath string) (InstallSource, string, string) {
	for _, candidate := range candidatePaths(binaryPath) {
		for _, entry := range []struct {
			directory string
			source    InstallSource
		}{
			{"Cellar", InstallSourceHomebrewFormula},
			{"Caskroom", InstallSourceHomebrewCask},
		} {
			marker := string(os.PathSeparator) + entry.directory + string(os.PathSeparator)
			prefix, remainder, ok := strings.Cut(candidate, marker)
			parts := strings.Split(remainder, string(os.PathSeparator))
			if ok && prefix != "" && len(parts) >= 3 && parts[0] != "" && parts[1] != "" {
				packageName := parts[0]
				if entry.source == InstallSourceHomebrewFormula {
					var receipt struct {
						Source struct {
							Tap string `json:"tap"`
						} `json:"source"`
					}
					content, err := os.ReadFile(filepath.Join(prefix, entry.directory, parts[0], parts[1], "INSTALL_RECEIPT.json"))
					if err == nil && json.Unmarshal(content, &receipt) == nil && receipt.Source.Tap != "" && receipt.Source.Tap != "homebrew/core" {
						packageName = receipt.Source.Tap + "/" + packageName
					}
				}
				return entry.source, packageName, prefix
			}
		}
	}
	return InstallSourceUnknown, "", ""
}

func resolveVersionSource(status Status) registry.Tool {
	tool := status.Tool
	// OpenCode 1 和 2 使用不同的 npm 包；日常升级保留用户当前的大版本。
	if tool.Name == "OpenCode" && tool.LatestVersion.Provider == registry.LatestVersionProviderNPM &&
		(tool.LatestVersion.Target == "opencode-ai" || tool.LatestVersion.Target == "@opencode/cli") {
		major, err := strconv.Atoi(strings.SplitN(status.Version, ".", 2)[0])
		switch {
		case err == nil && major < 2:
			tool.LatestVersion.Target = "opencode-ai"
		case err == nil:
			tool.LatestVersion.Target = "@opencode/cli"
		default:
			// 读取不到当前版本时，不猜测它属于哪个发布系列。
			tool.LatestVersion = registry.VersionSource{}
		}
	}
	if status.HomebrewPackage != "" && len(tool.UpgradeCmd) > 0 {
		switch status.InstallSource {
		case InstallSourceHomebrewCask:
			tool.LatestVersion = registry.VersionSource{
				Provider: registry.LatestVersionProviderHomebrewCask,
				Target:   status.HomebrewPackage,
			}
		case InstallSourceHomebrewFormula:
			tool.LatestVersion = registry.VersionSource{
				Provider: registry.LatestVersionProviderHomebrewFormula,
				Target:   status.HomebrewPackage,
			}
			// 第三方 tap 不在 Homebrew 公共 API 中，查询其安装时所属的 brew。
			if strings.Contains(status.HomebrewPackage, "/") {
				tool.LatestVersion.Args = []string{
					filepath.Join(status.HomebrewPrefix, "bin", "brew"), "info", "--json=v2", "--formula", status.HomebrewPackage,
				}
			}
		}
	}
	return tool
}

func detectOpenCodeInstallSource(binaryPath string) InstallSource {
	if source := detectNPMGlobalInstallSource(binaryPath); source != InstallSourceUnknown {
		return source
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		if pathMatchesDir(binaryPath, filepath.Join(homeDir, ".opencode", "bin")) {
			return InstallSourceOfficialScript
		}
	}

	return InstallSourceUnknown
}

func detectKimiCodeInstallSource(binaryPath string) InstallSource {
	if source := detectNPMGlobalInstallSource(binaryPath); source != InstallSourceUnknown {
		return source
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		if pathMatchesDir(binaryPath, filepath.Join(homeDir, ".kimi-code", "bin")) {
			return InstallSourceOfficialScript
		}
	}

	return InstallSourceUnknown
}

func detectNPMGlobalInstallSource(binaryPath string) InstallSource {
	paths := candidatePaths(binaryPath)
	for _, candidatePath := range paths {
		normalizedPath := filepath.ToSlash(candidatePath)
		// 这些包管理器也使用 node_modules，交给工具自身识别，不能强制当作 npm。
		if strings.Contains(normalizedPath, "/.bun/") || strings.Contains(normalizedPath, "/pnpm/") ||
			strings.Contains(normalizedPath, "/.pnpm/") || strings.Contains(normalizedPath, "/yarn/") {
			return InstallSourceUnknown
		}
	}
	for _, candidatePath := range paths {
		normalizedPath := filepath.ToSlash(candidatePath)
		if strings.Contains(normalizedPath, "/node_modules/") {
			return InstallSourceNPMGlobal
		}
	}
	return InstallSourceUnknown
}

func candidatePaths(path string) []string {
	paths := make([]string, 0, 2)

	appendPath := func(candidate string) {
		if candidate == "" {
			return
		}
		absolutePath, err := filepath.Abs(candidate)
		if err != nil {
			absolutePath = filepath.Clean(candidate)
		}
		for _, existingPath := range paths {
			if existingPath == absolutePath {
				return
			}
		}
		paths = append(paths, absolutePath)
	}

	appendPath(path)
	if resolvedPath, err := filepath.EvalSymlinks(path); err == nil {
		appendPath(resolvedPath)
	}

	return paths
}

func pathMatchesDir(path string, dir string) bool {
	if path == "" || dir == "" {
		return false
	}

	for _, candidatePath := range candidatePaths(path) {
		relativePath, err := filepath.Rel(dir, candidatePath)
		if err != nil {
			continue
		}
		if relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(os.PathSeparator))) {
			return true
		}
	}
	return false
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
	case registry.LatestVersionProviderHomebrewFormula:
		return fetchHomebrewFormulaLatest(tool.LatestVersion)
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

func fetchHomebrewFormulaLatest(source registry.VersionSource) string {
	if len(source.Args) > 0 {
		var data struct {
			Formulae []struct {
				Versions struct {
					Stable string `json:"stable"`
				} `json:"versions"`
			} `json:"formulae"`
		}
		output := runCommand(source.Args[0], source.Args[1:])
		if json.Unmarshal([]byte(output), &data) != nil || len(data.Formulae) != 1 {
			return ""
		}
		return registry.ParseVersion(data.Formulae[0].Versions.Stable, source.ParseStrategy)
	}
	url := fmt.Sprintf("https://formulae.brew.sh/api/formula/%s.json", source.Target)
	var data struct {
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}
	if err := fetchJSON(url, source.Headers, &data); err != nil {
		return ""
	}
	return registry.ParseVersion(data.Versions.Stable, source.ParseStrategy)
}

func runCommand(binary string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), latestVersionTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, binary, args...)
	output, err := command.Output()
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
