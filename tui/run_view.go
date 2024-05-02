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

func (m *TestRunModel) View() string {
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
		b.WriteString("\n" + getPodsTable(m.pods).Render())
	} else {
		b.WriteString("\n" + m.podViews[m.currentPod].View())
	}

	b.WriteString("\nCurrent run state: " + getState(m.runState))
	b.WriteString(configInfoStyle.Render("\nNamespace: " + m.namespace))

	start, end := m.pages.GetSliceBounds(len(m.pods))
	for _, item := range m.pods[start:end] {
		b.WriteString("\nPod: " + podLabelStyle.Render(item.name))
		b.WriteString("\n" + m.pages.View())
	}

	b.WriteString(helpStyle.Render("\n\nctrl+s: start run • ctrl+k: cancel ongoing run "))
	b.WriteString(helpStyle.Render("\nctrl+r: reset run to initial state (stop current run and clear files)"))
	b.WriteString(helpStyle.Render("\nh/l ←/→ page • ctrl+c: quit"))
	b.WriteString("\n\n")
	return b.String()
}

func (m *TestRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		podViewCmd tea.Cmd
		formCmd    tea.Cmd
		pagesCmd   tea.Cmd
		spinnerCmd tea.Cmd
		cmds       []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c"{
			return m, tea.Quit
		}
		changesApplied := m.handleKeyUpdates(msg)
		if changesApplied {
			return m, m.spinner.Tick
		}
	}

	if m.showConfirm {
		var confirmModel tea.Model
		confirmModel, formCmd = m.confirm.Update(msg)
		cmds = append(cmds, formCmd)
		m.handleConfirmationResult(confirmModel)
		cmds = append(cmds, m.spinner.Tick)
	} else {
		for i := range len(m.pods) {
			m.podViews[i], podViewCmd = m.podViews[i].Update(msg)
			cmds = append(cmds, podViewCmd)
		}

		var updSpinner spinner.Model
		updSpinner, spinnerCmd = m.spinner.Update(msg)
		m.spinner = updSpinner
		cmds = append(cmds, spinnerCmd)
		cmds = append(cmds, m.spinner.Tick)

		var updatedPaginator paginator.Model
		updatedPaginator, pagesCmd = m.pages.Update(msg)
		cmds = append(cmds, pagesCmd)
		m.currentPod = updatedPaginator.Page
		m.pages = updatedPaginator
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

func (m *TestRunModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *TestRunModel) InitRunView(cfg RunConfigData) {
	var logsViewers []viewport.Model

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")

	p.SetTotalPages(cfg.podsAmount)

	s := spinner.New()
	s.Spinner = spinner.Meter
	s.Spinner.FPS = 200 * time.Millisecond
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("155"))

	confirmationForm := huh.NewForm(huh.NewGroup(m.getConfirmationDialog()))

	for i := range cfg.podsAmount {
		vp := viewport.New(120, viewportHeight)
		vp.MouseWheelEnabled = true
		vp.SetContent("Run is not started. No logs yet for pod " + strconv.Itoa(i))
		logsViewers = append(logsViewers, vp)
	}

	m.runState = NotStarted
	m.namespace = cfg.namespace
	m.pods = cfg.pods
	m.currentPod = 0
	m.podViews = logsViewers
	m.pages = p
	m.isTableView = true
	m.confirm = *confirmationForm
	m.spinner = s
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
