package tui

import (
	"slices"
	"terminalui/kubeutils"
)

func (m *ConfiguratorModel) startRun() {
	m.run.runState = InProgress
	m.run.showSpinner = true
	for i, pod := range m.run.pods {

		testInfo := kubeutils.TestInfo{
			PodName:          pod.name,
			PropFileName:     pod.propsFilePath,
			ScenarioFileName: pod.scenarioFilePath,
		}
		err := m.cluster.KickstartTestForPod(m.ctx, testInfo)	
		if err != nil {
			// Display err
		}
		m.run.pods[i].runState = InProgress
	}

	m.run.table = getPodsTable(m.run.pods)
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

	m.table = getPodsTable(m.pods)
	m.runState = Completed
	m.showSpinner = false
}

func (m *TestRunModel) cancelRun() {
	for i := range m.pods {
		// Send command to cancel run
		m.pods[i].runState = Cancelled
	}

	m.table = getPodsTable(m.pods)

	m.runState = Cancelled
	m.showSpinner = false
}

func (m *TestRunModel) finishRun() {
	for i := range m.pods {
		// Download results from Pod if any
		m.pods[i].runState = Completed // Completed
	}

	runStillInProgress := slices.ContainsFunc(m.pods, func(p RunPodInfo) bool {
		return p.runState != Completed
	})

	if runStillInProgress {
		return
	}

	m.runState = Completed
}

func (m *TestRunModel) resetRun() {
	for i := range m.pods {
		// Stop current run
		// Clear files
		m.pods[i].runState = NotStarted
	}

	m.table = getPodsTable(m.pods)
	m.runState = NotStarted
	m.showSpinner = false
}

func collectResults(m *ConfiguratorModel) {
	rp := m.InitResultsPreparation()
	m.resultsCollection = rp
	m.currentView = Collect

	go func() {
		ch := make(chan kubeutils.ActionDone)
		defer close(ch)
		go m.resultsCollection.saveResults(ch)

		for r := range ch {
			m.Update(r)
			m.Update(m.resultsCollection.spinner.Tick)
		}
	}()
}
