package tui

import (
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

const (
	NotStarted int = iota
	InProgress
	Completed
	Cancelled
	Collected
)

type RunPodInfo struct {
	PodInfo

	runState   int
	err        error
	resultPath string
}

type TestRunModel struct {
	runState    int
	namespace   string
	pods        []RunPodInfo
	isTableView bool

	currentPod int
	podViews   []*viewport.Model
	pages      paginator.Model
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))
var borderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

func (m *MainModel) handleRunView() string {
	var b strings.Builder
	b.WriteString(focusedStyle.Render("\nLoad test orchestrator\n"))
	if m.run.isTableView {
		b.WriteString("\n" + getPodsTable(m.run.pods).Render())
	} else {
		b.WriteString("HERE WILL BE A VIEWPORT")
	}
	b.WriteString("\nCurrent run state: " + getState(m.run.runState))
	b.WriteString(configInfoStyle.Render("\nNamespace: " + m.run.namespace))

	start, end := m.run.pages.GetSliceBounds(len(m.run.pods))
	for _, item := range m.run.pods[start:end] {
		b.WriteString("\nPod: " + podLabelStyle.Render(item.name))
		b.WriteString("\n" + m.run.pages.View())

		b.WriteString(helpStyle.Render("\n\nctrl+s: start run • ctrl+k: cancel ongoing run "))
		b.WriteString(helpStyle.Render("\nctrl+r: reset run to initial state (stop current run and clear files)"))
		b.WriteString(helpStyle.Render("\nh/l ←/→ page • ctrl+c: quit"))
	}

	b.WriteString("\n\n")
	return b.String()
}

func (m *MainModel) handleRunUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			m.run.startRun()
			return m, nil
		case "ctrl+k":
			m.run.cancelRun()
			return m, nil
		case "ctrl+r":
			m.run.resetRun()
			return m, nil
		case "v":
			m.run.isTableView = !m.run.isTableView
			return m, nil
		}
	}

	updatedPaginator, cmdP := m.run.pages.Update(msg)
	m.run.pages = updatedPaginator
	return m, cmdP
}

func (m *MainModel) InitRunView() *TestRunModel {
	podCnt, _ := strconv.Atoi(m.configForm.inputs[2].Value())
	var testPods []RunPodInfo

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")
	p.SetTotalPages(podCnt)

	for i := range podCnt {
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
		podViews:    []*viewport.Model{},
		pages:       p,
		isTableView: true,
	}
}

func (m *TestRunModel) startRun() {
	m.runState = InProgress
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
}

func (m *TestRunModel) cancelRun() {
	for i := range m.pods {
		// Send command to cancel run
		m.pods[i].runState = Cancelled
	}

	m.runState = Cancelled
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
}

func getState(state int) string {
	switch state {
	case NotStarted:
		return "run not started"
	case InProgress:
		return "run in progress"
	case Completed:
		return "run completed"
	case Collected:
		return "run results collected"
	case Cancelled:
		return "run is cancelled"
	default:
		return "Unknown state"
	}
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
