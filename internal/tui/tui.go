package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/alexandria-proxy/alexandria-cli/internal/config"
)

func RunLangPicker(logo string) (string, error) {
	final, err := tea.NewProgram(NewLangPicker(logo), tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	if err != nil {
		return "", err
	}
	return final.(LangPicker).Chosen(), nil
}

func RunMenu(cfg config.Config, mono, color string) error {
	_, err := tea.NewProgram(NewMenu(cfg, mono, color), tea.WithAltScreen(), tea.WithMouseAllMotion()).Run()
	return err
}
