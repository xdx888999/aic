package actions

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/registry"
)

type ErrorCode string

const (
	ErrorUpgradeNotSupported   ErrorCode = "upgrade_not_supported"
	ErrorMissingPackageManager ErrorCode = "missing_package_manager"
	ErrorUpgradeTargetMismatch ErrorCode = "upgrade_target_mismatch"
)

type ActionError struct {
	Code ErrorCode
	Arg  string
	Hint string
}

func (e ActionError) Error() string {
	if e.Hint != "" && e.Arg != "" {
		return string(e.Code) + ":" + e.Arg + ":" + e.Hint
	}
	if e.Arg == "" {
		return string(e.Code)
	}
	return string(e.Code) + ":" + e.Arg
}

type UpgradeFinishedMsg struct {
	Index int
	Err   error
}

type ConfigClosedMsg struct {
	Err error
}

type ManualUpgradeLaunchedMsg struct {
	ToolName string
}

var lookupNPMGlobalBinDir = detectNPMGlobalBinDir

const (
	openCodeToolName = "OpenCode"
	kimiCodeToolName = "Kimi Code"
)

func UpgradeCmd(status detector.Status, index int) tea.Cmd {
	commandArgs := resolveUpgradeCommand(status)
	if len(commandArgs) == 0 {
		if appPath := resolveManualUpgradeAppPath(status.Tool); appPath != "" {
			c := exec.Command("open", appPath)
			return tea.ExecProcess(c, func(err error) tea.Msg {
				if err != nil {
					return UpgradeFinishedMsg{Index: index, Err: err}
				}
				return ManualUpgradeLaunchedMsg{ToolName: status.Tool.Name}
			})
		}
		return func() tea.Msg {
			return UpgradeFinishedMsg{Index: index, Err: ActionError{Code: ErrorUpgradeNotSupported}}
		}
	}

	pkgMgr := commandArgs[0]
	if _, err := exec.LookPath(pkgMgr); err != nil {
		return func() tea.Msg {
			return UpgradeFinishedMsg{Index: index, Err: ActionError{
				Code: ErrorMissingPackageManager,
				Arg:  pkgMgr,
			}}
		}
	}

	if err := validateUpgradeTarget(status, commandArgs); err != nil {
		return func() tea.Msg {
			return UpgradeFinishedMsg{Index: index, Err: err}
		}
	}

	c := exec.Command(commandArgs[0], commandArgs[1:]...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return UpgradeFinishedMsg{Index: index, Err: err}
	})
}

func resolveUpgradeCommand(status detector.Status) []string {
	switch status.Tool.Name {
	case openCodeToolName:
		return resolveOpenCodeUpgradeCommand(status)
	case kimiCodeToolName:
		return resolveKimiCodeUpgradeCommand(status)
	default:
		return append([]string(nil), status.Tool.UpgradeCmd...)
	}
}

func SupportsUpgradeAction(status detector.Status) bool {
	if !status.Installed {
		return false
	}
	if len(resolveUpgradeCommand(status)) > 0 {
		return true
	}
	return resolveManualUpgradeAppPath(status.Tool) != ""
}

func resolveOpenCodeUpgradeCommand(status detector.Status) []string {
	if !status.Installed || status.BinaryPath == "" {
		return append([]string(nil), status.Tool.UpgradeCmd...)
	}

	commandArgs := []string{status.BinaryPath, "upgrade"}
	if method := inferOpenCodeUpgradeMethod(status.InstallSource); method != "" {
		commandArgs = append(commandArgs, "--method", method)
	}
	return commandArgs
}

func inferOpenCodeUpgradeMethod(source detector.InstallSource) string {
	switch source {
	case detector.InstallSourceNPMGlobal:
		return "npm"
	case detector.InstallSourceOfficialScript:
		return "curl"
	}
	return ""
}

func resolveKimiCodeUpgradeCommand(status detector.Status) []string {
	if status.Installed && status.BinaryPath != "" {
		return []string{status.BinaryPath, "upgrade"}
	}
	return append([]string(nil), status.Tool.UpgradeCmd...)
}

func resolveManualUpgradeAppPath(tool registry.Tool) string {
	if tool.CurrentVersion.Provider != registry.CurrentVersionProviderAppBundle {
		return ""
	}

	for _, rawPath := range tool.CurrentVersion.Paths {
		appPath := expandHome(rawPath)
		if _, err := os.Stat(appPath); err == nil {
			return appPath
		}
	}
	return ""
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

func validateUpgradeTarget(status detector.Status, commandArgs []string) error {
	if !status.Installed || status.BinaryPath == "" || len(commandArgs) == 0 {
		return nil
	}

	switch commandArgs[0] {
	case "npm":
		globalBinDir, err := lookupNPMGlobalBinDir()
		if err != nil || globalBinDir == "" {
			return nil
		}
		if isPathWithinDir(status.BinaryPath, globalBinDir) {
			return nil
		}
		return ActionError{
			Code: ErrorUpgradeTargetMismatch,
			Arg:  status.BinaryPath,
			Hint: "npm_global",
		}
	default:
		return nil
	}
}

func detectNPMGlobalBinDir() (string, error) {
	command := exec.Command("npm", "prefix", "-g")
	output, err := command.Output()
	if err != nil {
		return "", err
	}

	prefix := strings.TrimSpace(string(output))
	if prefix == "" {
		return "", nil
	}

	if runtime.GOOS == "windows" {
		return prefix, nil
	}
	return filepath.Join(prefix, "bin"), nil
}

func isPathWithinDir(path string, dir string) bool {
	if path == "" || dir == "" {
		return false
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}

	relativePath, err := filepath.Rel(absoluteDir, absolutePath)
	if err != nil {
		return false
	}

	return relativePath == "." || (relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)))
}

func ResolveEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vim"
}

func OpenConfigCmd(configPath string) tea.Cmd {
	c := exec.Command(ResolveEditor(), configPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return ConfigClosedMsg{Err: err}
	})
}
