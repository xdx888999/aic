package detector

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xdx888999/aic/internal/registry"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func mockVersionResponse(t *testing.T, expectedURL string, statusCode int, body string) {
	t.Helper()
	original := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = original })
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != expectedURL {
			t.Errorf("查询了错误的更新源: %s，期望 %s", request.URL, expectedURL)
		}
		return &http.Response{StatusCode: statusCode, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
}

func writeExecutable(t *testing.T, path, script string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func builtinTool(t *testing.T, name string) registry.Tool {
	t.Helper()
	for _, tool := range registry.AllTools() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("注册表缺少工具 %s", name)
	return registry.Tool{}
}

func TestDetectOpenCodeFindsLatestReleaseAcrossMajorVersions(t *testing.T) {
	for _, tc := range []struct {
		name, output, current, target, latest string
	}{
		{"legacy latest", "1.19.5", "1.19.5", "@opencode/cli", "2.2.2"},
		{"legacy older", "1.19.0", "1.19.0", "@opencode/cli", "2.2.2"},
		{"v2", "opencode v2.1.3", "2.1.3", "@opencode/cli", "2.2.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			writeExecutable(t, filepath.Join(binDir, "opencode"), "echo '"+tc.output+"'\n")
			t.Setenv("PATH", binDir)
			mockVersionResponse(t, "https://registry.npmjs.org/"+tc.target+"/latest", http.StatusOK, `{"version":"`+tc.latest+`"}`)
			status := Detect(builtinTool(t, "OpenCode"))
			if !status.Installed || status.Version != tc.current || status.LatestVersion != tc.latest {
				t.Fatalf("版本检测错误: %+v", status)
			}
			if status.Tool.LatestVersion.Target != tc.target {
				t.Fatalf("更新源未保留在扫描结果中: %+v", status.Tool.LatestVersion)
			}
		})
	}
}

func TestDetectGrokBuildInstalledByHomebrew(t *testing.T) {
	prefix := t.TempDir()
	installed := filepath.Join(prefix, "Caskroom", "grok-build", "1.0.70", "grok-native")
	writeExecutable(t, installed, "echo 'grok 1.0.70 (f50904eaee6c)'\n")
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(binDir, "grok")
	if err := os.Symlink(installed, binaryPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	mockVersionResponse(t, "https://formulae.brew.sh/api/cask/grok-build.json", http.StatusOK, `{"version":"1.0.71"}`)
	status := Detect(builtinTool(t, "Grok Build"))
	if !status.Installed || status.Version != "1.0.70" || status.LatestVersion != "1.0.71" || status.BinaryPath != binaryPath {
		t.Fatalf("未正确识别 Grok Build: %+v", status)
	}
	resolvedPrefix, err := filepath.EvalSymlinks(prefix)
	if err != nil {
		t.Fatal(err)
	}
	if status.InstallSource != InstallSourceHomebrewCask || status.HomebrewPackage != "grok-build" || status.HomebrewPrefix != resolvedPrefix {
		t.Fatalf("未正确保留 Homebrew 安装位置: %+v", status)
	}
}

func TestReadCommandVersionSeparatesWarningsFromVersion(t *testing.T) {
	for _, tc := range []struct{ name, script, expected string }{
		{"warning before stdout", "echo 'WARNING: unable to create PATH aliases' >&2\necho 'codex-cli 0.165.0'\n", "0.165.0"},
		{"version on stderr", "echo 'grok 1.0.70' >&2\n", "1.0.70"},
		{"failed command", "echo 'opencode v2.1.3'\nexit 1\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tool")
			writeExecutable(t, path, tc.script)
			if actual := readCommandVersion(path, []string{"--version"}, "semver"); actual != tc.expected {
				t.Fatalf("版本号为 %q，期望 %q", actual, tc.expected)
			}
		})
	}
}

func TestCurrentNPMVersionBelongsToDetectedExecutable(t *testing.T) {
	dir := t.TempDir()
	packageDir := filepath.Join(dir, "lib", "node_modules", "@google", "gemini-cli")
	installed := filepath.Join(packageDir, "dist", "index.js")
	writeExecutable(t, installed, "exit 99\n")
	if err := os.WriteFile(filepath.Join(packageDir, "package.json"), []byte(`{"name":"@google/gemini-cli","version":"0.33.1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "gemini")
	if err := os.Symlink(installed, link); err != nil {
		t.Fatal(err)
	}
	// 即使 PATH 中的 npm 属于另一套 Node 环境，也只能读取命中的 Gemini 包。
	t.Setenv("PATH", t.TempDir())
	if actual := detectCurrentVersion(builtinTool(t, "Gemini CLI"), link); actual != "0.33.1" {
		t.Fatalf("读取了错误的 Gemini 版本: %q", actual)
	}
}

func TestDetectInstallSourceDoesNotTreatOtherManagersAsNPM(t *testing.T) {
	for _, tc := range []struct {
		path     string
		expected InstallSource
	}{
		{"/opt/homebrew/Cellar/gemini-cli/1.0/libexec/lib/node_modules/@google/gemini-cli/index.js", InstallSourceHomebrewFormula},
		{"/opt/homebrew/Caskroom/grok-build/1.0/grok", InstallSourceHomebrewCask},
		{"/usr/local/lib/node_modules/@opencode/cli/bin/opencode", InstallSourceNPMGlobal},
		{"/Users/test/.bun/install/global/node_modules/@opencode/cli/bin/opencode", InstallSourceUnknown},
		{"/Users/test/Library/pnpm/global/5/node_modules/@opencode/cli/bin/opencode", InstallSourceUnknown},
		{"/Users/test/.config/yarn/global/node_modules/@opencode/cli/bin/opencode", InstallSourceUnknown},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if actual := detectInstallSource(registry.Tool{Name: "OpenCode"}, tc.path); actual != tc.expected {
				t.Fatalf("安装源为 %q，期望 %q", actual, tc.expected)
			}
		})
	}
}

func TestHomebrewTapKeepsItsPackageAndUsesOwningBrew(t *testing.T) {
	prefix := t.TempDir()
	packageDir := filepath.Join(prefix, "Cellar", "opencode-v2", "2.1.3")
	binaryPath := filepath.Join(packageDir, "bin", "opencode")
	writeExecutable(t, binaryPath, "echo 'opencode v2.1.3'\n")
	if err := os.WriteFile(filepath.Join(packageDir, "INSTALL_RECEIPT.json"), []byte(`{"source":{"tap":"anomalyco/tap"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	source, name, root := detectHomebrewInstall(binaryPath)
	if source != InstallSourceHomebrewFormula || name != "anomalyco/tap/opencode-v2" || root != prefix {
		t.Fatalf("tap 识别错误: %s %s %s", source, name, root)
	}
	brew := filepath.Join(prefix, "bin", "brew")
	writeExecutable(t, brew, "echo 'brew warning' >&2\necho '{\"formulae\":[{\"versions\":{\"stable\":\"2.2.2\"}}]}'\n")
	tool := resolveVersionSource(Status{Tool: builtinTool(t, "OpenCode"), Version: "2.1.3", InstallSource: source, HomebrewPackage: name, HomebrewPrefix: root})
	if actual := fetchLatest(tool); actual != "2.2.2" {
		t.Fatalf("未能查询 tap 最新版本: %q", actual)
	}
}

func TestLatestVersionFailuresDoNotInventAVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		code int
		body string
	}{
		{"unavailable", http.StatusServiceUnavailable, `{"version":"2.2.2"}`},
		{"invalid json", http.StatusOK, "upstream error"},
		{"missing version", http.StatusOK, `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockVersionResponse(t, "https://registry.npmjs.org/@opencode/cli/latest", tc.code, tc.body)
			if actual := fetchLatest(builtinTool(t, "OpenCode")); actual != "" {
				t.Fatalf("失败时不应显示版本 %q", actual)
			}
		})
	}
}

func TestUnknownOpenCodeVersionStillQueriesCurrentRelease(t *testing.T) {
	tool := resolveVersionSource(Status{Tool: builtinTool(t, "OpenCode")})
	if tool.LatestVersion.Provider != registry.LatestVersionProviderNPM || tool.LatestVersion.Target != "@opencode/cli" {
		t.Fatalf("当前版本读取失败不应阻止查询最新发行版: %+v", tool.LatestVersion)
	}
}

func TestLegacyHomebrewTapQueriesV2Formula(t *testing.T) {
	tool := builtinTool(t, "OpenCode")
	prefix := t.TempDir()
	status := Status{
		Tool: tool, Version: "1.19.5", InstallSource: InstallSourceHomebrewFormula,
		HomebrewPackage: "anomalyco/tap/opencode", HomebrewPrefix: prefix,
	}
	resolved := resolveVersionSource(status)
	if resolved.LatestVersion.Target != "anomalyco/tap/opencode-v2" || resolved.LatestVersion.Provider != registry.LatestVersionProviderHomebrewFormula {
		t.Fatalf("旧 tap 应查询可迁移的 v2 formula: %+v", resolved.LatestVersion)
	}
	if got := resolved.LatestVersion.Args[len(resolved.LatestVersion.Args)-1]; got != resolved.LatestVersion.Target {
		t.Fatalf("brew info 查询了旧 formula: %v", resolved.LatestVersion.Args)
	}
}

func TestFetchHomebrewFormulaVersion(t *testing.T) {
	mockVersionResponse(t, "https://formulae.brew.sh/api/formula/opencode.json", http.StatusOK, `{"versions":{"stable":"2.2.2","head":null}}`)
	if actual := fetchHomebrewFormulaLatest(registry.VersionSource{Target: "opencode"}); actual != "2.2.2" {
		t.Fatalf("Homebrew formula 版本错误: %q", actual)
	}
}
