package actions

import (
	"errors"
	"os"
	"testing"

	"github.com/xdx888999/aic/internal/registry"
)

func TestUpgradeCmdReturnsErrorWhenNoCmdConfigured(t *testing.T) {
	tool := registry.Tool{Name: "NoCmd", UpgradeCmd: []string{}}
	cmd := UpgradeCmd(tool, 0)

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
	tool := registry.Tool{
		Name:       "MissingPkgMgr",
		UpgradeCmd: []string{"__nonexistent_pkg_manager__", "install", "something"},
	}
	cmd := UpgradeCmd(tool, 0)

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
