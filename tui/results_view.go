package tui

import (
	"context"
	"strings"
	"terminalui/kubeutils"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (m *ConfiguratorModel) InitResultsPreparation() *PrepareResultsModel {
	numLastResults := 10
	s := spinner.New()
	s.Style = spinnerStyle

	prepareCtx, cancel := context.WithCancel(m.ctx)

	pm := PrepareResultsModel{
		spinner: s,
		results: make([]kubeutils.ActionDone, numLastResults),
		pods:    m.pods,
		logger:  m.logger,
		ctx:     prepareCtx,
		cancel:  cancel,
	}

	confirmationForm := huh.NewForm(huh.NewGroup(pm.getConfirmationDialog()))
	pm.deletePodsConfirm = confirmationForm
	return &pm
}

func (m *ConfiguratorModel) handleResultsPreparationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		} else {
			if !m.resultsCollection.showConfirmation {
				return m, nil
			}
		}

	case kubeutils.ActionDone:
		m.resultsCollection.results = append(m.resultsCollection.results[1:], msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.resultsCollection.spinner, cmd = m.resultsCollection.spinner.Update(msg)
		return m, cmd
	}

	if m.resultsCollection.isCollected {
		var confirmModel tea.Model
		confirmModel, formCmd := m.resultsCollection.deletePodsConfirm.Update(msg)
		cmds = append(cmds, formCmd)

		if f, ok := confirmModel.(*huh.Form); ok {
			if f.State == huh.StateCompleted {
				if f.GetBool("conf") {
					go m.deletePods()
				} else {
					m.resultsCollection.showConfirmation = false
					m.resultsCollection.quitting = true
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *ConfiguratorModel) handleResultsPreparationView() string {
	var b strings.Builder

	if m.resultsCollection.showConfirmation {
		b.WriteString(m.resultsCollection.deletePodsConfirm.View())
	} else {
		if m.resultsCollection.isCollected {
			b.WriteString("\nLoad test results have been downloaded!")
		} else {
			if m.resultsCollection.err != nil {
				b.WriteString("\n" + accentInfo.Render(m.resultsCollection.err.Error()))
			}
			b.WriteString("\n" + m.resultsCollection.spinner.View() + " Preparing results...")
		}

		b.WriteString("\n\n")
		for _, res := range m.resultsCollection.results {
			b.WriteString(formatMsg(res) + "\n")
		}

		if !m.resultsCollection.isCollected {
			b.WriteString(helpStyle.Render("Results are being collected..."))
		}

		if m.resultsCollection.quitting {
			b.WriteString(alertStyle.Render("\nPress 'ctrl+c' to exit"))
		}
	}

	return appStyle.Render(b.String())
}

func (m *PrepareResultsModel) getConfirmationDialog() *huh.Confirm {
	return huh.NewConfirm().
		Title(accentInfo.Render("Would you like to delete JMeter pods?")).
		Affirmative("Yes").
		Negative("No").
		Key("conf")
}
