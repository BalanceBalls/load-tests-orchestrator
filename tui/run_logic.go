package tui

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"terminalui/kubeutils"
	"time"
)

const staleThreshold = 5

var staleErr = errors.New("Test is likely failed to finish. Check pod")

type PodUpdate struct {
	podIndex     int
	logs         string
	staleCounter int
	inProgress   bool
	state        TestRunState
	err          error
}

func (m *ConfiguratorModel) startRun() {
	m.run.runState = InProgress
	m.run.showSpinner = true
	for i, pod := range m.run.pods {
		_, propFile := filepath.Split(pod.propsFilePath)
		_, jmxFile := filepath.Split(pod.scenarioFilePath)

		testInfo := kubeutils.TestInfo{
			PodName:          pod.name,
			PropFileName:     propFile,
			ScenarioFileName: jmxFile,
		}
		err := m.cluster.KickstartTestForPod(m.ctx, testInfo)
		if err != nil {
			m.run.pods[i].err = err
		}
		m.run.pods[i].runState = InProgress
	}

	m.run.table = getPodsTable(m.run.pods)

	duration := 3 * time.Second
	ticker := time.NewTicker(duration)
	updChannel := make(chan PodUpdate)
	isDone := false

free:
	for !isDone {
		select {
		case <-ticker.C:
			m.logger.Info("TICK")
			if m.run.runState == InProgress {
				go m.checkIfRunComplete(m.ctx, m.run.pods, updChannel)
			} else {
				isDone = true
				break free
			}
		case upd := <-updChannel:
			m.logger.Info("Pod update event: ",
				slog.Any("pod", m.run.pods[upd.podIndex].name),
				slog.Any("state", upd.state),
				slog.Any("stale for", upd.staleCounter))

			m.run.pods[upd.podIndex].data.logs = upd.logs
			m.run.pods[upd.podIndex].data.staleFor = upd.staleCounter
			m.run.pods[upd.podIndex].runState = upd.state
			m.run.pods[upd.podIndex].err = upd.err

			m.run.table = getPodsTable(m.run.pods)

			runIsFinished := true
			runHasFailedTests := false
			for _, pod := range m.run.pods {
				if pod.runState == Failed {
					runHasFailedTests = true
				}
				if pod.runState != Completed && pod.runState != Failed {
					runIsFinished = false
					break
				}
			}

			if runIsFinished {
				if runHasFailedTests {
					m.run.runState = Failed
				} else {
					m.run.runState = Completed
				}
			}
		}
	}

	m.logger.Info("RUN COMPLETE")

	m.run.runState = Completed
	m.run.showSpinner = false
}

func (m ConfiguratorModel) checkIfRunComplete(ctx context.Context, pods []RunPodInfo, ch chan<- PodUpdate) {
	for i, pod := range pods {
		podUpd := PodUpdate{podIndex: i, inProgress: true, state: InProgress}

		testInfo := kubeutils.TestInfo{
			PodName: pod.name,
		}
		isFinished, logs, err := m.cluster.CheckProgress(ctx, testInfo)
		if err != nil {
			podUpd.err = err
		}

		if strings.EqualFold(pod.data.logs, logs) {
			podUpd.staleCounter += 1
		}

		if pod.data.staleFor > staleThreshold {
			// cPod.err = staleErr
			podUpd.inProgress = false
			podUpd.state = Failed
		}

		podUpd.logs = logs
		if isFinished {
			podUpd.inProgress = false
			podUpd.state = Completed
		}

		ch <- podUpd
	}
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
