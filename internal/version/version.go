package version

import "fmt"

const (
	// 这些默认值会在正式构建时通过 ldflags 覆盖，便于发行版输出精确版本信息。
	defaultVersion   = "dev"
	defaultCommit    = "unknown"
	defaultBuildTime = "unknown"
)

var (
	Version   = defaultVersion
	Commit    = defaultCommit
	BuildTime = defaultBuildTime
)

// String 返回统一的版本输出格式，便于命令行和自动化脚本解析。
func String() string {
	return fmt.Sprintf("aic %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}
