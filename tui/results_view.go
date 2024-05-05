package tui

import (
	"context"
	"math/rand"
	"strings"
	"sync"
	"terminalui/kubeutils"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var resultCollectionActions = []string{"archive results", "download results"}

func (m *ConfiguratorModel) InitResultsPreparation() *PrepareResultsModel {
	numLastResults := 10
	s := spinner.New()
	s.Style = spinnerStyle

	prepareCtx, cancel := context.WithCancel(m.ctx)

	pm := PrepareResultsModel{
		spinner: s,
		results: make([]kubeutils.ActionDone, numLastResults),
		pods:    m.pods,
		logger:  m.logger,
		ctx:     prepareCtx,
		cancel:  cancel,
	}

	return &pm
}

func (m *ConfiguratorModel) handleResultsPreparationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.resultsCollection.cancel()
			return m, tea.Quit
		}

	case kubeutils.ActionDone:
		m.resultsCollection.results = append(m.resultsCollection.results[1:], msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.resultsCollection.spinner, cmd = m.resultsCollection.spinner.Update(msg)
		return m, cmd
	default:
		if m.resultsCollection.quitting == true {
			time.Sleep(4 * time.Second)
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *ConfiguratorModel) handleResultsPreparationView() string {
	var b strings.Builder

	if m.resultsCollection.quitting {
		b.WriteString("Load test results have been downloaded!")
	} else {
		b.WriteString(m.resultsCollection.spinner.View() + " Preparing results...")
	}

	b.WriteString("\n\n")
	for _, res := range m.resultsCollection.results {
		b.WriteString(formatMsg(res) + "\n")
	}

	if !m.resultsCollection.quitting {
		b.WriteString(helpStyle.Render("Pods are being prepared..."))
	}

	if m.resultsCollection.quitting {
		b.WriteString(alertStyle.Render("\nPress 'ctrl+c' to exit"))
	}

	return appStyle.Render(b.String())
}

func (m *PrepareResultsModel) executeStep() {
	select {
	case <-m.ctx.Done():
		m.logger.Info("Stopped doing stuff")
		return
	default:
		time.Sleep(time.Duration(rand.Intn(800)) * time.Millisecond)
	}
}

func (m *PrepareResultsModel) runPodPreparation(pod PodInfo, ch chan<- kubeutils.ActionDone) {
	for _, step := range resultCollectionActions {
		start := time.Now()
		m.executeStep()
		ch <- kubeutils.ActionDone{
			PodName:  pod.name,
			Name:     step,
			Duration: time.Since(start),
		}
	}
}

func (pm *PrepareResultsModel) saveResults(ch chan<- kubeutils.ActionDone) {
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
