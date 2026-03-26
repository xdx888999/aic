package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xdx888999/aic/internal/i18n"
	"github.com/xdx888999/aic/internal/tui"
	"github.com/xdx888999/aic/internal/version"
)

func main() {
	if len(os.Args) > 1 && isVersionFlag(os.Args[1]) {
		fmt.Println(version.String())
		return
	}

	locale := i18n.DefaultLocale()
	localizer := i18n.NewLocalizer(locale)
	p := tea.NewProgram(tui.NewWithLocale(locale), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, localizer.Text("stderr_error"), err)
		os.Exit(1)
	}
}

func isVersionFlag(arg string) bool {
	return arg == "--version" || arg == "-version" || arg == "-v"
}
