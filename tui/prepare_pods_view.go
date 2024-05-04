package tui

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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

	prepareCtx, cancel := context.WithCancel(m.ctx)

	pm := PreparePodsModel{
		spinner: s,
		results: make([]stepDone, numLastResults),
		pods:    m.pods,
		logger:  m.logger,
		ctx:     prepareCtx,
		cancel:  cancel,
	}

	return &pm
}

func (m *ConfiguratorModel) handlePodsPreparationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.preparation.cancel()
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
			runView := m.InitRunView()
			m.run = runView
			m.currentView = Run
			return m, m.run.spinner.Tick
		}
	}

	return m, nil
}

func (m *ConfiguratorModel) handlePodsPreparationView() string {
	var b strings.Builder

	if m.preparation.quitting {
		b.WriteString("Pods are now ready to run load tests!")
	} else {
		b.WriteString(m.preparation.spinner.View() + " Preparing pods...")
	}

	b.WriteString("\n\n")
	for _, res := range m.preparation.results {
		b.WriteString(res.String() + "\n")
	}

	if !m.preparation.quitting {
		b.WriteString(helpStyle.Render("Pods are being prepared..."))
	}

	if m.preparation.quitting {
		b.WriteString(alertStyle.Render("\nPress 'c' to continue... "))
	}

	return appStyle.Render(b.String())
}

func (m *PreparePodsModel) doStuff() {
	select {
	case <-m.ctx.Done():
		m.logger.Info("Stopped doing stuff")
		return
	default:
		time.Sleep(time.Duration(rand.Intn(800)) * time.Millisecond)
	}
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
