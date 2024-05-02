package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)

	propsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3268a8")).
			Italic(true).
			MarginLeft(1)
)

const useHighPerformanceRenderer = true
const viewportHeight = 20

type ConfirmationModel struct {
	isConfirmed      bool
	content          string
	ready            bool
	viewport         viewport.Model
	confirmationForm huh.Form
}

func (m *MainModel) handleConfirmationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func setupPods(m *MainModel) {
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

func (m MainModel) handleConfirmationView() string {
	helpMsg := fmt.Sprintf("%s\n%s\n",
		helpStyle.Render("\nj/k: down, up • ctrl+d/u: half page down, up"),
		helpStyle.Render("\nb: go back to configuration • ctrl+c: quit"),
	)
	conf := ""
	viewPortPosition := int(m.setupConfirmation.viewport.ScrollPercent() * 100)

	if viewPortPosition > 98 {
		conf = "\n" + m.setupConfirmation.confirmationForm.View()
	}

	if m.setupConfirmation.isConfirmed {
		conf += "\n" + alertStyle.Render("Configuration confirmed! Press 'c' to continue")
	}

	return fmt.Sprintf("%s\n%s\n%s\n",
		m.setupConfirmation.viewport.View(),
		conf,
		helpMsg)
}

func (m MainModel) InitConfirmation() ConfirmationModel {
	vp := viewport.New(70, viewportHeight)
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
		b.WriteString(configInfoStyle.
			Render("\n------------------" + podLabel + "--------------------------"))
		b.WriteString(configInfoStyle.
			Render("\nScenario file :" + pod.scenarioFilePath))
		b.WriteString(configInfoStyle.
			Render("\nProperties file: " + pod.propsFilePath))

		b.WriteString("\n--------------------------PROPS-----------------------------")
		for _, l := range readFile(pod.propsFilePath) {
			l = strings.TrimSpace(l)
			b.WriteString(propsStyle.Render("\n" + l))
		}
	}

	return b.String()
}

func (m MainModel) GetConfirmationDialog() *huh.Confirm {
	return huh.NewConfirm().
		Title(accentInfo.Render("Do you want to proceed with this config?")).
		Affirmative("Yes").
		Negative("No").
		Key("conf").
		Value(&m.setupConfirmation.isConfirmed)
}
