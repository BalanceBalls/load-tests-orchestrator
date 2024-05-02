package tui

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	NotStarted int = iota
	StartConfirm
	InProgress
	Completed
	CancelConfirm
	Cancelled
	ResetConfirm
	Collected
)

type RunPodInfo struct {
	PodInfo

	runState   int
	err        error
	resultPath string
}

type TestRunModel struct {
	runState     int
	namespace    string
	pods         []RunPodInfo
	isTableView  bool
	currentPod   int
	podViews     []viewport.Model
	pages        paginator.Model
	confirm      huh.Form
	spinner      spinner.Model
	showSpinner  bool
	isConfirmed  bool
	showConfirm  bool
	prevRunState *int
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
var borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

var (
	notStartedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	inProgressStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	completedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

func (m *MainModel) handleRunView() string {
	var b strings.Builder
	b.WriteString(focusedStyle.Render("\nLoad test orchestrator"))
	if m.run.showSpinner {
		b.WriteString(" | " + m.run.spinner.View())
	}

	b.WriteString("\n\n")

	if m.run.showConfirm {
		b.WriteString(m.run.confirm.View())
		return b.String()
	}

	if m.run.isTableView {
		b.WriteString("\n" + getPodsTable(m.run.pods).Render())
	} else {
		b.WriteString("\n" + m.run.podViews[m.run.currentPod].View())
	}

	b.WriteString("\nCurrent run state: " + getState(m.run.runState))
	b.WriteString(configInfoStyle.Render("\nNamespace: " + m.run.namespace))

	start, end := m.run.pages.GetSliceBounds(len(m.run.pods))
	for _, item := range m.run.pods[start:end] {
		b.WriteString("\nPod: " + podLabelStyle.Render(item.name))
		b.WriteString("\n" + m.run.pages.View())
	}

	b.WriteString(helpStyle.Render("\n\nctrl+s: start run • ctrl+k: cancel ongoing run "))
	b.WriteString(helpStyle.Render("\nctrl+r: reset run to initial state (stop current run and clear files)"))
	b.WriteString(helpStyle.Render("\nh/l ←/→ page • ctrl+c: quit"))
	b.WriteString("\n\n")
	return b.String()
}

func (m *MainModel) handleRunUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		podViewCmd tea.Cmd
		formCmd    tea.Cmd
		pagesCmd   tea.Cmd
		spinnerCmd tea.Cmd
		cmds       []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		changesApplied := m.run.handleKeyUpdates(msg)
		if changesApplied {
			return m, m.run.spinner.Tick
		}
	}

	if m.run.showConfirm {
		var confirmModel tea.Model
		confirmModel, formCmd = m.run.confirm.Update(msg)
		cmds = append(cmds, formCmd)
		m.run.handleConfirmationResult(confirmModel)
		cmds = append(cmds, m.run.spinner.Tick)
	} else {
		for i := range len(m.run.pods) {
			m.run.podViews[i], podViewCmd = m.run.podViews[i].Update(msg)
			cmds = append(cmds, podViewCmd)
		}

		var updSpinner spinner.Model
		updSpinner, spinnerCmd = m.run.spinner.Update(msg)
		m.run.spinner = updSpinner
		cmds = append(cmds, spinnerCmd)
		cmds = append(cmds, m.run.spinner.Tick)

		var updatedPaginator paginator.Model
		updatedPaginator, pagesCmd = m.run.pages.Update(msg)
		cmds = append(cmds, pagesCmd)
		m.run.currentPod = updatedPaginator.Page
		m.run.pages = updatedPaginator
	}

	return m, tea.Batch(cmds...)
}

func (m *TestRunModel) handleKeyUpdates(msg tea.KeyMsg) bool {
	if m.showConfirm {
		return false
	}

	switch msg.String() {
	case "ctrl+s":
		if m.runState == NotStarted {
			m.runState = StartConfirm
			m.confirm = *huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return true
	case "ctrl+k":
		if m.runState == InProgress {
			m.runState = CancelConfirm
			m.confirm = *huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return true
	case "ctrl+r":
		switch m.runState {
		case InProgress, Cancelled, Collected:
			prev := m.runState
			m.prevRunState = &prev
			m.runState = ResetConfirm
			m.confirm = *huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))
			m.showConfirm = true
		}
		return true
	case "v":
		m.isTableView = !m.isTableView
		return true
	}

	return false
}

func (m *MainModel) InitRunView() *TestRunModel {
	podCnt, _ := strconv.Atoi(m.configForm.inputs[2].Value())
	var testPods []RunPodInfo
	var logsViewers []viewport.Model

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")
	p.SetTotalPages(podCnt)

	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Spinner.FPS = 200 * time.Millisecond
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("155"))

	confirmationForm := huh.NewForm(huh.NewGroup(m.GetConfirmationDialog()))

	for i := range podCnt {
		vp := viewport.New(120, viewportHeight)
		vp.MouseWheelEnabled = true
		vp.SetContent("Run is not started. No logs yet for pod " + strconv.Itoa(i))
		logsViewers = append(logsViewers, vp)

		tPod := RunPodInfo{
			PodInfo: PodInfo{
				id:               m.pods[i].id,
				name:             m.pods[i].name,
				scenarioFilePath: m.pods[i].scenarioFilePath,
				propsFilePath:    m.pods[i].propsFilePath,
			},
			runState:   NotStarted,
			err:        nil,
			resultPath: "",
		}
		testPods = append(testPods, tPod)
	}

	return &TestRunModel{
		runState:    NotStarted,
		namespace:   m.configForm.inputs[1].Value(),
		pods:        testPods,
		currentPod:  0,
		podViews:    logsViewers,
		pages:       p,
		isTableView: true,
		confirm:     *confirmationForm,
		spinner:     s,
	}
}

func (m *TestRunModel) startRun() {
	m.runState = InProgress
	m.showSpinner = true
	for i := range m.pods {

		// Perform a command to start jmeter test
		// Then either set state to InProgress or set an error
		m.pods[i].runState = InProgress
	}
}

func (m *TestRunModel) checkIfRunComplete() {
	for i := range m.pods {
		// Check if a pod is finished the run (just check if an archive is generated)
		m.pods[i].runState = InProgress // Completed
	}

	runInProgress := slices.ContainsFunc(m.pods, func(p RunPodInfo) bool {
		return p.runState != Completed
	})

	if runInProgress {
		return
	}

	m.runState = Completed
	m.showSpinner = false
}

func (m *TestRunModel) cancelRun() {
	for i := range m.pods {
		// Send command to cancel run
		m.pods[i].runState = Cancelled
	}

	m.runState = Cancelled
	m.showSpinner = false
}

func (m *TestRunModel) collectResults() {
	for i := range m.pods {
		// Download results from Pod if any
		m.pods[i].runState = Collected // Completed
	}

	resultCollectionInProgress := slices.ContainsFunc(m.pods, func(p RunPodInfo) bool {
		return p.runState != Collected
	})

	if resultCollectionInProgress {
		return
	}

	m.runState = Collected
}

func (m *TestRunModel) resetRun() {
	for i := range m.pods {
		// Stop current run
		// Clear files
		m.pods[i].runState = NotStarted
	}

	m.runState = NotStarted
	m.showSpinner = false
}

func getState(state int) string {
	stateStr := ""
	switch state {
	case NotStarted:
		stateStr = notStartedStyle.Render("run not started")
	case InProgress:
		stateStr = inProgressStyle.Render("run in progress")
	case Completed:
		stateStr = completedStyle.Render("run completed")
	case Collected:
		stateStr = completedStyle.Render("run results collected")
	case Cancelled:
		stateStr = accentInfo.Render("run is cancelled")
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
		tRow := []string{row.name, getState(row.runState), rowErr}
		rows[i] = tRow
	}

	return rows
}

func getPodsTable(pods []RunPodInfo) *table.Table {
	rows := getTableRows(pods)
	t := table.New().
		Border(lipgloss.ThickBorder()).
		BorderStyle(borderStyle).
		BorderRow(true).
		Headers("Pod", "State", "Error").
		Width(100).
		Rows(rows...)

	return t
}

func (m *TestRunModel) handleConfirmationResult(cf tea.Model) {
	if f, ok := cf.(*huh.Form); ok {
		m.confirm = *f

		if m.confirm.State == huh.StateCompleted {
			if f.GetBool("conf") {
				m.isConfirmed = true
				m.showConfirm = false

				switch m.runState {
				case StartConfirm:
					go m.startRun()
				case CancelConfirm:
					go m.cancelRun()
				case ResetConfirm:
					go m.resetRun()
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
		msg = "Do you want to stop current run and reset pods for a new run?"
	}

	return huh.NewConfirm().
		Title(accentInfo.Render(msg)).
		Affirmative("Yes").
		Negative("No").
		Key("conf").
		Value(&m.isConfirmed)
}
