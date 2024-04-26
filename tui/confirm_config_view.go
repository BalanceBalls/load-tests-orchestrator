package tui

import (
	"fmt"
	"strconv"
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

	subtle = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}

	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	confirmationInfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

const useHighPerformanceRenderer = false

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

	sc := m.setupConfirmation
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "b":
			m.currentView = PodsSetup
		case "c":
			if m.setupConfirmation.isConfirmed {
				setupPods(m)
				return m, m.preparation.spinner.Tick
			}
		}
	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(sc.headerView())
		footerHeight := lipgloss.Height(sc.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		m.setupConfirmation.viewport.Width = msg.Width
		m.setupConfirmation.viewport.Height = msg.Height - verticalMarginHeight

		if useHighPerformanceRenderer {
			cmds = append(cmds, viewport.Sync(m.setupConfirmation.viewport))
		}
	}

	cf, fCmd := m.setupConfirmation.confirmationForm.Update(msg)
	if f, ok := cf.(*huh.Form); ok {
		m.setupConfirmation.confirmationForm = *f
		if f.GetBool("conf") {
			m.setupConfirmation.isConfirmed = true
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

	if m.setupConfirmation.confirmationForm.GetBool("conf") {
		conf += "\n" + alertStyle.Render("Configuration confirmed! Press 'c' to continue")
	}

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
		m.setupConfirmation.headerView(),
		m.setupConfirmation.viewport.View(),
		conf,
		m.setupConfirmation.footerView(),
		helpMsg)
}

func (m ConfirmationModel) headerView() string {
	title := titleStyle.Render("Config review")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m ConfirmationModel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m MainModel) InitConfirmation() ConfirmationModel {
	vp := viewport.New(70, 30)
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
		b.WriteString(configInfoStyle.Render("\nPod name: " + pod.name))
		b.WriteString(configInfoStyle.Render("\nScenario file :" + pod.scenarioFilePath))
		b.WriteString(configInfoStyle.Render("\nProperties file: " + pod.propsFilePath))

		b.WriteString("\n--------------------------PROPS-----------------------------")
		for i, l := range readFile(pod.propsFilePath) {
			ln := strconv.Itoa(i)
			b.WriteString(configuredStyle.Render("\n  |" + ln + ". " + l))
		}
		b.WriteString("\n////////////////////////////////////////////////////////////")
	}

	return b.String()
}

func (m MainModel) GetConfirmationDialog() *huh.Confirm {
	return huh.NewConfirm().
		Title("Do you want to proceed with this config?").
		Affirmative("Yes").
		Negative("No").
		Key("conf").
		Value(&m.setupConfirmation.isConfirmed)
}
