package actions

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/registry"
)

func needsOpenCodeMajorUpgrade(status detector.Status) bool {
	if status.Tool.Name != openCodeToolName || !status.Installed || status.BinaryPath == "" {
		return false
	}
	if status.InstallSource == detector.InstallSourceHomebrewFormula {
		// Core 的 opencode 已直接发布 2.x；官方旧 tap 则必须更换 formula。
		if status.HomebrewPackage != "anomalyco/tap/opencode" {
			return false
		}
	}
	current, latest := openCodeMajor(status.Version), openCodeMajor(status.LatestVersion)
	return current >= 0 && current < 2 && latest >= 2
}

func openCodeMajor(version string) int {
	major, err := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
	if err != nil {
		return -1
	}
	return major
}

type openCodeMajorUpgrade struct {
	status detector.Status
	args   []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (c *openCodeMajorUpgrade) SetStdin(r io.Reader)  { c.stdin = r }
func (c *openCodeMajorUpgrade) SetStdout(w io.Writer) { c.stdout = w }
func (c *openCodeMajorUpgrade) SetStderr(w io.Writer) { c.stderr = w }

func (c *openCodeMajorUpgrade) binaryPath() string {
	if c.status.InstallSource == detector.InstallSourceHomebrewFormula && c.status.HomebrewPrefix != "" {
		// formula 升级会更换 Cellar 目录，始终通过所属前缀的稳定入口再次调用。
		return filepath.Join(c.status.HomebrewPrefix, "bin", "opencode")
	}
	return c.status.BinaryPath
}

func (c *openCodeMajorUpgrade) env() []string {
	env := os.Environ()
	if c.status.HomebrewPrefix != "" {
		env = append(env, "PATH="+filepath.Join(c.status.HomebrewPrefix, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return env
}

func (c *openCodeMajorUpgrade) output(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, c.binaryPath(), args...)
	command.Env = c.env()
	output, err := command.Output()
	return strings.TrimSpace(string(output)), err
}

func (c *openCodeMajorUpgrade) supportsMajorUpgrade() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, c.binaryPath(), "upgrade", "--help")
	command.Env = c.env()
	// 1.x 将帮助信息写入 stderr；版本读取仍只取 stdout，避免误读告警中的版本号。
	output, err := command.CombinedOutput()
	return strings.Contains(string(output), "--major"), err
}

func (c *openCodeMajorUpgrade) run(args ...string) error {
	command := exec.Command(c.binaryPath(), args...)
	command.Env = c.env()
	command.Stdin, command.Stdout, command.Stderr = c.stdin, c.stdout, c.stderr
	return command.Run()
}

func (c *openCodeMajorUpgrade) version() (string, error) {
	output, err := c.output("--version")
	return registry.ParseVersion(output, "semver"), err
}

func (c *openCodeMajorUpgrade) incomplete() error {
	version, _ := c.version()
	if version == "" {
		version = "?"
	}
	return ActionError{Code: ErrorOpenCodeMigrationIncomplete, Arg: version}
}

func (c *openCodeMajorUpgrade) Run() error {
	supported, err := c.supportsMajorUpgrade()
	if err != nil {
		return err
	}
	if !supported {
		// 较早的 1.x 没有迁移命令，先更新旧版升级器，再重新检测能力。
		args := []string{"upgrade"}
		if method := inferOpenCodeUpgradeMethod(c.status.InstallSource); method != "" {
			args = append(args, "--method", method)
		}
		if err := c.run(args...); err != nil {
			return err
		}
		if version, err := c.version(); err == nil && openCodeMajor(version) >= 2 {
			return nil
		}
		supported, err = c.supportsMajorUpgrade()
		if err != nil || !supported {
			return c.incomplete()
		}
	}
	err = c.run(c.args[1:]...)
	// 1.x 升级器可能失败却退出 0，也可能安装完成后发生终端错误；以实际版本为准。
	if version, err := c.version(); err == nil && openCodeMajor(version) >= 2 {
		return nil
	}
	if err != nil {
		return err
	}
	return c.incomplete()
}
