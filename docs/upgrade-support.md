# 升级支持核对

核对日期：2026-10-30。根据本机可执行文件、命令帮助及上游文档核对。本机未安装的工具没有进行实际升级验证。

## 本机已安装工具

| 工具 | 检测到的安装方式 | 升级入口 | 核对结果 |
| --- | --- | --- | --- |
| Claude Code | npm | `npm install -g @anthropic-ai/claude-code@latest` | npm 入口有效；原生安装使用 `claude update` |
| Codex CLI | npm | `npm install -g @openai/codex@latest` | npm 入口有效；本机命令帮助同时确认支持原生安装的 `codex update` |
| Kimi Code | 官方安装脚本 | 当前可执行文件的 `upgrade` 子命令 | 本机帮助与官方文档一致，升级器按安装渠道处理 |
| OpenCode | 官方安装脚本 | 同系列使用 `upgrade <新版本> --method curl`；1.x 到 2.x 使用 `upgrade --major --method curl` | 最新版本查询 `@opencode/cli`，不再将旧包 `opencode-ai` 的末版当成产品最新版 |
| Grok Build | Homebrew cask `grok-build` | 所属 Homebrew 的 `upgrade --cask grok-build` | 新增注册项；最新版本也查询此 cask，避免自更新覆盖 Homebrew 管理的文件 |
| Qwen Code | npm | `npm install -g @qwen-code/qwen-code@latest` | npm 入口有效；原生安装使用本机帮助列出的 `qwen update` |

npm 命令还会显式指定 `https://registry.npmjs.org`，并在执行前验证当前 npm 全局目录。核对过程不批量升级这些工具。

## 安装渠道处理

- 沿可执行文件的符号链接识别 Homebrew `Cellar` / `Caskroom`，优先于 `node_modules` 判断。升级使用该安装目录里的 `brew`，避免混用两套 Homebrew。
- Homebrew Core / Cask 使用对应公共 API 查询版本。第三方 formula 从 `INSTALL_RECEIPT.json` 保留 tap 名称，再通过所属 `brew info --json=v2` 查询。第三方 tap 的查询结果取决于本地 tap 元数据是否已更新。
- OpenCode 非 Homebrew 安装统一查询 `@opencode/cli`，1.x 也能显示 2.x 更新。读取不到当前版本时仍可显示最新发行版，但不据此推断应执行跨版本迁移。Homebrew Core 跟随 `opencode`；官方旧 tap 的 `anomalyco/tap/opencode` 查询并迁移到 `anomalyco/tap/opencode-v2`。
- OpenCode 1.x 到 2.x 使用自身的 `upgrade --major`。较早版本不支持此参数时，先执行普通升级更新旧版升级器，再重新检测迁移能力；帮助信息同时读取 stdout 和 stderr。官方脚本安装指定 `curl`，npm 安装指定 `npm`，官方旧 tap 指定 `brew` 并保留所属 Homebrew 前缀；其他安装方式交由其升级器识别。
- OpenCode 跨版本升级后重新读取实际版本，确认已经安装 2.x 后才报告成功。旧版升级器可能失败却退出 0，也可能安装完成后因终端错误返回非零退出码，不能只凭退出码判断。2.x 的常规升级继续只传入比当前版本更新的目标版本。
- Grok 原生安装使用 `grok update`，npm 安装使用官方包 `@xai-official/grok`，Homebrew 安装使用 cask。
- Claude Code、Codex CLI、Qwen Code、Kilo Code 的 npm 与原生升级入口分别配置。Kilo 的 `kilo upgrade` 已按官方文档核对，但本机未安装，未实测。
- Gemini 从实际命中的可执行文件寻找所属包的版本信息，找不到时运行该文件的 `--version`，不再读取另一套 npm 的全局安装记录。
- npm 前缀查询失败或当前可执行文件不属于该前缀时取消升级。桌面应用继续使用原有的应用内升级入口。

## 验证范围

- 自动测试覆盖 OpenCode 1.x 检测 2.x、旧升级器引导更新、stdout / stderr 帮助信息、退出码与实际版本不一致、Homebrew Core 与官方旧 tap 的不同升级路径，以及 Homebrew Grok 检测、npm 来源不匹配、更新源失败和 stderr 警告干扰。
- 本机界面扫描能显示全部 6 个已安装工具；OpenCode 从原本的错误 1.x 更新源恢复为 2.x，Grok 显示 Homebrew 更新源。
- 已实际执行修复版生成的 OpenCode 升级命令，将本机从 2.1.3 升至 2.2.2；命令成功退出，`opencode --version` 返回 `opencode v2.2.2`。其他工具仅核对，没有执行升级。
- 升级后在修复版界面选中 OpenCode 按 `u`，确认能调用升级器、识别已经是最新版，并自动返回列表重新扫描。
- v0.1.7 在 `/tmp` 的独立 npm 前缀中安装真实 OpenCode 1.19.5，通过 AIC 界面按 `u` 完成到 2.2.2 的迁移；界面显示升级完成并自动刷新为当前版本与最新版本均为 2.2.2。旧升级器在此终端环境出现 `EINVAL: invalid argument, kqueue` 并返回非零退出码，安装仍完成，AIC 通过实际版本复核确认结果。
- 更早 1.x 的升级器更新、官方脚本跨版本迁移和 Homebrew formula 迁移由自动测试覆盖，本次未逐一执行真实安装迁移。
- 其余未安装条目保留已有配置，不据此宣称所有安装方式均已实测。

## 依据

- [OpenCode 2 安装文档](https://opencode.ai/v2/docs)
- [OpenCode 1 升级文档](https://opencode.ai/docs/cli/#upgrade)
- [OpenCode 1.19.5 跨版本升级入口](https://github.com/anomalyco/opencode/blob/v1.19.5/packages/opencode/src/cli/cmd/upgrade.ts)
- [OpenCode 升级器源码](https://github.com/anomalyco/opencode/blob/dev/packages/cli/src/services/updater.ts)
- [Grok CLI 命令参考](https://docs.x.ai/build/cli/reference)
- [Homebrew Grok Build cask](https://formulae.brew.sh/cask/grok-build)
- [Claude Code 升级说明](https://code.claude.com/docs/en/setup#update-manually)
- [Kimi Code 升级说明](https://www.kimi.com/code/docs/en/kimi-code-cli/reference/kimi-command)
- [Qwen Code 升级说明](https://qwenlm.github.io/qwen-code-docs/en/users/support/troubleshooting/)
- [Kilo CLI 升级说明](https://kilo.ai/docs/code-with-ai/platforms/cli)
