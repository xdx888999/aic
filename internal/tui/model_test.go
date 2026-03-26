package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdx888999/aic/internal/actions"
	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/i18n"
	"github.com/xdx888999/aic/internal/registry"
)

func TestVisibleToolsHideUninstalledByDefault(t *testing.T) {
	model := Model{
		tools: []detector.Status{
			{Tool: registry.Tool{Name: "Installed Tool"}, Installed: true},
			{Tool: registry.Tool{Name: "Missing Tool"}, Installed: false},
		},
	}

	visible := model.visibleTools()
	if len(visible) != 1 {
		t.Fatalf("期望默认只显示 1 个已安装工具，实际为 %d", len(visible))
	}
	if visible[0].Tool.Name != "Installed Tool" {
		t.Fatalf("期望显示已安装工具，实际显示 %q", visible[0].Tool.Name)
	}
}

func TestVisibleToolsShowAllAfterToggle(t *testing.T) {
	model := Model{
		showAll: true,
		tools: []detector.Status{
			{Tool: registry.Tool{Name: "Installed Tool"}, Installed: true},
			{Tool: registry.Tool{Name: "Missing Tool"}, Installed: false},
		},
	}

	visible := model.visibleTools()
	if len(visible) != 2 {
		t.Fatalf("期望切换后显示全部 2 个工具，实际为 %d", len(visible))
	}
}

func TestSelectedToolReturnsOriginalIndex(t *testing.T) {
	// 这里验证“可见列表索引”和“原始工具索引”不会混淆，
	// 否则升级和配置操作可能作用到错误的工具上。
	model := Model{
		cursor:  1,
		showAll: true,
		tools: []detector.Status{
			{Tool: registry.Tool{Name: "First Tool"}, Installed: true},
			{Tool: registry.Tool{Name: "Second Tool"}, Installed: false},
			{Tool: registry.Tool{Name: "Third Tool"}, Installed: true},
		},
	}

	selected, index, ok := model.selectedTool()
	if !ok {
		t.Fatal("期望能够选中工具，但返回了未选中")
	}
	if index != 1 {
		t.Fatalf("期望返回原始索引 1，实际为 %d", index)
	}
	if selected.Tool.Name != "Second Tool" {
		t.Fatalf("期望选中 Second Tool，实际为 %q", selected.Tool.Name)
	}
}

func TestClampCursorResetsWhenVisibleToolsEmpty(t *testing.T) {
	model := Model{
		cursor: 3,
		tools: []detector.Status{
			{Tool: registry.Tool{Name: "Missing Tool"}, Installed: false},
		},
	}

	model.clampCursor()
	if model.cursor != 0 {
		t.Fatalf("期望空列表时光标重置为 0，实际为 %d", model.cursor)
	}
}

func TestRenderListShowsHiddenHint(t *testing.T) {
	output := renderList(
		[]detector.Status{{Tool: registry.Tool{Name: "Installed Tool"}, Installed: true}},
		0,
		120,
		false,
		2,
		i18n.NewLocalizer(i18n.LocaleChinese),
	)

	if !strings.Contains(output, "按 [a] 查看全部") {
		t.Fatalf("期望列表包含隐藏提示，实际输出为 %q", output)
	}
}

func TestRenderListShowsUpdateSourceColumn(t *testing.T) {
	output := renderList(
		[]detector.Status{
			{
				Tool: registry.Tool{
					Name: "Codex CLI",
					LatestVersion: registry.VersionSource{
						Provider: registry.LatestVersionProviderNPM,
					},
				},
				Installed:     true,
				Version:       "0.34.1",
				LatestVersion: "0.35.0",
			},
		},
		0,
		140,
		true,
		1,
		i18n.NewLocalizer(i18n.LocaleChinese),
	)

	if !strings.Contains(output, "更新源") {
		t.Fatalf("期望表头包含更新源列，实际输出为 %q", output)
	}
	if !strings.Contains(output, "npm") {
		t.Fatalf("期望来源列显示 npm，实际输出为 %q", output)
	}
	if !strings.Contains(output, "0.34.1 ↑") {
		t.Fatalf("期望可升级时当前版本带向上箭头，实际输出为 %q", output)
	}
}

func TestRenderListShowsNoSourceText(t *testing.T) {
	output := renderList(
		[]detector.Status{
			{
				Tool: registry.Tool{
					Name: "Kiro CLI",
				},
				Installed: true,
				Version:   "1.0.0",
			},
		},
		0,
		140,
		true,
		1,
		i18n.NewLocalizer(i18n.LocaleChinese),
	)

	if !strings.Contains(output, "无上游") {
		t.Fatalf("期望来源列显示无上游，实际输出为 %q", output)
	}
}

func TestRenderListActionCellDoesNotContainExtraNewline(t *testing.T) {
	output := renderList(
		[]detector.Status{
			{
				Tool: registry.Tool{
					Name:       "Codex CLI",
					UpgradeCmd: []string{"npm", "update", "-g", "@openai/codex"},
				},
				Installed:  true,
				HasConfig:  true,
				Version:    "0.34.1",
				ConfigPath: "/tmp/config.json",
			},
		},
		0,
		140,
		true,
		1,
		i18n.NewLocalizer(i18n.LocaleChinese),
	)

	if !strings.Contains(output, "u升级 c配置") {
		t.Fatalf("期望操作列使用紧凑文案，实际输出为 %q", output)
	}
	if strings.Contains(output, "u升级\n") || strings.Contains(output, "c配置\n") {
		t.Fatalf("期望操作列不在单元格内换行，实际输出为 %q", output)
	}
}

func TestRenderListEnglishActionTextFitsColumn(t *testing.T) {
	output := renderList(
		[]detector.Status{
			{
				Tool: registry.Tool{
					Name:       "Codex CLI",
					UpgradeCmd: []string{"npm", "install", "-g", "@openai/codex@latest"},
				},
				Installed: true,
				HasConfig: true,
				Version:   "0.34.1",
			},
		},
		0,
		140,
		true,
		1,
		i18n.NewLocalizer(i18n.LocaleEnglish),
	)

	if !strings.Contains(output, "u upg c cfg") {
		t.Fatalf("期望英文操作列使用紧凑文案，实际输出为 %q", output)
	}
}

func TestRenderConfigHelpShowsEnterAndEscHints(t *testing.T) {
	model := Model{
		configPath: "/tmp/config.json",
		configTool: "Codex CLI",
		localizer:  i18n.NewLocalizer(i18n.LocaleChinese),
	}

	output := model.renderConfigHelp()
	if !strings.Contains(output, "Enter") {
		t.Fatalf("期望配置帮助页显式提示 Enter，实际输出为 %q", output)
	}
	if !strings.Contains(output, "Esc") {
		t.Fatalf("期望配置帮助页显式提示 Esc，实际输出为 %q", output)
	}
	if !strings.Contains(output, "打开编辑器") {
		t.Fatalf("期望配置帮助页说明 Enter 的作用，实际输出为 %q", output)
	}
}

func TestUpgradeFinishedSwitchesToLoadingState(t *testing.T) {
	model := Model{
		state:     stateList,
		localizer: i18n.NewLocalizer(i18n.LocaleChinese),
	}

	updatedModel, cmd := model.Update(actions.UpgradeFinishedMsg{})
	result, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("期望返回具体的 Model 类型")
	}
	if result.state != stateLoading {
		t.Fatalf("期望升级成功后切换到加载态，实际状态为 %v", result.state)
	}
	if cmd == nil {
		t.Fatal("期望升级成功后触发重新扫描命令")
	}
}

func TestUpgradeFailedKeepsListState(t *testing.T) {
	model := Model{
		state:     stateList,
		localizer: i18n.NewLocalizer(i18n.LocaleChinese),
	}

	updatedModel, _ := model.Update(actions.UpgradeFinishedMsg{Err: errors.New("upgrade failed")})
	result, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("期望返回具体的 Model 类型")
	}
	if result.state != stateList {
		t.Fatalf("期望升级失败后保留列表态，实际状态为 %v", result.state)
	}
	if !result.msgErr {
		t.Fatal("期望升级失败时标记错误状态")
	}
}

func TestLoadingViewShowsStatusMessage(t *testing.T) {
	model := Model{
		state:     stateLoading,
		message:   "升级完成，正在重新检测...",
		localizer: i18n.NewLocalizer(i18n.LocaleChinese),
	}

	output := model.View()
	if !strings.Contains(output, "升级完成，正在重新检测...") {
		t.Fatalf("期望加载页显示状态消息，实际输出为 %q", output)
	}
}

func TestNewDefaultsToEnglish(t *testing.T) {
	model := New()
	if model.localizer.Locale() != i18n.LocaleEnglish {
		t.Fatalf("期望默认启动语言为英文，实际为 %q", model.localizer.Locale())
	}
}

func TestLanguageKeyTogglesLocale(t *testing.T) {
	model := NewWithLocale(i18n.LocaleEnglish)
	model.state = stateList

	updatedModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	result, ok := updatedModel.(Model)
	if !ok {
		t.Fatal("期望返回具体的 Model 类型")
	}
	if result.localizer.Locale() != i18n.LocaleChinese {
		t.Fatalf("期望按 l 后切换到中文，实际为 %q", result.localizer.Locale())
	}
	if !strings.Contains(result.message, "中文") {
		t.Fatalf("期望切换后显示语言提示，实际消息为 %q", result.message)
	}
}

func TestTitleShowsLanguageBadge(t *testing.T) {
	model := NewWithLocale(i18n.LocaleEnglish)
	model.state = stateList

	output := model.View()
	if !strings.Contains(output, "EN") {
		t.Fatalf("期望标题栏显示当前语言标识，实际输出为 %q", output)
	}
}
