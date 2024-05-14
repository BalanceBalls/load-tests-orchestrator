package tui

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func loadTestConfiguratorModel(appCtx context.Context, appLogger *slog.Logger, updateIntervalSec int, podKeepAliveSec int) *ConfiguratorModel {
	m := ConfiguratorModel{
		ctx:               appCtx,
		logger:            appLogger,
		updateIntervalSec: updateIntervalSec,
		podKeepAliveSec:   podKeepAliveSec,
		currentView:       Config}

	m.initConfigForm()
	m.logger.Info("First form initiated", slog.Any("pod keep alive", podKeepAliveSec), slog.Any("upd interval", updateIntervalSec))

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
		return m.handleTestsSetupUpdate(msg)
	case FilePick:
		return m.handleFilepickerUpdate(msg)
	case ReviewSetup:
		return m.handleConfirmationUpdate(msg)
	case PreparePods:
		return m.handlePodsPreparationUpdate(msg)
	case Run:
		return m.handleRunViewUpdate(msg)
	case Collect:
		return m.handleResultsPreparationUpdate(msg)
	default:
		return m, nil
	}
}

func (m *ConfiguratorModel) View() string {
	switch m.currentView {
	case Config:
		return m.handleConfigFormView()
	case PodsSetup:
		return m.handleTestsSetupView()
	case FilePick:
		return m.handleFilepickerView()
	case ReviewSetup:
		return m.handleConfirmationView()
	case PreparePods:
		return m.handlePodsPreparationView()
	case Run:
		return m.handleRunView()
	case Collect:
		return m.handleResultsPreparationView()
	default:
		return ""
	}
}

func DisplayUI(ctx context.Context, logger *slog.Logger, updateIntervalSec int, podKeepAliveSec int) {
	logger.Info("Loading UI...")
	configurationProgram := tea.NewProgram(
		loadTestConfiguratorModel(ctx, logger, updateIntervalSec, podKeepAliveSec),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion())

	_, err := configurationProgram.Run()

	if err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
