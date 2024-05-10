package tui

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"terminalui/kubeutils"
	"time"
)

const staleThreshold = 5

var errStale = errors.New("test is likely failed to finish. Check pod")

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

free:
	for {
		select {
		case <-ticker.C:
			m.logger.Info("TICK")
			if m.run.runState == InProgress {
				go m.checkIfRunComplete(m.ctx, m.run.pods, updChannel)
			} else {
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
					m.run.runState = Done
				}
			}
		}
	}

	m.logger.Info("RUN COMPLETE")
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
			podUpd.state = Failed
			podUpd.inProgress = false
		}

		if strings.EqualFold(pod.data.logs, logs) {
			podUpd.staleCounter += 1
		}

		if pod.data.staleFor > staleThreshold {
			podUpd.err = errStale
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

func (m *ConfiguratorModel) cancelRun() {
	for i, pod := range m.pods {
		testInfo := kubeutils.TestInfo{
			PodName: pod.name,
		}
		m.cluster.CancelRunForPod(m.ctx, testInfo)
		m.run.pods[i].runState = Cancelled
	}

	m.run.table = getPodsTable(m.run.pods)

	m.run.runState = Cancelled
	m.run.showSpinner = false
}

func (m *ConfiguratorModel) resetRun() {
	for i, pod := range m.pods {
		testInfo := kubeutils.TestInfo{
			PodName: pod.name,
		}
		m.cluster.ResetPodForNewRun(m.ctx, testInfo)
		m.run.pods[i].runState = NotStarted
		m.run.pods[i].data.logs = "Pod is now ready for a new run"
	}

	m.run.table = getPodsTable(m.run.pods)
	m.run.runState = NotStarted
	m.run.showSpinner = false
}

func collectResults(m *ConfiguratorModel) {
	rp := m.InitResultsPreparation()
	m.resultsCollection = rp
	m.currentView = Collect

	go func() {
		ch := make(chan kubeutils.ActionDone)
		defer close(ch)
		go m.saveResults(ch)

		for r := range ch {
			m.Update(r)
			m.Update(m.resultsCollection.spinner.Tick)
		}
	}()
}
