package tui

import (
	"context"
	"fmt"
	"strings"
	"terminalui/kubeutils"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func formatMsg(ad kubeutils.ActionDone) string {
	if ad.Duration == 0 {
		return dotStyle.Render(strings.Repeat(".", 30))
	}
	return fmt.Sprintf("* Pod: %s; Step: %s; Took: %s",
		podLabelStyle.Render(ad.PodName),
		stepNameStyle.Render(ad.Name),
		durationStyle.Render(ad.Duration.String()))
}

func (m *ConfiguratorModel) InitPodsPreparation() *PreparePodsModel {
	numLastResults := 10
	s := spinner.New()
	s.Style = spinnerStyle

	prepareCtx, cancel := context.WithCancel(m.ctx)

	pm := PreparePodsModel{
		spinner: s,
		results: make([]kubeutils.ActionDone, numLastResults),
		pods:    m.pods,
		logger:  m.logger,
		ctx:     prepareCtx,
		cancel:  cancel,
	}

	return &pm
}

func (m *ConfiguratorModel) handlePodsPreparationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.preparation.cancel()
			return m, tea.Quit
		}

	case kubeutils.ActionDone:
		m.preparation.results = append(m.preparation.results[1:], msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.preparation.spinner, cmd = m.preparation.spinner.Update(msg)
		return m, cmd
	default:
		if m.preparation.quitting {
			runView := m.InitRunView()
			m.run = runView
			m.currentView = Run
			return m, m.run.spinner.Tick
		}
	}

	return m, nil
}

func (m *ConfiguratorModel) handlePodsPreparationView() string {
	var b strings.Builder

	if m.preparation.quitting && m.preparation.err == "" {
		b.WriteString("Pods are now ready to run load tests!\n")
	} else {
		if m.preparation.err != "" {
			b.WriteString(accentInfo.Render("\n" + m.preparation.err + "\n"))
		}
		b.WriteString(m.preparation.spinner.View() + " Preparing pods...\n")
	}

	for _, res := range m.preparation.results {
		b.WriteString(formatMsg(res) + "\n")
	}

	if !m.preparation.quitting {
		b.WriteString(helpStyle.Render("\n\n\nPods are being prepared..."))
	}

	if m.preparation.quitting {
		b.WriteString(alertStyle.Render("\nPress 'c' to continue... "))
	}

	return appStyle.Render(b.String())
}
