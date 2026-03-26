package registry

import (
	"strings"
	"testing"
)

func TestDefaultParseVer(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extract semver",
			input:    "codex 0.34.1\nbuild abcdef",
			expected: "0.34.1",
		},
		{
			name:     "fallback to first line",
			input:    "nightly-build",
			expected: "nightly-build",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual := DefaultParseVer(testCase.input)
			if actual != testCase.expected {
				t.Fatalf("期望版本 %q，实际为 %q", testCase.expected, actual)
			}
		})
	}
}

func TestParseToolsJSONLoadsConfigDrivenRegistry(t *testing.T) {
	content := []byte(`[
		{
			"name": "Example Tool",
			"binary": "example",
			"upgrade_cmd": ["example", "upgrade"],
			"config_paths": ["~/.example/config.json"],
			"current_version": {
				"provider": "command",
				"args": ["--version"]
			},
			"latest_version": {
				"provider": "npm",
				"target": "@example/tool"
			}
		}
	]`)

	tools, err := parseToolsJSON(content)
	if err != nil {
		t.Fatalf("期望 JSON 注册表能够被正确解析，实际报错: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("期望解析出 1 个工具，实际为 %d", len(tools))
	}
	if tools[0].CurrentVersion.ParseStrategy != defaultParseStrategy {
		t.Fatalf("期望默认解析策略为 %q，实际为 %q", defaultParseStrategy, tools[0].CurrentVersion.ParseStrategy)
	}
}

func TestAllToolsIncludesDesktopProviders(t *testing.T) {
	tools := AllTools()
	toolByName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		toolByName[tool.Name] = tool
	}

	expectedNames := []string{"Claude Code", "Codex CLI", "Cursor", "Trae Agent", "Windsurf"}
	for _, name := range expectedNames {
		if _, ok := toolByName[name]; !ok {
			t.Fatalf("期望工具注册表包含 %q", name)
		}
	}

	cursor := toolByName["Cursor"]
	if cursor.CurrentVersion.Provider != CurrentVersionProviderAppBundle {
		t.Fatalf("期望 Cursor 使用 app bundle 读取当前版本，实际为 %q", cursor.CurrentVersion.Provider)
	}
	if cursor.LatestVersion.Provider != LatestVersionProviderHomebrewCask {
		t.Fatalf("期望 Cursor 使用 homebrew cask 提供最新版本，实际为 %q", cursor.LatestVersion.Provider)
	}

	trae := toolByName["Trae Agent"]
	if trae.ConfigPaths[0] != "~/.trae/argv.json" {
		t.Fatalf("期望 Trae Agent 首选配置路径为 ~/.trae/argv.json，实际为 %q", trae.ConfigPaths[0])
	}
	if trae.LatestVersion.Provider != LatestVersionProviderHomebrewCask {
		t.Fatalf("期望 Trae Agent 使用 homebrew cask 提供最新版本，实际为 %q", trae.LatestVersion.Provider)
	}

	windsurf := toolByName["Windsurf"]
	if !strings.Contains(strings.Join(windsurf.CurrentVersion.Paths, ","), "Windsurf.app") {
		t.Fatalf("期望 Windsurf 声明 app bundle 路径，实际为 %v", windsurf.CurrentVersion.Paths)
	}

	gemini := toolByName["Gemini CLI"]
	expectedUpgrade := []string{"npm", "install", "-g", "@google/gemini-cli@latest"}
	if strings.Join(gemini.UpgradeCmd, " ") != strings.Join(expectedUpgrade, " ") {
		t.Fatalf("期望 Gemini CLI 的升级命令为 %v，实际为 %v", expectedUpgrade, gemini.UpgradeCmd)
	}
	if gemini.CurrentVersion.Provider != CurrentVersionProviderNPMGlobal {
		t.Fatalf("期望 Gemini CLI 使用 npm_global 检测当前版本，实际为 %q", gemini.CurrentVersion.Provider)
	}
	if gemini.LatestVersion.Provider != LatestVersionProviderNPMDistTag {
		t.Fatalf("期望 Gemini CLI 使用 npm_dist_tag 检测最新版本，实际为 %q", gemini.LatestVersion.Provider)
	}
}

func TestDisplayLatestVersionProvider(t *testing.T) {
	testCases := []struct {
		provider string
		expected string
	}{
		{provider: LatestVersionProviderNPM, expected: "npm"},
		{provider: LatestVersionProviderNPMDistTag, expected: "npm"},
		{provider: LatestVersionProviderPyPI, expected: "PyPI"},
		{provider: LatestVersionProviderGitHubRelease, expected: "GitHub"},
		{provider: LatestVersionProviderHomebrewCask, expected: "Homebrew"},
		{provider: "", expected: "无上游"},
	}

	for _, testCase := range testCases {
		actual := DisplayLatestVersionProvider(testCase.provider)
		if actual != testCase.expected {
			t.Fatalf("期望 provider %q 显示为 %q，实际为 %q", testCase.provider, testCase.expected, actual)
		}
	}
}
