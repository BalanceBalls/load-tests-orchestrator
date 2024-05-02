package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func loadTestConfiguratorModel() *ConfiguratorModel {
	m := ConfiguratorModel{
		currentView: Config}

	m.initConfigForm()
	m.initPaginatorView(0)

	return &m
}

func (m *ConfiguratorModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ConfiguratorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.currentView == FilePick {
				m.currentView = PodsSetup
			}
		}
	}

	switch m.currentView {
	case Config:
		return m.handleConfigFormUpdate(msg)
	case PodsSetup:
		return m.handlePaginatorUpdate(msg)
	case FilePick:
		return m.handleFilepickerUpdate(msg)
	case ReviewSetup:
		return m.handleConfirmationUpdate(msg)
	case PreparePods:
		return m.handlePodsPreparationUpdate(msg)
	case Run:
		return m.handleRunViewUpdate(msg)
	default:
		return m, nil
	}
}

func (m *ConfiguratorModel) View() string {
	switch m.currentView {
	case Config:
		return m.handleConfigFormView()
	case PodsSetup:
		return m.handlePaginatorView()
	case FilePick:
		return m.handleFilepickerView()
	case ReviewSetup:
		return m.handleConfirmationView()
	case PreparePods:
		return m.handlePodsPreparationView()
	case Run:
		return m.handleRunView()
	default:
		return ""
	}
}

func DisplayUI() {
	configurationProgram := tea.NewProgram(
		loadTestConfiguratorModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	_, err := configurationProgram.Run()

	if err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
