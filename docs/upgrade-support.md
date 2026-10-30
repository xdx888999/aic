# 升级支持核对

核对日期：2026-10-30。根据本机可执行文件、命令帮助及上游文档核对。本机未安装的工具没有进行实际升级验证。

## 本机已安装工具

| 工具 | 检测到的安装方式 | 升级入口 | 核对结果 |
| --- | --- | --- | --- |
| Claude Code | npm | `npm install -g @anthropic-ai/claude-code@latest` | npm 入口有效；原生安装使用 `claude update` |
| Codex CLI | npm | `npm install -g @openai/codex@latest` | npm 入口有效；本机命令帮助同时确认支持原生安装的 `codex update` |
| Kimi Code | 官方安装脚本 | 当前可执行文件的 `upgrade` 子命令 | 本机帮助与官方文档一致，升级器按安装渠道处理 |
| OpenCode | 官方安装脚本 | 当前可执行文件的 `upgrade <新版本> --method curl` | 2.x 更新源已改为 `@opencode/cli`；旧的 `opencode-ai` 继续发布 1.x |
| Grok Build | Homebrew cask `grok-build` | 所属 Homebrew 的 `upgrade --cask grok-build` | 新增注册项；最新版本也查询此 cask，避免自更新覆盖 Homebrew 管理的文件 |
| Qwen Code | npm | `npm install -g @qwen-code/qwen-code@latest` | npm 入口有效；原生安装使用本机帮助列出的 `qwen update` |

npm 命令还会显式指定 `https://registry.npmjs.org`，并在执行前验证当前 npm 全局目录。核对过程不批量升级这些工具。

## 安装渠道处理

- 沿可执行文件的符号链接识别 Homebrew `Cellar` / `Caskroom`，优先于 `node_modules` 判断。升级使用该安装目录里的 `brew`，避免混用两套 Homebrew。
- Homebrew Core / Cask 使用对应公共 API 查询版本。第三方 formula 从 `INSTALL_RECEIPT.json` 保留 tap 名称，再通过所属 `brew info --json=v2` 查询。第三方 tap 的查询结果取决于本地 tap 元数据是否已更新。
- OpenCode 非 Homebrew 安装根据当前版本选择 1.x / 2.x 发布源，不主动迁移旧版。读取不到当前版本时不猜测发布系列。Homebrew 安装则跟随已安装的 formula。
- OpenCode 保留自身的升级器，并只把比当前版本新的检测结果作为目标版本。官方脚本安装指定 `curl`，npm 安装指定 `npm`；其他安装方式交由其升级器识别。
- Grok 原生安装使用 `grok update`，npm 安装使用官方包 `@xai-official/grok`，Homebrew 安装使用 cask。
- Claude Code、Codex CLI、Qwen Code、Kilo Code 的 npm 与原生升级入口分别配置。Kilo 的 `kilo upgrade` 已按官方文档核对，但本机未安装，未实测。
- Gemini 从实际命中的可执行文件寻找所属包的版本信息，找不到时运行该文件的 `--version`，不再读取另一套 npm 的全局安装记录。
- npm 前缀查询失败或当前可执行文件不属于该前缀时取消升级。桌面应用继续使用原有的应用内升级入口。

## 验证范围

- 自动测试覆盖 OpenCode 1.x / 2.x 分流、Homebrew Grok 检测、第三方 tap、升级命令选择、npm 来源不匹配、更新源失败和 stderr 警告干扰。
- 本机界面扫描能显示全部 6 个已安装工具；OpenCode 从原本的错误 1.x 更新源恢复为 2.x，Grok 显示 Homebrew 更新源。
- 已实际执行修复版生成的 OpenCode 升级命令，将本机从 2.1.3 升至 2.2.2；命令成功退出，`opencode --version` 返回 `opencode v2.2.2`。其他工具仅核对，没有执行升级。
- 升级后在修复版界面选中 OpenCode 按 `u`，确认能调用升级器、识别已经是最新版，并自动返回列表重新扫描。
- 其余未安装条目保留已有配置，不据此宣称所有安装方式均已实测。

## 依据

- [OpenCode 2 安装文档](https://opencode.ai/v2/docs)
- [OpenCode 1 升级文档](https://opencode.ai/docs/cli/#upgrade)
- [OpenCode 升级器源码](https://github.com/anomalyco/opencode/blob/dev/packages/cli/src/services/updater.ts)
- [Grok CLI 命令参考](https://docs.x.ai/build/cli/reference)
- [Homebrew Grok Build cask](https://formulae.brew.sh/cask/grok-build)
- [Claude Code 升级说明](https://code.claude.com/docs/en/setup#update-manually)
- [Kimi Code 升级说明](https://www.kimi.com/code/docs/en/kimi-code-cli/reference/kimi-command)
- [Qwen Code 升级说明](https://qwenlm.github.io/qwen-code-docs/en/users/support/troubleshooting/)
- [Kilo CLI 升级说明](https://kilo.ai/docs/code-with-ai/platforms/cli)
