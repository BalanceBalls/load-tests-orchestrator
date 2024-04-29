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
	pages      *paginator.Model
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
		b.WriteString("\n" + m.paginator.View())

		b.WriteString(helpStyle.Render("\n\ns: pick scenario file • p: pick properties file"))
		b.WriteString(helpStyle.Render("\nc: continue with current config"))
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
		case "v":
			m.run.isTableView = !m.run.isTableView
		}
	}

	updatedPaginator, cmdP := m.run.pages.Update(msg)
	m.run.pages = &updatedPaginator
	return m, cmdP
}

func (m *MainModel) InitRunView() *TestRunModel {
	podCnt, _ := strconv.Atoi(m.inputs[2].Value())
	var testPods []RunPodInfo

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Bold(true).Render("[+]")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render(" * ")
	p.SetTotalPages(len(m.pods))

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
		namespace:   m.inputs[1].Value(),
		pods:        testPods,
		currentPod:  0,
		podViews:    []*viewport.Model{},
		pages:       &p,
		isTableView: true,
	}
}

func (m *TestRunModel) startRun() {
	m.runState = InProgress
	for _, pod := range m.pods {

		// Perform a command to start jmeter test
		// Then either set state to InProgress or set an error
		pod.runState = InProgress
	}
}

func (m *TestRunModel) checkIfRunComplete() {
	for _, pod := range m.pods {
		// Check if a pod is finished the run (just check if an archive is generated)
		pod.runState = InProgress // Completed
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
	for _, pod := range m.pods {
		// Send command to cancel run
		pod.runState = Cancelled
	}

	m.runState = Cancelled
}

func (m *TestRunModel) collectResults() {
	for _, pod := range m.pods {
		// Download results from Pod if any
		pod.runState = Collected // Completed
	}

	resultCollectionInProgress := slices.ContainsFunc(m.pods, func(p RunPodInfo) bool {
		return p.runState != Collected
	})

	if resultCollectionInProgress {
		return
	}

	m.runState = Collected
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
