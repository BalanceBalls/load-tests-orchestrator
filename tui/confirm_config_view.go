package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (m *ConfiguratorModel) handleConfirmationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "b":
			m.currentView = PodsSetup
		case "ctrl+p":
			setupPods(m)
			return m, m.preparation.spinner.Tick
		case "c":
			if m.setupConfirmation.isConfirmed {
				setupPods(m)
				return m, m.preparation.spinner.Tick
			}
		}
	case tea.WindowSizeMsg:
		m.setupConfirmation.viewport.Width = msg.Width
		m.setupConfirmation.viewport.Height = viewportHeight
		cmds = append(cmds, viewport.Sync(m.setupConfirmation.viewport))
	}

	var fCmd tea.Cmd
	viewPortPosition := int(m.setupConfirmation.viewport.ScrollPercent() * 100)
	if viewPortPosition > 98 {
		var cf tea.Model
		cf, fCmd = m.setupConfirmation.confirmationForm.Update(msg)
		if f, ok := cf.(*huh.Form); ok {
			m.setupConfirmation.confirmationForm = *f
			if f.GetBool("conf") {
				m.setupConfirmation.isConfirmed = true
			}
		}
	}

	m.setupConfirmation.viewport, cmd = m.setupConfirmation.viewport.Update(msg)
	cmds = append(cmds, cmd, fCmd)

	return m, tea.Batch(cmds...)
}

func setupPods(m *ConfiguratorModel) {
	pf := m.InitPodsPreparation()
	m.preparation = pf
	m.currentView = PreparePods

	go func() {
		ch := make(chan stepDone)
		defer close(ch)
		go m.preparation.beginSetup(ch)

		for r := range ch {
			m.Update(r)
			m.Update(m.preparation.spinner.Tick)
		}
	}()
}

func (m ConfiguratorModel) handleConfirmationView() string {
	var b strings.Builder
	helpMsg := helpStyle.Render("\nj/k: down, up • ctrl+d/u: half page down, up") +
		helpStyle.Render("\nb: go back to configuration • ctrl+c: quit")

	conf := ""
	viewPortPosition := int(m.setupConfirmation.viewport.ScrollPercent() * 100)

	if viewPortPosition > 98 {
		conf = "\n" + m.setupConfirmation.confirmationForm.View()
	}

	if m.setupConfirmation.isConfirmed {
		conf += "\n" + alertStyle.Render("Configuration confirmed! Press 'c' to continue")
	}

	b.WriteString(m.setupConfirmation.viewport.View())
	b.WriteString("\n" + conf)
	b.WriteString("\n" + helpMsg)

	return b.String()
}

func (m ConfiguratorModel) InitConfirmation() ConfirmationModel {
	vp := viewport.New(120, viewportHeight)
	vp.MouseWheelEnabled = true
	vp.SetContent(prepareRunInfo(m.pods))

	f := huh.NewForm(huh.NewGroup(m.GetConfirmationDialog()))

	cm := ConfirmationModel{
		isConfirmed:      false,
		content:          prepareRunInfo(m.pods),
		ready:            true,
		viewport:         vp,
		confirmationForm: *f,
	}

	return cm
}

func prepareRunInfo(pods []PodInfo) string {
	var b strings.Builder

	b.WriteString(accentInfo.Render("\nThe test will run with the following configuration:\n"))
	for _, pod := range pods {
		podLabel := podLabelStyle.Render("Pod name: " + pod.name)
		b.WriteString("\n" + podLabel + "\n")
		b.WriteString(configInfoStyle.
			Render("\nScenario file :" + pod.scenarioFilePath))
		b.WriteString(configInfoStyle.
			Render("\nProperties file: " + pod.propsFilePath))

		propsFileRows := readFile(pod.propsFilePath)
		propsFileContent := strings.Join(propsFileRows, "\n")
		propsFileContent = strings.TrimSpace(propsFileContent)
		b.WriteString(propsStyle.Render("\n" + propsFileContent))
	}

	return b.String()
}

func (m ConfiguratorModel) GetConfirmationDialog() *huh.Confirm {
	return huh.NewConfirm().
		Title(accentInfo.Render("Do you want to proceed with this config?")).
		Affirmative("Yes").
		Negative("No").
		Key("conf")
}
