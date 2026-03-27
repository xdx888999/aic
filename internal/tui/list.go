package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xdx888999/aic/internal/detector"
	"github.com/xdx888999/aic/internal/i18n"
	"github.com/xdx888999/aic/internal/registry"
)

const (
	defaultTableWidth  = 92
	minTableWidth      = 68
	columnGapWidth     = 3
	nameColumnWidth    = 15
	statusColumnWidth  = 10
	currentColumnWidth = 12
	latestColumnWidth  = 12
	sourceColumnWidth  = 10
	actionColumnWidth  = 12
)

type tableColumn struct {
	title string
	width int
	min   int
}

func renderList(tools []detector.Status, cursor int, width int, showAll bool, totalCount int, localizer i18n.Localizer) string {
	var builder strings.Builder

	hiddenCount := totalCount - len(tools)
	summary := localizer.Text("summary_visible", len(tools), totalCount)
	if !showAll && hiddenCount > 0 {
		summary += localizer.Text("summary_hidden_hint", hiddenCount)
	}
	builder.WriteString(helpStyle.Render(summary))
	builder.WriteString("\n")

	if len(tools) == 0 {
		builder.WriteString("\n")
		builder.WriteString(missingStyle.Render(localizer.Text("empty_tools")))
		builder.WriteString("\n")
		if hiddenCount > 0 {
			builder.WriteString(helpStyle.Render(localizer.Text("empty_tools_hint_show_all")))
		} else {
			builder.WriteString(helpStyle.Render(localizer.Text("empty_tools_hint_rescan")))
		}
		builder.WriteString("\n")
		return builder.String()
	}

	columns := fitColumns(width, localizer)
	header := renderHeader(columns)
	builder.WriteString(header)
	builder.WriteString("\n")
	builder.WriteString(separatorStyle.Render("  " + strings.Repeat("─", lipgloss.Width(header)-2)))
	builder.WriteString("\n")

	for index, tool := range tools {
		line := renderRow(tool, columns, index == cursor, localizer)
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return builder.String()
}

func fitColumns(terminalWidth int, localizer i18n.Localizer) []tableColumn {
	columns := []tableColumn{
		{title: localizer.Text("column_tool"), width: nameColumnWidth, min: 10},
		{title: localizer.Text("column_status"), width: statusColumnWidth, min: 8},
		{title: localizer.Text("column_current_version"), width: currentColumnWidth, min: 10},
		{title: localizer.Text("column_latest_version"), width: latestColumnWidth, min: 10},
		{title: localizer.Text("column_update_source"), width: sourceColumnWidth, min: 8},
		{title: localizer.Text("column_actions"), width: actionColumnWidth, min: 10},
	}

	targetWidth := max(min(terminalWidth-4, defaultTableWidth), minTableWidth)
	currentWidth := totalTableWidth(columns)
	if currentWidth <= targetWidth {
		return columns
	}

	shrinkOrder := []int{5, 0, 2, 3, 4, 1}
	for currentWidth > targetWidth {
		shrunk := false
		for _, index := range shrinkOrder {
			if columns[index].width <= columns[index].min {
				continue
			}
			columns[index].width--
			currentWidth--
			shrunk = true
			if currentWidth <= targetWidth {
				break
			}
		}
		if !shrunk {
			break
		}
	}

	return columns
}

func totalTableWidth(columns []tableColumn) int {
	width := 2
	for index, column := range columns {
		width += column.width
		if index < len(columns)-1 {
			width += columnGapWidth
		}
	}
	return width
}

func renderHeader(columns []tableColumn) string {
	cells := make([]string, 0, len(columns))
	for _, column := range columns {
		cells = append(cells, headerStyle.Render(padOrTrim(column.title, column.width)))
	}
	return "  " + strings.Join(cells, " │ ")
}

func renderRow(tool detector.Status, columns []tableColumn, selected bool, localizer i18n.Localizer) string {
	cursorMark := "  "
	if selected {
		cursorMark = "▸ "
	}

	statusText, statusStyle := renderStatusCell(tool, localizer)
	currentText, currentStyle, latestText, latestStyle := renderVersionCells(tool)
	sourceText, sourceStyle := renderSourceCell(tool, localizer)
	actionText := renderActionText(tool, localizer)

	cells := []string{
		renderTableCell(tool.Tool.Name, columns[0].width, versionStyle, selected),
		renderTableCell(statusText, columns[1].width, statusStyle, selected),
		renderTableCell(currentText, columns[2].width, currentStyle, selected),
		renderTableCell(latestText, columns[3].width, latestStyle, selected),
		renderTableCell(sourceText, columns[4].width, sourceStyle, selected),
		renderTableCell(actionText, columns[5].width, actionStyle, selected),
	}

	separator := separatorStyle.Render(" │ ")
	if selected {
		separator = selectedStyle.Render(" │ ")
	}

	return cursorMark + strings.Join(cells, separator)
}

func renderStatusCell(tool detector.Status, localizer i18n.Localizer) (string, lipgloss.Style) {
	if tool.Installed {
		return localizer.Text("status_installed"), installedStyle
	}
	return localizer.Text("status_missing"), missingStyle
}

func renderVersionCells(tool detector.Status) (string, lipgloss.Style, string, lipgloss.Style) {
	currentVersion := tool.Version
	if currentVersion == "" {
		currentVersion = "-"
	}

	latestVersion := tool.LatestVersion
	if latestVersion == "" {
		latestVersion = "-"
	}

	if !tool.Installed {
		return "-", missingStyle, latestVersion, missingStyle
	}

	if currentVersion == "-" {
		return currentVersion, missingStyle, latestVersion, missingStyle
	}

	if latestVersion == "-" {
		return currentVersion, upToDateStyle, latestVersion, missingStyle
	}

	compareResult := detector.CompareVersions(currentVersion, latestVersion)
	if compareResult < 0 {
		return currentVersion + " ↑", outdatedStyle, latestVersion, outdatedStyle
	}

	return currentVersion, upToDateStyle, latestVersion, missingStyle
}

func renderActionText(tool detector.Status, localizer i18n.Localizer) string {
	parts := make([]string, 0, 2)
	if tool.Installed && len(tool.Tool.UpgradeCmd) > 0 {
		parts = append(parts, localizer.Text("action_upgrade"))
	}
	if tool.HasConfig {
		parts = append(parts, localizer.Text("action_config"))
	}
	return strings.Join(parts, " ")
}

func renderSourceCell(tool detector.Status, localizer i18n.Localizer) (string, lipgloss.Style) {
	sourceText := registry.DisplayLatestVersionProvider(tool.Tool.LatestVersion.Provider)
	if sourceText == "" {
		sourceText = localizer.Text("source_none")
	}

	switch tool.Tool.LatestVersion.Provider {
	case registry.LatestVersionProviderNPM:
		return sourceText, npmSourceStyle
	case registry.LatestVersionProviderNPMDistTag:
		return sourceText, npmSourceStyle
	case registry.LatestVersionProviderPyPI:
		return sourceText, pypiSourceStyle
	case registry.LatestVersionProviderGitHubRelease:
		return sourceText, gitHubSourceStyle
	case registry.LatestVersionProviderHomebrewCask:
		return sourceText, homebrewSourceStyle
	default:
		return sourceText, noSourceStyle
	}
}

func renderTableCell(content string, width int, style lipgloss.Style, selected bool) string {
	cellStyle := style.Width(width)
	if selected {
		cellStyle = cellStyle.Inherit(selectedStyle)
	}
	return cellStyle.Render(padOrTrim(content, width))
}

func padOrTrim(content string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(content) <= width {
		return content + strings.Repeat(" ", width-lipgloss.Width(content))
	}

	if width <= 3 {
		return strings.Repeat(".", width)
	}

	var builder strings.Builder
	currentWidth := 0
	targetWidth := width - 3

	for _, runeValue := range content {
		runeWidth := lipgloss.Width(string(runeValue))
		if currentWidth+runeWidth > targetWidth {
			break
		}
		builder.WriteRune(runeValue)
		currentWidth += runeWidth
	}

	trimmed := builder.String() + "..."
	if padding := width - lipgloss.Width(trimmed); padding > 0 {
		trimmed += strings.Repeat(" ", padding)
	}
	return trimmed
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
