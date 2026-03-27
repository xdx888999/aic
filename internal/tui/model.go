package tui

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xdx888999/aic/internal/actions"
	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/i18n"
	"github.com/xdx888999/aic/internal/registry"
)

type state int

const (
	stateLoading state = iota
	stateList
	stateConfigHelp
)

const (
	msgDurationSuccess = 3 * time.Second
	msgDurationError   = 5 * time.Second
)

type toolsDetectedMsg []detector.Status
type rescanCompleteMsg []detector.Status
type clearMessageMsg struct{}

type Model struct {
	tools      []detector.Status
	cursor     int
	state      state
	width      int
	height     int
	showAll    bool
	spinner    spinner.Model
	message    string
	msgErr     bool
	configPath string
	configTool string
	localizer  i18n.Localizer
}

func New() Model {
	return NewWithLocale(i18n.DefaultLocale())
}

func NewWithLocale(locale i18n.Locale) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle
	return Model{
		state:     stateLoading,
		showAll:   false,
		spinner:   s,
		localizer: i18n.NewLocalizer(locale),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		detectAllCmd(),
	)
}

func detectAllCmd() tea.Cmd {
	return func() tea.Msg {
		tools := registry.AllTools()
		results := detector.DetectAll(tools)
		return toolsDetectedMsg(results)
	}
}

func rescanCmd() tea.Cmd {
	return func() tea.Msg {
		tools := registry.AllTools()
		results := detector.DetectAll(tools)
		return rescanCompleteMsg(results)
	}
}

func clearMsgAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case toolsDetectedMsg:
		m.tools = []detector.Status(msg)
		m.state = stateList
		m.clampCursor()
		return m, nil

	case rescanCompleteMsg:
		m.tools = []detector.Status(msg)
		m.state = stateList
		m.clampCursor()
		m.message = m.localizer.Text("scan_complete")
		m.msgErr = false
		return m, clearMsgAfter(msgDurationSuccess)

	case actions.UpgradeFinishedMsg:
		if msg.Err != nil {
			m.message = m.formatUpgradeError(msg.Err)
			m.msgErr = true
			return m, clearMsgAfter(msgDurationError)
		} else {
			m.message = m.localizer.Text("upgrade_complete_rescanning")
			m.msgErr = false
			m.state = stateLoading
		}
		return m, tea.Batch(
			m.spinner.Tick,
			rescanCmd(),
			clearMsgAfter(msgDurationError),
		)

	case actions.ConfigClosedMsg:
		if msg.Err != nil {
			m.message = m.localizer.Text("config_closed_error", msg.Err)
			m.msgErr = true
			return m, clearMsgAfter(msgDurationError)
		}
		return m, nil

	case clearMessageMsg:
		m.message = ""
		return m, nil

	case spinner.TickMsg:
		if m.state == stateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch m.state {
		case stateConfigHelp:
			switch {
			case key.Matches(msg, keys.Language):
				m.toggleLocale()
				return m, nil
			case key.Matches(msg, keys.Confirm):
				m.state = stateList
				return m, actions.OpenConfigCmd(m.configPath)
			case key.Matches(msg, keys.Cancel), key.Matches(msg, keys.Quit):
				m.state = stateList
				return m, nil
			}
			return m, nil

		case stateList:
			switch {
			case key.Matches(msg, keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, keys.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, keys.Down):
				if m.cursor < m.visibleCount()-1 {
					m.cursor++
				}
			case key.Matches(msg, keys.Toggle):
				m.showAll = !m.showAll
				m.clampCursor()
			case key.Matches(msg, keys.Language):
				m.toggleLocale()
			case key.Matches(msg, keys.Upgrade):
				t, selectedIndex, ok := m.selectedTool()
				if ok && t.Installed && len(t.Tool.UpgradeCmd) > 0 {
					return m, actions.UpgradeCmd(t.Tool, selectedIndex)
				}
			case key.Matches(msg, keys.Config):
				t, _, ok := m.selectedTool()
				if ok && t.HasConfig {
					m.configPath = t.ConfigPath
					m.configTool = t.Tool.Name
					m.state = stateConfigHelp
					return m, nil
				}
			case key.Matches(msg, keys.Rescan):
				m.state = stateLoading
				m.message = ""
				return m, tea.Batch(m.spinner.Tick, rescanCmd())
			}

		default:
			return m, nil
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateLoading:
		s := "\n  " + m.spinner.View() + " " + m.localizer.Text("loading_detecting_tools") + "\n"
		if m.message != "" {
			s += "\n"
			if m.msgErr {
				s += "  " + errorStyle.Render("✗ "+m.message) + "\n"
			} else {
				s += "  " + successStyle.Render("✓ "+m.message) + "\n"
			}
		}
		return s

	case stateConfigHelp:
		return m.renderConfigHelp()

	case stateList:
		visibleTools := m.visibleTools()
		s := "\n"
		titleLine := lipgloss.JoinHorizontal(
			lipgloss.Top,
			titleStyle.Render(m.localizer.Text("title")),
			" ",
			languageBadgeStyle.Render(m.localizer.Text("language_badge")),
			"  ",
			signatureStyle.Render("xdx_lab"),
		)
		s += "  " + titleLine + "\n\n"
		s += renderList(visibleTools, m.cursor, m.width, m.showAll, len(m.tools), m.localizer)

		if m.message != "" {
			s += "\n"
			if m.msgErr {
				s += "  " + errorStyle.Render("✗ "+m.message) + "\n"
			} else {
				s += "  " + successStyle.Render("✓ "+m.message) + "\n"
			}
		}

		help := m.localizer.Text("list_help")
		s += helpStyle.Render(help) + "\n"

		footer := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render(
			m.localizer.Text("list_footer"))
		s += footer + "\n"

		return s
	}
	return ""
}

func (m Model) editorName() string {
	return actions.ResolveEditor()
}

func (m Model) visibleTools() []detector.Status {
	if m.showAll {
		return m.tools
	}

	visible := make([]detector.Status, 0, len(m.tools))
	for _, tool := range m.tools {
		if tool.Installed {
			visible = append(visible, tool)
		}
	}
	return visible
}

func (m Model) visibleToolIndexes() []int {
	indexes := make([]int, 0, len(m.tools))
	for index, tool := range m.tools {
		// 这里保留原始索引，避免“只显示已安装”时列表位置变化，
		// 导致升级或打开配置命中了错误的工具。
		if m.showAll || tool.Installed {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

func (m Model) visibleCount() int {
	return len(m.visibleToolIndexes())
}

func (m *Model) clampCursor() {
	visibleCount := m.visibleCount()
	if visibleCount == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= visibleCount {
		m.cursor = visibleCount - 1
	}
}

func (m Model) selectedTool() (detector.Status, int, bool) {
	visibleIndexes := m.visibleToolIndexes()
	if len(visibleIndexes) == 0 || m.cursor < 0 || m.cursor >= len(visibleIndexes) {
		return detector.Status{}, -1, false
	}
	selectedIndex := visibleIndexes[m.cursor]
	return m.tools[selectedIndex], selectedIndex, true
}

func (m Model) renderConfigHelp() string {
	editor := m.editorName()

	title := titleStyle.Render(m.localizer.Text("config_title", m.configTool))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("99")).
		Padding(1, 2).
		MarginTop(1).
		MarginLeft(2)

	var content string
	content += headerStyle.Render(m.localizer.Text("config_file")) + "\n"
	content += "  " + versionStyle.Render(m.configPath) + "\n\n"
	content += headerStyle.Render(m.localizer.Text("config_editor", editor)) + "\n\n"

	if editor == "vim" || editor == "nvim" || editor == "vi" {
		content += headerStyle.Render(m.localizer.Text("config_common_actions")) + "\n"
		content += separatorStyle.Render(strings.Repeat("─", 40)) + "\n"
		content += m.localizer.Text("editor_vim_insert") + "\n"
		content += m.localizer.Text("editor_vim_escape") + "\n"
		content += m.localizer.Text("editor_vim_write") + "\n"
		content += m.localizer.Text("editor_vim_quit") + "\n"
		content += m.localizer.Text("editor_vim_write_quit") + "\n"
		content += m.localizer.Text("editor_vim_force_quit") + "\n"
		content += separatorStyle.Render(strings.Repeat("─", 40)) + "\n"
		content += m.localizer.Text("editor_vim_delete_line") + "\n"
		content += m.localizer.Text("editor_vim_undo") + "\n"
		content += m.localizer.Text("editor_vim_search") + "\n"
		content += m.localizer.Text("editor_vim_start") + "\n"
		content += m.localizer.Text("editor_vim_end") + "\n"
	} else if editor == "nano" {
		content += headerStyle.Render(m.localizer.Text("config_common_actions")) + "\n"
		content += separatorStyle.Render(strings.Repeat("─", 40)) + "\n"
		content += m.localizer.Text("editor_nano_write") + "\n"
		content += m.localizer.Text("editor_nano_quit") + "\n"
		content += m.localizer.Text("editor_nano_cut") + "\n"
		content += m.localizer.Text("editor_nano_paste") + "\n"
		content += m.localizer.Text("editor_nano_search") + "\n"
		content += m.localizer.Text("editor_nano_help") + "\n"
	} else {
		content += missingStyle.Render(m.localizer.Text("config_editor_reference", editor)) + "\n"
	}

	content += "\n"
	content += headerStyle.Render(m.localizer.Text("config_next_step")) + "\n"
	content += separatorStyle.Render(strings.Repeat("─", 40)) + "\n"
	content += "  " + primaryKeyStyle.Render("Enter") + keyHintStyle.Render(m.localizer.Text("config_next_step_enter")) + "\n"
	content += "  " + secondaryKeyStyle.Render("Esc") + keyHintStyle.Render(m.localizer.Text("config_next_step_esc")) + "\n"

	box := boxStyle.Render(content)

	footer := "\n  " +
		primaryKeyStyle.Render("Enter") +
		keyHintStyle.Render(m.localizer.Text("config_footer_enter")) +
		"    " +
		secondaryKeyStyle.Render("Esc") +
		keyHintStyle.Render(m.localizer.Text("config_footer_esc"))

	return "\n  " + title + "\n" + box + footer + "\n"
}

func (m Model) formatUpgradeError(err error) string {
	var actionErr actions.ActionError
	if errors.As(err, &actionErr) {
		switch actionErr.Code {
		case actions.ErrorUpgradeNotSupported:
			return m.localizer.Text("upgrade_failed", m.localizer.Text("error_upgrade_not_supported"))
		case actions.ErrorMissingPackageManager:
			return m.localizer.Text("upgrade_failed", m.localizer.Text("error_missing_package_manager", actionErr.Arg))
		}
	}
	return m.localizer.Text("upgrade_failed", err)
}

func (m *Model) toggleLocale() {
	if m.localizer.Locale() == i18n.LocaleChinese {
		m.localizer = i18n.NewLocalizer(i18n.LocaleEnglish)
		m.message = m.localizer.Text("language_switched_en")
	} else {
		m.localizer = i18n.NewLocalizer(i18n.LocaleChinese)
		m.message = m.localizer.Text("language_switched_zh")
	}
	m.msgErr = false
}
