package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	Language key.Binding
	Upgrade  key.Binding
	Config   key.Binding
	Rescan   key.Binding
	Quit     key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "上移"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "下移"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "显示全部"),
	),
	Language: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "切换语言"),
	),
	Upgrade: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "升级"),
	),
	Config: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "配置"),
	),
	Rescan: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "重新扫描"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "退出"),
	),
	Confirm: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "确认"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "取消"),
	),
}
