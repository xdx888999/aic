package version

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

// String 返回简洁稳定的版本输出，便于用户查看和脚本解析。
func String() string {
	return "aic " + Version
}
