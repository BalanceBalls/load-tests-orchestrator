package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *ConfiguratorModel) handleTestsSetupView() string {
	var b strings.Builder
	namespace := m.configForm.inputs[1].Value()

	b.WriteString(focusedStyle.Render("\nPrepare pods"))
	if m.err != nil {
		b.WriteString(accentInfo.Render("\n\n Error: " + m.err.Error() + "\n"))
	}
	b.WriteString(configInfoStyle.Render("\nNamespace: " + namespace))
	start, end := m.paginator.GetSliceBounds(len(m.pods))
	for _, item := range m.pods[start:end] {
		sf := alertStyle.Render("not set")
		if m.pods[m.paginator.Page].scenarioFilePath != "" {
			sf = configuredStyle.Render(m.pods[m.paginator.Page].scenarioFilePath)
		}

		pf := alertStyle.Render("not set")
		if m.pods[m.paginator.Page].propsFilePath != "" {
			pf = configuredStyle.Render(m.pods[m.paginator.Page].propsFilePath)
		}

		b.WriteString(configInfoStyle.Render("\nScneario file: " + sf))
		b.WriteString(configInfoStyle.Render("\nProperties file: " + pf))

		b.WriteString(configInfoStyle.Render("\nPod name" + divider))
		b.WriteString(podLabelStyle.Render(item.name))
		b.WriteString("\n" + m.paginator.View())
	}

	b.WriteString(helpStyle.Render("\n\ns: pick scenario file • p: pick properties file"))
	b.WriteString(helpStyle.Render("\nc: continue with current config"))
	b.WriteString(helpStyle.Render("\nh/l ←/→ page • ctrl+c: quit"))
	b.WriteString("\n\n")
	return b.String()
}

func (m *ConfiguratorModel) handleTestsSetupUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			m.currentView = FilePick
			m.filepicker = &FilePickerModule{
				model: GetFilePicker(true),
				mode:  0,
			}
			return m, m.filepicker.model.Init()
		case "p":
			m.currentView = FilePick
			m.filepicker = &FilePickerModule{
				model: GetFilePicker(false),
				mode:  1,
			}
			return m, m.filepicker.model.Init()
		case "c":
			isConfigured := true
			for _, pod := range m.pods {
				if pod.propsFilePath == "" || pod.scenarioFilePath == "" {
					isConfigured = false
					m.err = errors.New("not all pods have test files configured")
				}
			}

			if isConfigured {
				m.err = nil
				initiatedConfirm := m.InitConfirmation()
				m.setupConfirmation = &initiatedConfirm
				m.currentView = ReviewSetup
			}
			return m, nil
		}
	}

	updatedPaginator, cmd := m.paginator.Update(msg)
	m.paginator = &updatedPaginator
	return m, cmd
}

func (m *ConfiguratorModel) initTestsSetupView(totalPages int) {
	m.pods = make([]PodInfo, totalPages)
	m.err = nil

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")
	p.SetTotalPages(totalPages)

	podPrefix := m.configForm.inputs[0].Value()
	podCount, _ := strconv.Atoi(m.configForm.inputs[3].Value())
	for i := range podCount {
		m.pods[i].name = fmt.Sprintf("%s-%d", podPrefix, i)
		m.pods[i].id = i
	}
	m.paginator = &p
}
