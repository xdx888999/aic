package actions

import (
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdx888999/aic/internal/registry"
)

type ErrorCode string

const (
	ErrorUpgradeNotSupported   ErrorCode = "upgrade_not_supported"
	ErrorMissingPackageManager ErrorCode = "missing_package_manager"
)

type ActionError struct {
	Code ErrorCode
	Arg  string
}

func (e ActionError) Error() string {
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

func UpgradeCmd(tool registry.Tool, index int) tea.Cmd {
	if len(tool.UpgradeCmd) == 0 {
		return func() tea.Msg {
			return UpgradeFinishedMsg{Index: index, Err: ActionError{Code: ErrorUpgradeNotSupported}}
		}
	}

	pkgMgr := tool.UpgradeCmd[0]
	if _, err := exec.LookPath(pkgMgr); err != nil {
		return func() tea.Msg {
			return UpgradeFinishedMsg{Index: index, Err: ActionError{
				Code: ErrorMissingPackageManager,
				Arg:  pkgMgr,
			}}
		}
	}

	c := exec.Command(tool.UpgradeCmd[0], tool.UpgradeCmd[1:]...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return UpgradeFinishedMsg{Index: index, Err: err}
	})
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
