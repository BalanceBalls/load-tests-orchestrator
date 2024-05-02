package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	padding  = 2
	maxWidth = 80
)

type tickMsg time.Time
type stepDone struct {
	podName  string
	name     string
	duration time.Duration
}

var (
	podSetupActions = []string{"creating pod", "copying files", "setting up jmeter", "checking jmeter"}
)

var (
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	dotStyle      = helpStyle.Copy().UnsetMargins()
	durationStyle = dotStyle.Copy()
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ebeb13"))
)

type PreparePodsModel struct {
	pods     []PodInfo
	spinner  spinner.Model
	results  []stepDone
	quitting bool
}

func (sd stepDone) String() string {
	if sd.duration == 0 {
		return dotStyle.Render(strings.Repeat(".", 30))
	}
	return fmt.Sprintf("* Pod: %s; Step: %s; Took: %s",
		podLabelStyle.Render(sd.podName),
		stepNameStyle.Render(sd.name),
		durationStyle.Render(sd.duration.String()))
}

func (m *ConfiguratorModel) InitPodsPreparation() *PreparePodsModel {
	numLastResults := 10
	s := spinner.New()
	s.Style = spinnerStyle

	pm := PreparePodsModel{
		spinner: s,
		results: make([]stepDone, numLastResults),
		pods:    m.pods,
	}

	return &pm
}

func (m *ConfiguratorModel) handlePodsPreparationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case stepDone:
		m.preparation.results = append(m.preparation.results[1:], msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.preparation.spinner, cmd = m.preparation.spinner.Update(msg)
		return m, cmd
	default:
		if m.preparation.quitting == true {
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *ConfiguratorModel) handlePodsPreparationView() string {
	var s string

	if m.preparation.quitting {
		s += "Pods are now ready to run load tests!"
	} else {
		s += m.preparation.spinner.View() + " Preparing pods..."
	}

	s += "\n\n"

	for _, res := range m.preparation.results {
		s += res.String() + "\n"
	}

	if !m.preparation.quitting {
		s += helpStyle.Render("Press any key to exit")
	}

	if m.preparation.quitting {
		s += "\n"
		s += alertStyle.Render("Press 'c' to continue... ")
	}

	return appStyle.Render(s)
}

func (m *PreparePodsModel) doStuff() {
	time.Sleep(time.Duration(rand.Intn(800)) * time.Millisecond)
}

func (m *PreparePodsModel) runPodPreparation(pod PodInfo, ch chan<- stepDone) {
	for _, step := range podSetupActions {
		start := time.Now()
		m.doStuff()
		ch <- stepDone{
			podName:  pod.name,
			name:     step,
			duration: time.Since(start),
		}
	}
}

func (pm *PreparePodsModel) beginSetup(ch chan<- stepDone) {
	var wg sync.WaitGroup
	for _, pod := range pm.pods {
		wg.Add(1)
		go func(p PodInfo) {
			defer wg.Done()
			pm.runPodPreparation(p, ch)
		}(pod)
	}
	wg.Wait()
	pm.quitting = true
}
