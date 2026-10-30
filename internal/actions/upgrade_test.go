package actions

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/registry"
)

func TestUpgradeCommandFollowsInstallationSource(t *testing.T) {
	tools := make(map[string]registry.Tool)
	for _, tool := range registry.AllTools() {
		tools[tool.Name] = tool
	}
	for _, name := range []string{"Claude Code", "Codex CLI", "Qwen Code", "Grok Build", "Kilo Code"} {
		t.Run(name, func(t *testing.T) {
			tool := tools[name]
			binaryPath := filepath.Join(t.TempDir(), tool.Binary)
			status := detector.Status{Tool: tool, Installed: true, BinaryPath: binaryPath, InstallSource: detector.InstallSourceNPMGlobal}
			if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, tool.UpgradeCmd) {
				t.Fatalf("npm 安装使用了错误命令: %v", actual)
			}
			status.InstallSource = detector.InstallSourceUnknown
			expected := append([]string{binaryPath}, tool.NativeUpgradeCmd[1:]...)
			if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, expected) {
				t.Fatalf("原生安装升级命令为 %v，期望 %v", actual, expected)
			}
		})
	}
}

func TestHomebrewUpgradeUsesInstalledPackageAndPrefix(t *testing.T) {
	for _, tc := range []struct {
		name, packageName, kind string
		source                  detector.InstallSource
	}{
		{"Grok Build", "grok-build", "--cask", detector.InstallSourceHomebrewCask},
		{"Codex CLI", "codex", "--cask", detector.InstallSourceHomebrewCask},
		{"OpenCode", "opencode", "--formula", detector.InstallSourceHomebrewFormula},
		{"OpenCode", "anomalyco/tap/opencode-v2", "--formula", detector.InstallSourceHomebrewFormula},
	} {
		t.Run(tc.packageName, func(t *testing.T) {
			prefix := t.TempDir()
			status := detector.Status{
				Installed: true, InstallSource: tc.source, HomebrewPackage: tc.packageName, HomebrewPrefix: prefix,
				Tool: registry.Tool{Name: tc.name, UpgradeCmd: []string{"npm", "install", "-g", "wrong-package"}},
			}
			expected := []string{filepath.Join(prefix, "bin", "brew"), "upgrade", tc.kind, tc.packageName}
			if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, expected) {
				t.Fatalf("Homebrew 升级命令为 %v，期望 %v", actual, expected)
			}
		})
	}
}

func TestOpenCodeUpgradeUsesNewerDetectedVersion(t *testing.T) {
	for _, tc := range []struct {
		name, current, latest string
		target                []string
	}{
		{"upgrade v2", "2.1.3", "2.2.2", []string{"2.2.2"}},
		{"keep v1", "1.19.0", "1.19.5", []string{"1.19.5"}},
		{"stale source", "2.1.3", "1.19.5", nil},
		{"up to date", "2.2.2", "2.2.2", nil},
		{"unknown latest", "2.1.3", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "opencode")
			status := detector.Status{
				Installed: true, BinaryPath: binary, InstallSource: detector.InstallSourceOfficialScript,
				Version: tc.current, LatestVersion: tc.latest, Tool: registry.Tool{Name: "OpenCode"},
			}
			expected := append([]string{binary, "upgrade"}, tc.target...)
			expected = append(expected, "--method", "curl")
			if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, expected) {
				t.Fatalf("OpenCode 升级命令为 %v，期望 %v", actual, expected)
			}
		})
	}
}

func TestNPMUpgradeStopsWhenPrefixCannotBeVerified(t *testing.T) {
	original := lookupNPMGlobalBinDir
	t.Cleanup(func() { lookupNPMGlobalBinDir = original })
	lookupNPMGlobalBinDir = func() (string, error) { return "", errors.New("npm failed") }
	status := detector.Status{Installed: true, BinaryPath: "/some/bin/tool"}
	err := validateUpgradeTarget(status, []string{"npm", "install", "-g", "example"})
	var actionErr ActionError
	if !errors.As(err, &actionErr) || actionErr.Code != ErrorUpgradeTargetUnverified {
		t.Fatalf("无法验证 npm 安装位置时必须取消升级: %v", err)
	}
}

func TestOpenCodeSelfUpgradeValidatesNPMEnvironment(t *testing.T) {
	original := lookupNPMGlobalBinDir
	t.Cleanup(func() { lookupNPMGlobalBinDir = original })
	lookupNPMGlobalBinDir = func() (string, error) { return "/other-node/bin", nil }
	status := detector.Status{
		Installed: true, BinaryPath: "/old-node/bin/opencode", InstallSource: detector.InstallSourceNPMGlobal,
		Tool: registry.Tool{Name: "OpenCode"},
	}
	err := validateUpgradeTarget(status, resolveUpgradeCommand(status))
	var actionErr ActionError
	if !errors.As(err, &actionErr) || actionErr.Code != ErrorUpgradeTargetMismatch {
		t.Fatalf("OpenCode 也必须防止升级另一套 npm 环境: %v", err)
	}
}
