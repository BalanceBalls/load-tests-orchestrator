package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

func (cm *ConfiguratorModel) handleRunView() string {
	m := cm.run
	var b strings.Builder
	b.WriteString(focusedStyle.Render("\nLoad test orchestrator"))
	if m.showSpinner {
		b.WriteString(" | " + m.spinner.View())
	}

	b.WriteString("\n\n")

	if m.showConfirm {
		b.WriteString(m.confirm.View())
		return b.String()
	}

	if m.isTableView {
		b.WriteString("\n" + m.table)
	} else {
		b.WriteString(podLogsStyle.Render("\n" + m.podViews[m.currentPod].View()))
	}

	b.WriteString("\nCurrent run state: " + m.runState.String())
	b.WriteString(configInfoStyle.Render("\nNamespace: " + m.namespace))

	start, end := m.pages.GetSliceBounds(len(m.pods))
	for _, item := range m.pods[start:end] {
		b.WriteString("\nPod: " + podLabelStyle.Render(item.name))
		b.WriteString("\n" + m.pages.View())
	}

	b.WriteString(getHelpText())
	return b.String()
}

func (cm *ConfiguratorModel) handleRunViewUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	m := cm.run
	var (
		formCmd    tea.Cmd
		pagesCmd   tea.Cmd
		spinnerCmd tea.Cmd
		cmds       []tea.Cmd
	)

	if m.runState == Done {
		collectResults(cm)
		return cm, cm.resultsCollection.spinner.Tick
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return cm, tea.Quit
		}
		changeCmd := m.handleKeyUpdates(msg)
		if changeCmd != nil {
			return cm, changeCmd
		}
	case spinner.TickMsg:
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	if m.showConfirm {
		var confirmModel tea.Model
		confirmModel, formCmd = m.confirm.Update(msg)
		cmds = append(cmds, formCmd)
		cm.handleConfirmationResult(confirmModel)
		cmds = append(cmds, m.spinner.Tick)
	} else {
		var updatedPaginator paginator.Model
		updatedPaginator, pagesCmd = m.pages.Update(msg)
		cmds = append(cmds, pagesCmd)
		m.currentPod = updatedPaginator.Page
		m.pages = updatedPaginator

		if !m.isTableView {
			m.podViews[m.currentPod].SetContent(m.pods[m.currentPod].data.logs)
			updatedPodView, podViewCmd := m.podViews[m.currentPod].Update(msg)
			m.podViews[m.currentPod] = updatedPodView

			cmds = append(cmds, podViewCmd)
		}
	}

	return cm, tea.Batch(cmds...)
}

func (m *TestRunModel) handleKeyUpdates(msg tea.KeyMsg) tea.Cmd {
	if m.showConfirm {
		return nil
	}

	switch msg.String() {
	case "ctrl+s":
		if m.runState == NotStarted {
			m.runState = StartConfirm
			m.confirm = huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return m.spinner.Tick
	case "ctrl+k":
		if m.runState == InProgress {
			m.runState = CancelConfirm
			m.confirm = huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return m.spinner.Tick
	case "ctrl+r":
		switch m.runState {
		case Completed, Cancelled:
			prev := m.runState
			m.prevRunState = &prev
			m.runState = ResetConfirm
			m.confirm = huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return m.spinner.Tick
	case "v":
		m.isTableView = !m.isTableView
		return m.spinner.Tick
	case "d":
		m.podViews[m.currentPod].GotoBottom()
		return m.spinner.Tick
	}

	return nil
}

func (m *TestRunModel) Init() tea.Cmd {
	return tea.Batch(tea.ClearScreen, m.spinner.Tick)
}

func (m *ConfiguratorModel) InitRunView() *TestRunModel {
	podsAmount, _ := strconv.Atoi(m.configForm.inputs[3].Value())
	namespace := m.configForm.inputs[1].Value()

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")

	p.SetTotalPages(podsAmount)

	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("155"))

	var loadTestPods []RunPodInfo
	var podViews []viewport.Model
	for i := range podsAmount {
		tPod := RunPodInfo{
			PodInfo: PodInfo{
				id:               m.pods[i].id,
				name:             m.pods[i].name,
				scenarioFilePath: m.pods[i].scenarioFilePath,
				propsFilePath:    m.pods[i].propsFilePath,
				data:             PodLogs{logs: "no logs yet"},
			},
			runState:   NotStarted,
			err:        nil,
			resultPath: "",
		}
		vp := viewport.New(200, viewportHeight)
		vp.MouseWheelEnabled = true

		podViews = append(podViews, vp)
		loadTestPods = append(loadTestPods, tPod)
	}

	runModel := &TestRunModel{
		runState:    NotStarted,
		namespace:   namespace,
		pods:        loadTestPods,
		isTableView: true,
		currentPod:  0,
		podViews:    podViews,
		pages:       p,
		spinner:     s,
		showSpinner: false,
		isConfirmed: false,
		showConfirm: false,
	}

	runModel.table = getPodsTable(runModel.pods)
	confirmationForm := huh.NewForm(huh.NewGroup(runModel.getConfirmationDialog()))
	runModel.confirm = confirmationForm

	return runModel
}

func (state TestRunState) String() string {
	stateStr := ""
	switch state {
	case NotStarted:
		stateStr = notStartedStyle.Render("run not started")
	case InProgress:
		stateStr = inProgressStyle.Render("run in progress")
	case Completed:
		stateStr = completedStyle.Render("run completed")
	case Cancelled:
		stateStr = accentInfo.Render("run is cancelled")
	case Failed:
		stateStr = accentInfo.Render("run failed")
	default:
		stateStr = "Unknown state"
	}

	return stateStr
}

func getTableRows(pods []RunPodInfo) [][]string {
	rows := make([][]string, len(pods))
	for i, row := range pods {
		rowErr := "-"
		if row.err != nil {
			rowErr = row.err.Error()
		}
		tRow := []string{row.name, row.runState.String(), rowErr}
		rows[i] = tRow
	}

	return rows
}

func getPodsTable(pods []RunPodInfo) string {
	rows := getTableRows(pods)
	t := table.New().
		Border(lipgloss.ThickBorder()).
		BorderStyle(tableBorderStyle).
		BorderRow(true).
		Headers("Pod", "State", "Error").
		Width(100).
		Rows(rows...)

	return t.Render()
}

func (cm *ConfiguratorModel) handleConfirmationResult(cf tea.Model) {
	m := cm.run
	if f, ok := cf.(*huh.Form); ok {
		m.confirm = f

		if m.confirm.State == huh.StateCompleted {
			if f.GetBool("conf") {
				m.isConfirmed = true
				m.showConfirm = false

				switch m.runState {
				case StartConfirm:
					go cm.startRun()
				case CancelConfirm:
					go cm.cancelRun()
				case ResetConfirm:
					go cm.resetRun()
				}
			} else {
				m.isConfirmed = false
				m.showConfirm = false

				switch m.runState {
				case StartConfirm:
					m.runState = NotStarted
				case CancelConfirm:
					m.runState = InProgress
				case ResetConfirm:
					m.runState = *m.prevRunState
					m.prevRunState = nil
				}
			}
		}
	}
}

func (m *TestRunModel) getConfirmationDialog() *huh.Confirm {
	msg := ""
	switch m.runState {
	case StartConfirm:
		msg = "Do you want to start a load test run?"
	case CancelConfirm:
		msg = "Do you want to stop current load test run?"
	case ResetConfirm:
		msg = "Do you want to reset pods for a new run?"
	}

	return huh.NewConfirm().
		Title(accentInfo.Render(msg)).
		Affirmative("Yes").
		Negative("No").
		Key("conf").
		Value(&m.isConfirmed)
}

func getHelpText() string {
	var b strings.Builder
	b.WriteString(helpStyle.Render("\n\nctrl+s: start run • ctrl+k: cancel ongoing run "))
	b.WriteString(helpStyle.Render("\nctrl+r: reset run to initial state (remove files produced by previous run)"))
	b.WriteString(helpStyle.Render("\nd: scroll logs to bottom • v: switch between table and logs views"))
	b.WriteString(helpStyle.Render("\nctrl+d: scroll half page down • ctrl+u: scroll half page up"))
	b.WriteString(helpStyle.Render("\nh/l ←/→ page • ctrl+c: quit"))
	b.WriteString("\n\n")
	return b.String()
}
