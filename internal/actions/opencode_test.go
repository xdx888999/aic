package actions

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/registry"
)

const migrationFixture = `#!/bin/sh
case "$1 $2" in
  '--version ')
    printf 'Warning: test plugin 9.0.0 is outdated\n' >&2
    /bin/cat "$AIC_MIGRATION_TEST_DIR/version"
    exit 0
    ;;
  'upgrade --help')
    if [ "$AIC_MIGRATION_TEST_MODE" != "stdout-help" ]; then
      exec 1>&2
    fi
    version=$(/bin/cat "$AIC_MIGRATION_TEST_DIR/version")
    if [ "$version" = "1.18.0" ]; then
      printf 'upgrade [target]\n'
    else
      printf '%s\n' 'upgrade [target] --major'
    fi
    exit 0
    ;;
esac
printf '%s\n' "$*" >> "$AIC_MIGRATION_TEST_DIR/commands"
if [ "$AIC_MIGRATION_TEST_MODE" = "bootstrap-fails" ]; then
  printf 'upstream update failed\n' >&2
  exit 7
fi
if [ "$AIC_MIGRATION_TEST_MODE" = "silent-failure" ]; then
  printf 'Upgrade failed\n' >&2
  exit 0
fi
if [ "$2" = "--major" ]; then
  printf '2.2.2\n' > "$AIC_MIGRATION_TEST_DIR/version"
  printf 'OpenCode 2 installed\n'
  command -v brew > "$AIC_MIGRATION_TEST_DIR/brew-path" || :
  if [ "$AIC_MIGRATION_TEST_MODE" = "installed-with-error" ]; then
    printf 'terminal cleanup failed\n' >&2
    exit 1
  fi
else
  printf '1.19.5\n' > "$AIC_MIGRATION_TEST_DIR/version"
fi
`

func migrationCommand(t *testing.T, version, mode string) (*openCodeMajorUpgrade, string, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("AIC_MIGRATION_TEST_DIR", dir)
	t.Setenv("AIC_MIGRATION_TEST_MODE", mode)
	if err := os.WriteFile(filepath.Join(dir, "version"), []byte(version), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "opencode")
	if err := os.WriteFile(binary, []byte(migrationFixture), 0o755); err != nil {
		t.Fatal(err)
	}
	status := detector.Status{
		Tool:      registry.Tool{Name: "OpenCode", Binary: "opencode"},
		Installed: true, BinaryPath: binary, Version: version, LatestVersion: "2.2.2",
		InstallSource: detector.InstallSourceOfficialScript,
	}
	command := &openCodeMajorUpgrade{status: status, args: resolveUpgradeCommand(status)}
	var output bytes.Buffer
	command.SetStdin(strings.NewReader(""))
	command.SetStdout(&output)
	command.SetStderr(&output)
	return command, dir, &output
}

func TestOpenCodeMigrationUpdatesOlderInstallerWhenNeeded(t *testing.T) {
	for _, tc := range []struct{ name, version, mode, commands string }{
		{"help on stderr", "1.19.5", "success", "upgrade --major --method curl\n"},
		{"help on stdout", "1.19.5", "stdout-help", "upgrade --major --method curl\n"},
		{"older installer", "1.18.0", "success", "upgrade --method curl\nupgrade --major --method curl\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command, dir, output := migrationCommand(t, tc.version, tc.mode)
			if err := command.Run(); err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(filepath.Join(dir, "commands"))
			if err != nil || string(actual) != tc.commands {
				t.Fatalf("迁移步骤错误: %q, %v", actual, err)
			}
			version, err := command.version()
			if err != nil || version != "2.2.2" || !strings.Contains(output.String(), "OpenCode 2 installed") {
				t.Fatalf("升级结果或输出未正确传递: %q, %q, %v", version, output.String(), err)
			}
		})
	}
}

func TestOpenCodeMigrationRejectsSuccessfulExitWithoutVersionChange(t *testing.T) {
	for _, version := range []string{"1.19.5", "1.18.0"} {
		t.Run(version, func(t *testing.T) {
			command, _, _ := migrationCommand(t, version, "silent-failure")
			var actionErr ActionError
			if err := command.Run(); !errors.As(err, &actionErr) || actionErr.Code != ErrorOpenCodeMigrationIncomplete || actionErr.Arg != version {
				t.Fatalf("退出码为 0 但版本未变，不能报告迁移成功: %v", err)
			}
		})
	}
}

func TestOpenCodeMigrationVerifiesVersionAfterUpdaterError(t *testing.T) {
	command, _, _ := migrationCommand(t, "1.19.5", "installed-with-error")
	if err := command.Run(); err != nil {
		t.Fatalf("实际已安装 2.x，不应因旧升级器的终端错误误报迁移失败: %v", err)
	}
}

func TestOpenCodeMigrationStopsAfterBootstrapFailure(t *testing.T) {
	command, dir, _ := migrationCommand(t, "1.18.0", "bootstrap-fails")
	if err := command.Run(); err == nil {
		t.Fatal("更新旧升级器失败时必须返回错误")
	}
	actual, err := os.ReadFile(filepath.Join(dir, "commands"))
	if err != nil || string(actual) != "upgrade --method curl\n" {
		t.Fatalf("失败后不应继续执行迁移: %q, %v", actual, err)
	}
}

func TestOpenCodeMigrationKeepsHomebrewPrefixAfterFormulaReplacement(t *testing.T) {
	command, dir, _ := migrationCommand(t, "1.19.5", "success")
	prefix := filepath.Join(dir, "brew-prefix")
	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "opencode"), []byte(migrationFixture), 0o755); err != nil {
		t.Fatal(err)
	}
	brewPath := filepath.Join(binDir, "brew")
	if err := os.WriteFile(brewPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	command.status.BinaryPath = filepath.Join(prefix, "Cellar", "opencode", "1.19.5", "bin", "opencode")
	command.status.HomebrewPrefix = prefix
	command.status.HomebrewPackage = "anomalyco/tap/opencode"
	command.status.InstallSource = detector.InstallSourceHomebrewFormula
	command.args = resolveUpgradeCommand(command.status)
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(filepath.Join(dir, "brew-path"))
	if err != nil || strings.TrimSpace(string(actual)) != brewPath {
		t.Fatalf("迁移没有使用安装所属的 Homebrew: %q, %v", actual, err)
	}
}

func TestOpenCodeMajorUpgradeCommandMatchesInstallationSource(t *testing.T) {
	for _, tc := range []struct {
		name        string
		source      detector.InstallSource
		packageName string
		args        []string
	}{
		{"official script", detector.InstallSourceOfficialScript, "", []string{"upgrade", "--major", "--method", "curl"}},
		{"npm", detector.InstallSourceNPMGlobal, "", []string{"upgrade", "--major", "--method", "npm"}},
		{"unknown", detector.InstallSourceUnknown, "", []string{"upgrade", "--major"}},
		{"legacy tap", detector.InstallSourceHomebrewFormula, "anomalyco/tap/opencode", []string{"upgrade", "--major", "--method", "brew"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := detector.Status{
				Tool:      registry.Tool{Name: "OpenCode", UpgradeCmd: []string{"opencode", "upgrade"}},
				Installed: true, BinaryPath: "/example/bin/opencode", Version: "1.19.5", LatestVersion: "2.2.2",
				InstallSource: tc.source, HomebrewPackage: tc.packageName, HomebrewPrefix: "/example",
			}
			expected := append([]string{status.BinaryPath}, tc.args...)
			if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, expected) {
				t.Fatalf("迁移命令为 %v，期望 %v", actual, expected)
			}
		})
	}
}

func TestOpenCodeHomebrewCoreUpgradeKeepsExistingFormula(t *testing.T) {
	status := detector.Status{
		Tool:      registry.Tool{Name: "OpenCode", UpgradeCmd: []string{"opencode", "upgrade"}},
		Installed: true, BinaryPath: "/example/bin/opencode", Version: "1.19.5", LatestVersion: "2.2.1",
		InstallSource: detector.InstallSourceHomebrewFormula, HomebrewPackage: "opencode", HomebrewPrefix: "/example",
	}
	if needsOpenCodeMajorUpgrade(status) {
		t.Fatal("Homebrew Core 已经发布 2.x，不应切换到第三方 tap")
	}
	expected := []string{"/example/bin/brew", "upgrade", "--formula", "opencode"}
	if actual := resolveUpgradeCommand(status); !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Homebrew Core 升级命令错误: %v", actual)
	}
}
