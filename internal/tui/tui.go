package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func RunLangPicker(logo string) (string, error) {
	final, err := tea.NewProgram(NewLangPicker(logo), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	return final.(LangPicker).Chosen(), nil
}
