package registry

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	defaultParseStrategy     = "semver"
	defaultConfigEnvKey      = "AIC_TOOLS_FILE"
	defaultConfigFileName    = "tools.json"
	repositoryConfigFilePath = "internal/registry/tools.json"
)

const (
	CurrentVersionProviderCommand   = "command"
	CurrentVersionProviderAppBundle = "app_bundle"
	CurrentVersionProviderNPMGlobal = "npm_global"
)

const (
	LatestVersionProviderNone          = ""
	LatestVersionProviderNPM           = "npm"
	LatestVersionProviderNPMDistTag    = "npm_dist_tag"
	LatestVersionProviderPyPI          = "pypi"
	LatestVersionProviderGitHubRelease = "github_release"
	LatestVersionProviderHomebrewCask  = "homebrew_cask"
)

type Tool struct {
	Name           string        `json:"name"`
	Binary         string        `json:"binary"`
	UpgradeCmd     []string      `json:"upgrade_cmd"`
	ConfigPaths    []string      `json:"config_paths"`
	CurrentVersion VersionSource `json:"current_version"`
	LatestVersion  VersionSource `json:"latest_version"`
}

type VersionSource struct {
	Provider      string            `json:"provider"`
	Target        string            `json:"target"`
	Args          []string          `json:"args"`
	Paths         []string          `json:"paths"`
	URL           string            `json:"url"`
	Pattern       string            `json:"pattern"`
	Headers       map[string]string `json:"headers"`
	ParseStrategy string            `json:"parse_strategy"`
}

//go:embed tools.json
var embeddedToolsJSON []byte

var semverRe = regexp.MustCompile(`\d+\.\d+[\.\d]*`)

func DefaultParseVer(raw string) string {
	line := strings.TrimSpace(strings.SplitN(raw, "\n", 2)[0])
	if match := semverRe.FindString(line); match != "" {
		return match
	}
	return line
}

func ParseVersion(raw string, strategy string) string {
	switch strings.TrimSpace(strategy) {
	case "", defaultParseStrategy:
		return DefaultParseVer(raw)
	default:
		return strings.TrimSpace(strings.SplitN(raw, "\n", 2)[0])
	}
}

func AllTools() []Tool {
	return loadTools()
}

func loadTools() []Tool {
	for _, path := range candidateConfigPaths() {
		tools, err := loadToolsFromFile(path)
		if err == nil {
			return tools
		}
	}

	tools, err := parseToolsJSON(embeddedToolsJSON)
	if err != nil {
		return []Tool{}
	}
	return tools
}

func candidateConfigPaths() []string {
	candidates := make([]string, 0, 4)
	seen := make(map[string]bool, 4)

	appendPath := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}

		resolved, err := filepath.Abs(path)
		if err != nil {
			resolved = path
		}
		if seen[resolved] {
			return
		}
		seen[resolved] = true
		candidates = append(candidates, resolved)
	}

	appendPath(os.Getenv(defaultConfigEnvKey))

	if cwd, err := os.Getwd(); err == nil {
		appendPath(filepath.Join(cwd, defaultConfigFileName))
		appendPath(filepath.Join(cwd, repositoryConfigFilePath))
	}

	if executablePath, err := os.Executable(); err == nil {
		appendPath(filepath.Join(filepath.Dir(executablePath), defaultConfigFileName))
	}

	return candidates
}

func loadToolsFromFile(path string) ([]Tool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseToolsJSON(content)
}

func parseToolsJSON(content []byte) ([]Tool, error) {
	var tools []Tool
	if err := json.Unmarshal(content, &tools); err != nil {
		return nil, err
	}

	for index := range tools {
		normalizeTool(&tools[index])
	}

	return tools, nil
}

func normalizeTool(tool *Tool) {
	if tool.CurrentVersion.ParseStrategy == "" {
		tool.CurrentVersion.ParseStrategy = defaultParseStrategy
	}
	if tool.LatestVersion.ParseStrategy == "" {
		tool.LatestVersion.ParseStrategy = defaultParseStrategy
	}
}

func DisplayLatestVersionProvider(provider string) string {
	switch provider {
	case LatestVersionProviderNPM:
		return "npm"
	case LatestVersionProviderNPMDistTag:
		return "npm"
	case LatestVersionProviderPyPI:
		return "PyPI"
	case LatestVersionProviderGitHubRelease:
		return "GitHub"
	case LatestVersionProviderHomebrewCask:
		return "Homebrew"
	default:
		return ""
	}
}
