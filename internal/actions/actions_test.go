package actions

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/registry"
)

func TestUpgradeCmdReturnsErrorWhenNoCmdConfigured(t *testing.T) {
	status := detector.Status{Tool: registry.Tool{Name: "NoCmd", UpgradeCmd: []string{}}}
	cmd := UpgradeCmd(status, 0)

	msg := cmd()
	finished, ok := msg.(UpgradeFinishedMsg)
	if !ok {
		t.Fatalf("期望返回 UpgradeFinishedMsg，实际为 %T", msg)
	}

	var actionErr ActionError
	if !errors.As(finished.Err, &actionErr) {
		t.Fatalf("期望错误为 ActionError，实际为 %v", finished.Err)
	}
	if actionErr.Code != ErrorUpgradeNotSupported {
		t.Fatalf("期望错误码为 %q，实际为 %q", ErrorUpgradeNotSupported, actionErr.Code)
	}
}

func TestUpgradeCmdReturnsErrorWhenPackageManagerMissing(t *testing.T) {
	status := detector.Status{
		Tool: registry.Tool{
			Name:       "MissingPkgMgr",
			UpgradeCmd: []string{"__nonexistent_pkg_manager__", "install", "something"},
		},
	}
	cmd := UpgradeCmd(status, 0)

	msg := cmd()
	finished, ok := msg.(UpgradeFinishedMsg)
	if !ok {
		t.Fatalf("期望返回 UpgradeFinishedMsg，实际为 %T", msg)
	}

	var actionErr ActionError
	if !errors.As(finished.Err, &actionErr) {
		t.Fatalf("期望错误为 ActionError，实际为 %v", finished.Err)
	}
	if actionErr.Code != ErrorMissingPackageManager {
		t.Fatalf("期望错误码为 %q，实际为 %q", ErrorMissingPackageManager, actionErr.Code)
	}
	if actionErr.Arg != "__nonexistent_pkg_manager__" {
		t.Fatalf("期望错误参数为 %q，实际为 %q", "__nonexistent_pkg_manager__", actionErr.Arg)
	}
}

func TestActionErrorMessageIncludesArg(t *testing.T) {
	err := ActionError{Code: ErrorMissingPackageManager, Arg: "npm"}
	if err.Error() != "missing_package_manager:npm" {
		t.Fatalf("期望错误信息为 %q，实际为 %q", "missing_package_manager:npm", err.Error())
	}
}

func TestActionErrorMessageIncludesHint(t *testing.T) {
	err := ActionError{
		Code: ErrorUpgradeTargetMismatch,
		Arg:  "/tmp/opencode",
		Hint: "npm_global",
	}
	if err.Error() != "upgrade_target_mismatch:/tmp/opencode:npm_global" {
		t.Fatalf("期望错误信息为 %q，实际为 %q", "upgrade_target_mismatch:/tmp/opencode:npm_global", err.Error())
	}
}

func TestActionErrorMessageWithoutArg(t *testing.T) {
	err := ActionError{Code: ErrorUpgradeNotSupported}
	if err.Error() != "upgrade_not_supported" {
		t.Fatalf("期望错误信息为 %q，实际为 %q", "upgrade_not_supported", err.Error())
	}
}

func TestResolveEditorReturnsEnvVar(t *testing.T) {
	t.Setenv("EDITOR", "nano")
	if got := ResolveEditor(); got != "nano" {
		t.Fatalf("期望编辑器为 %q，实际为 %q", "nano", got)
	}
}

func TestResolveEditorDefaultsToVim(t *testing.T) {
	os.Unsetenv("EDITOR")
	if got := ResolveEditor(); got != "vim" {
		t.Fatalf("期望默认编辑器为 %q，实际为 %q", "vim", got)
	}
}

func TestUpgradeCmdReturnsErrorWhenBinaryPathDoesNotMatchNPMGlobalInstall(t *testing.T) {
	originalLookup := lookupNPMGlobalBinDir
	lookupNPMGlobalBinDir = func() (string, error) {
		return filepath.Join(t.TempDir(), "bin"), nil
	}
	defer func() {
		lookupNPMGlobalBinDir = originalLookup
	}()

	status := detector.Status{
		Installed:  true,
		BinaryPath: "/Users/test/.opencode/bin/opencode",
		Tool: registry.Tool{
			Name:       "Example CLI",
			UpgradeCmd: []string{"npm", "install", "-g", "example-cli@latest"},
		},
	}

	cmd := UpgradeCmd(status, 0)
	msg := cmd()
	finished, ok := msg.(UpgradeFinishedMsg)
	if !ok {
		t.Fatalf("期望返回 UpgradeFinishedMsg，实际为 %T", msg)
	}

	var actionErr ActionError
	if !errors.As(finished.Err, &actionErr) {
		t.Fatalf("期望错误为 ActionError，实际为 %v", finished.Err)
	}
	if actionErr.Code != ErrorUpgradeTargetMismatch {
		t.Fatalf("期望错误码为 %q，实际为 %q", ErrorUpgradeTargetMismatch, actionErr.Code)
	}
	if actionErr.Arg != status.BinaryPath {
		t.Fatalf("期望错误参数为 %q，实际为 %q", status.BinaryPath, actionErr.Arg)
	}
	if actionErr.Hint != "npm_global" {
		t.Fatalf("期望错误提示为 %q，实际为 %q", "npm_global", actionErr.Hint)
	}
}

func TestValidateUpgradeTargetAllowsNPMGlobalInstallWhenBinaryPathMatches(t *testing.T) {
	tempDir := t.TempDir()
	globalBinDir := filepath.Join(tempDir, "bin")

	originalLookup := lookupNPMGlobalBinDir
	lookupNPMGlobalBinDir = func() (string, error) {
		return globalBinDir, nil
	}
	defer func() {
		lookupNPMGlobalBinDir = originalLookup
	}()

	status := detector.Status{
		Installed:  true,
		BinaryPath: filepath.Join(globalBinDir, "opencode"),
		Tool: registry.Tool{
			Name:       "Example CLI",
			UpgradeCmd: []string{"npm", "install", "-g", "example-cli@latest"},
		},
	}

	if err := validateUpgradeTarget(status, status.Tool.UpgradeCmd); err != nil {
		t.Fatalf("期望匹配 npm 全局安装路径时不报错，实际报错: %v", err)
	}
}

func TestResolveUpgradeCommandUsesBinaryUpgradeForOpenCode(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("期望能够读取用户目录，实际报错: %v", err)
	}

	status := detector.Status{
		Installed:     true,
		BinaryPath:    filepath.Join(homeDir, ".opencode", "bin", "opencode"),
		InstallSource: detector.InstallSourceOfficialScript,
		Tool: registry.Tool{
			Name:       "OpenCode",
			UpgradeCmd: []string{"npm", "install", "-g", "opencode-ai@latest"},
		},
	}

	commandArgs := resolveUpgradeCommand(status)
	expected := []string{filepath.Join(homeDir, ".opencode", "bin", "opencode"), "upgrade", "--method", "curl"}
	if len(commandArgs) != len(expected) {
		t.Fatalf("期望升级命令长度为 %d，实际为 %d，内容: %v", len(expected), len(commandArgs), commandArgs)
	}
	for index := range expected {
		if commandArgs[index] != expected[index] {
			t.Fatalf("期望第 %d 个参数为 %q，实际为 %q；完整命令: %v", index, expected[index], commandArgs[index], commandArgs)
		}
	}
}

func TestResolveUpgradeCommandUsesNPMMethodForOpenCodeInstalledViaNPM(t *testing.T) {
	tempDir := t.TempDir()
	globalBinDir := filepath.Join(tempDir, "bin")

	originalLookup := lookupNPMGlobalBinDir
	lookupNPMGlobalBinDir = func() (string, error) {
		return globalBinDir, nil
	}
	defer func() {
		lookupNPMGlobalBinDir = originalLookup
	}()

	status := detector.Status{
		Installed:     true,
		BinaryPath:    filepath.Join(globalBinDir, "opencode"),
		InstallSource: detector.InstallSourceNPMGlobal,
		Tool: registry.Tool{
			Name:       "OpenCode",
			UpgradeCmd: []string{"npm", "install", "-g", "opencode-ai@latest"},
		},
	}

	commandArgs := resolveUpgradeCommand(status)
	expected := []string{filepath.Join(globalBinDir, "opencode"), "upgrade", "--method", "npm"}
	if len(commandArgs) != len(expected) {
		t.Fatalf("期望升级命令长度为 %d，实际为 %d，内容: %v", len(expected), len(commandArgs), commandArgs)
	}
	for index := range expected {
		if commandArgs[index] != expected[index] {
			t.Fatalf("期望第 %d 个参数为 %q，实际为 %q；完整命令: %v", index, expected[index], commandArgs[index], commandArgs)
		}
	}
}

func TestResolveUpgradeCommandUsesCurrentBinaryForKimiCode(t *testing.T) {
	binaryPath := filepath.Join(t.TempDir(), "kimi")
	status := detector.Status{
		Installed:  true,
		BinaryPath: binaryPath,
		Tool: registry.Tool{
			Name:       "Kimi Code",
			UpgradeCmd: []string{"kimi", "upgrade"},
		},
	}

	commandArgs := resolveUpgradeCommand(status)
	expected := []string{binaryPath, "upgrade"}
	if len(commandArgs) != len(expected) {
		t.Fatalf("期望升级命令长度为 %d，实际为 %d，内容: %v", len(expected), len(commandArgs), commandArgs)
	}
	for index := range expected {
		if commandArgs[index] != expected[index] {
			t.Fatalf("期望第 %d 个参数为 %q，实际为 %q；完整命令: %v", index, expected[index], commandArgs[index], commandArgs)
		}
	}
}

func TestSupportsUpgradeActionReturnsTrueForAppBundleTool(t *testing.T) {
	tempDir := t.TempDir()
	appPath := filepath.Join(tempDir, "Cursor.app")
	if err := os.MkdirAll(appPath, 0o755); err != nil {
		t.Fatalf("期望创建应用目录成功，实际报错: %v", err)
	}

	status := detector.Status{
		Installed: true,
		Tool: registry.Tool{
			Name:       "Cursor",
			UpgradeCmd: []string{},
			CurrentVersion: registry.VersionSource{
				Provider: registry.CurrentVersionProviderAppBundle,
				Paths:    []string{appPath},
			},
		},
	}

	if !SupportsUpgradeAction(status) {
		t.Fatal("期望 app bundle 工具支持升级动作，实际返回不支持")
	}
}
