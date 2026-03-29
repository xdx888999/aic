# 更新日志

本项目的显著变更会记录在这里。

建议后续发布时遵循以下约定：

- 按版本维护变更记录
- 未发布内容先写在 `Unreleased`
- 正式发版时再归档到具体版本

## [Unreleased]

## [v0.1.3] - 2026-03-30

### Changed

- `OpenCode` 升级逻辑改为根据当前命中的安装来源自动选择正确升级方式，不再固定走 npm
- `Kimi CLI` 升级逻辑改为根据当前命中的安装来源自动选择对应升级入口，支持区分 Conda 与 uv tool
- `Cursor`、`Trae Agent`、`Windsurf` 这类桌面应用现在支持通过 `u` 键直接打开软件，引导用户在应用内完成升级
- `Trae Agent` 在“最新版本 / 更新源”列改为明确提示“按 u 手检 / 官方无接口”，避免误解为检测异常

### Fixed

- 修复 `aic` 在升级 `OpenCode` 时可能重新安装 npm 版本、导致本机出现重复安装的问题
- 修复 `aic` 在升级 `Kimi CLI` 时可能升级到错误 Python 工具链的问题
- 修复桌面类工具虽然支持手动升级，但界面中不显示升级动作的问题

### CI

- GitHub Actions 升级到 Node 24 兼容链路，`actions/checkout` 升级到 `v5`，`actions/setup-go` 升级到 `v6`
- 发布流程改为直接安装并执行 `goreleaser v2.12.7`，去除对旧版 JavaScript action runtime 的依赖

## [v0.1.1] - 2026-03-27

### Added

- 新增 `aic --version`，支持输出版本号、提交哈希和构建时间
- 新增 `Makefile`，统一 `build`、`test`、`run`、`clean` 和 `release-dry-run`
- 新增 GitHub Actions 测试工作流
- 新增 GitHub Actions 发布工作流
- 新增 GoReleaser 配置，为 GitHub Releases 做准备
- README 新增从 GitHub Release 下载安装的说明
- GoReleaser 新增默认发行说明模板，用于后续自动生成更完整的 Release 正文
- README 删除 npm 安装计划，只保留 GitHub Release 和 Homebrew 方向
- README 将 Homebrew 调整为已支持的正式安装方式
- GoReleaser 新增 `brews` 配置，为后续自动更新 `homebrew-tap` 做准备

### Changed

- `aic --version` 输出改为更简洁的 `aic <version>` 形式
- `aic -version` 现在也会正确输出版本信息，不再误进入主程序
- README 新增 `aic` 自身更新说明，明确区分 Homebrew 用户与手动下载用户的更新方式

## [v0.1.0] - 2026-03-27

### Added

- 初始公开版本
- 支持扫描本机常见 AI CLI 工具
- 支持显示安装状态、当前版本、最新版本和更新源
- 支持默认隐藏未安装工具，并通过快捷键切换显示全部候选工具
- 支持在界面中一键升级已检测到的工具
- 支持打开工具配置文件
- 支持升级完成后自动重新扫描并刷新界面
- 支持在当前版本后显示可升级箭头提示
- 支持真实截图和基础项目文档

### Changed

- 将工具注册表从硬编码调整为配置文件驱动
- 优化表格列宽和对齐方式，修复操作列错位问题
- 调整选中态颜色，提高可读性
- 将“版本来源”文案调整为“更新源”，降低歧义
- 将操作列文案压缩为紧凑形式，避免窄终端下被截断

### Fixed

- 修复 Gemini CLI 当前版本与最新版本检测不准确的问题
- 修复 Gemini 更新源样式显示异常的问题
- 修复升级成功后界面未自动刷新内容的问题
- 修复配置帮助页中 Enter / Esc 提示不明显的问题
- 修复英文界面下操作列内容显示不全的问题
- 修复桌面类工具配置路径和版本 provider 的部分缺失问题
