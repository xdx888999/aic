package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Locale string

const (
	LocaleEnglish Locale = "en"
	LocaleChinese Locale = "zh"
)

const (
	envAICLang   = "AIC_LANG"
	envLCAll     = "LC_ALL"
	envLCMessage = "LC_MESSAGES"
	envLANG      = "LANG"
)

type Localizer struct {
	locale Locale
}

func DetectLocale() Locale {
	return DetectLocaleFromLookup(os.Getenv)
}

func DefaultLocale() Locale {
	override := strings.TrimSpace(strings.ToLower(os.Getenv(envAICLang)))
	switch override {
	case "zh", "zh-cn", "zh_cn":
		return LocaleChinese
	case "en", "en-us", "en_us":
		return LocaleEnglish
	default:
		return LocaleEnglish
	}
}

func DetectLocaleFromLookup(lookup func(string) string) Locale {
	override := strings.TrimSpace(strings.ToLower(lookup(envAICLang)))
	switch override {
	case "zh", "zh-cn", "zh_cn":
		return LocaleChinese
	case "en", "en-us", "en_us":
		return LocaleEnglish
	}

	for _, key := range []string{envLCAll, envLCMessage, envLANG} {
		value := strings.TrimSpace(strings.ToLower(lookup(key)))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "zh") {
			return LocaleChinese
		}
	}
	return LocaleEnglish
}

func NewLocalizer(locale Locale) Localizer {
	if locale != LocaleChinese {
		locale = LocaleEnglish
	}
	return Localizer{locale: locale}
}

func (l Localizer) Locale() Locale {
	return l.locale
}

func (l Localizer) Text(key string, args ...any) string {
	template := translations[l.locale][key]
	if template == "" {
		template = translations[LocaleEnglish][key]
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

var translations = map[Locale]map[string]string{
	LocaleChinese: {
		"stderr_error":                  "错误: %v\n",
		"loading_detecting_tools":       "正在检测已安装的工具...",
		"scan_complete":                 "扫描完成",
		"upgrade_failed":                "升级失败: %v",
		"upgrade_complete_rescanning":   "升级完成，正在重新检测...",
		"manual_upgrade_opened":         "已打开 %s，请在软件内完成升级",
		"config_closed_error":           "编辑器退出异常: %v",
		"title":                         "aic — AI CLI 工具管理器",
		"list_help":                     "  [a] 显示/隐藏未安装  [u] 升级  [c] 配置  [l] 切换语言  [r] 重新扫描  [q] 退出",
		"list_footer":                   "  使用 ↑↓/jk 移动光标",
		"language_badge":                "中文",
		"language_switched_zh":          "已切换到中文",
		"language_switched_en":          "Switched to English",
		"summary_visible":               "  已显示 %d / %d",
		"summary_hidden_hint":           "，已隐藏 %d 个未安装工具，按 [a] 查看全部",
		"empty_tools":                   "  当前未检测到已安装的 AI CLI 工具",
		"empty_tools_hint_show_all":     "  按 [a] 查看全部候选工具，按 [r] 重新扫描",
		"empty_tools_hint_rescan":       "  按 [r] 重新扫描",
		"column_tool":                   "工具",
		"column_status":                 "状态",
		"column_current_version":        "当前版本",
		"column_latest_version":         "最新版本",
		"column_update_source":          "更新源",
		"column_actions":                "操作",
		"status_installed":              "已安装",
		"status_missing":                "未安装",
		"action_upgrade":                "u升级",
		"action_config":                 "c配置",
		"source_none":                   "无上游",
		"source_manual_check":           "官方无接口",
		"latest_manual_check":           "按u手检",
		"config_title":                  "编辑配置 — %s",
		"config_file":                   "配置文件",
		"config_editor":                 "编辑器: %s",
		"config_common_actions":         "常用操作",
		"config_editor_reference":       "  请参考 %s 的文档了解操作方式",
		"config_next_step":              "下一步",
		"config_next_step_enter":        " 打开编辑器并开始修改",
		"config_next_step_esc":          " 返回工具列表，不进入编辑器",
		"config_footer_enter":           " 打开编辑器",
		"config_footer_esc":             " 返回列表",
		"editor_vim_insert":             "  i           进入编辑模式（插入文本）",
		"editor_vim_escape":             "  Esc         退出编辑模式（回到命令模式）",
		"editor_vim_write":              "  :w          保存文件",
		"editor_vim_quit":               "  :q          退出编辑器",
		"editor_vim_write_quit":         "  :wq         保存并退出",
		"editor_vim_force_quit":         "  :q!         不保存强制退出",
		"editor_vim_delete_line":        "  dd          删除整行",
		"editor_vim_undo":               "  u           撤销",
		"editor_vim_search":             "  /关键词     搜索",
		"editor_vim_start":              "  gg          跳到文件开头",
		"editor_vim_end":                "  G           跳到文件末尾",
		"editor_nano_write":             "  Ctrl+O      保存文件",
		"editor_nano_quit":              "  Ctrl+X      退出编辑器",
		"editor_nano_cut":               "  Ctrl+K      剪切整行",
		"editor_nano_paste":             "  Ctrl+U      粘贴",
		"editor_nano_search":            "  Ctrl+W      搜索",
		"editor_nano_help":              "  Ctrl+G      查看帮助",
		"npm_global":                    "npm 全局安装",
		"error_upgrade_not_supported":   "该工具不支持命令行升级",
		"error_missing_package_manager": "未找到 %s，请先安装",
		"error_upgrade_target_mismatch": "当前命中的可执行文件 %s 不属于 %s，请先统一安装来源后再升级",
	},
	LocaleEnglish: {
		"stderr_error":                  "Error: %v\n",
		"loading_detecting_tools":       "Detecting installed tools...",
		"scan_complete":                 "Scan complete",
		"upgrade_failed":                "Upgrade failed: %v",
		"upgrade_complete_rescanning":   "Upgrade complete, rescanning...",
		"manual_upgrade_opened":         "Opened %s. Please complete the upgrade inside the app",
		"config_closed_error":           "Editor exited with an error: %v",
		"title":                         "aic — AI CLI Tool Manager",
		"list_help":                     "  [a] Show/hide uninstalled  [u] Upgrade  [c] Config  [l] Switch language  [r] Rescan  [q] Quit",
		"list_footer":                   "  Use ↑↓/jk to move",
		"language_badge":                "EN",
		"language_switched_zh":          "已切换到中文",
		"language_switched_en":          "Switched to English",
		"summary_visible":               "  Showing %d / %d",
		"summary_hidden_hint":           ", %d uninstalled tools hidden, press [a] to show all",
		"empty_tools":                   "  No installed AI CLI tools detected",
		"empty_tools_hint_show_all":     "  Press [a] to show all candidates, press [r] to rescan",
		"empty_tools_hint_rescan":       "  Press [r] to rescan",
		"column_tool":                   "Tool",
		"column_status":                 "Status",
		"column_current_version":        "Current",
		"column_latest_version":         "Latest",
		"column_update_source":          "Source",
		"column_actions":                "Actions",
		"status_installed":              "Installed",
		"status_missing":                "Missing",
		"action_upgrade":                "u upg",
		"action_config":                 "c cfg",
		"source_none":                   "None",
		"source_manual_check":           "No API",
		"latest_manual_check":           "Press u",
		"config_title":                  "Edit Config — %s",
		"config_file":                   "Config File",
		"config_editor":                 "Editor: %s",
		"config_common_actions":         "Common Actions",
		"config_editor_reference":       "  See %s documentation for usage details",
		"config_next_step":              "Next Step",
		"config_next_step_enter":        " Open the editor and start editing",
		"config_next_step_esc":          " Return to the tool list without opening the editor",
		"config_footer_enter":           " Open editor",
		"config_footer_esc":             " Back to list",
		"editor_vim_insert":             "  i           Enter insert mode",
		"editor_vim_escape":             "  Esc         Leave insert mode",
		"editor_vim_write":              "  :w          Save the file",
		"editor_vim_quit":               "  :q          Quit the editor",
		"editor_vim_write_quit":         "  :wq         Save and quit",
		"editor_vim_force_quit":         "  :q!         Quit without saving",
		"editor_vim_delete_line":        "  dd          Delete the current line",
		"editor_vim_undo":               "  u           Undo",
		"editor_vim_search":             "  /keyword    Search",
		"editor_vim_start":              "  gg          Jump to the top",
		"editor_vim_end":                "  G           Jump to the bottom",
		"editor_nano_write":             "  Ctrl+O      Save the file",
		"editor_nano_quit":              "  Ctrl+X      Quit the editor",
		"editor_nano_cut":               "  Ctrl+K      Cut the current line",
		"editor_nano_paste":             "  Ctrl+U      Paste",
		"editor_nano_search":            "  Ctrl+W      Search",
		"editor_nano_help":              "  Ctrl+G      Show help",
		"npm_global":                    "the npm global installation",
		"error_upgrade_not_supported":   "This tool does not support command-line upgrades",
		"error_missing_package_manager": "%s was not found. Please install it first",
		"error_upgrade_target_mismatch": "The detected executable %s does not belong to %s. Please unify the installation source before upgrading",
	},
}
