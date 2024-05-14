package tui

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"terminalui/kubeutils"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const staleThreshold = 5

var errStale = errors.New("test is likely failed to finish. Check pod")

func (m *ConfiguratorModel) getClusterConfig(kubeCtx string) (*kubeutils.Cluster, error) {
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, ".kube", "config")

	config, err := kubeutils.BuildConfigWithContextFromFlags(kubeCtx, defaultPath)
	if err != nil {
		m.logger.Error("error creating Kubernetes client configuration: ", slog.Any("err", err))
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		m.logger.Error("error creating Kubernetes client: ", slog.Any("err", err))
		return nil, err
	}

	cluster := kubeutils.Cluster{
		RestCfg:         config,
		Clientset:       clientset,
		PodPrefix:       m.configForm.inputs[0].Value(),
		Namespace:       m.configForm.inputs[1].Value(),
		KubeCtxName:     m.configForm.inputs[2].Value(),
		PodKeepAliveSec: m.podKeepAliveSec,
		PodsCache: &kubeutils.PodsCache{
			Pods: make(map[string]*v1.Pod, len(m.pods)),
		},
		Logger: *m.logger,
	}
	return &cluster, nil
}

func (m *ConfiguratorModel) checkClusterConnection(ch chan<- ConfigDone) {
	isConnected, err := m.cluster.Ping(m.ctx)
	if err != nil {
		m.configForm.err = err
	}

	m.configForm.connectionEstablished = isConnected
	ch <- ConfigDone{ConnectionOk: isConnected}
}

func (m *ConfiguratorModel) setupPods() {
	pf := m.InitPodsPreparation()
	m.preparation = pf
	m.currentView = PreparePods

	go func() {
		ch := make(chan kubeutils.ActionDone)
		defer close(ch)
		go m.beginPodsPreparation(ch)

		for r := range ch {
			m.Update(r)
			m.Update(m.preparation.spinner.Tick)
		}
	}()
}

func (m *ConfiguratorModel) beginPodsPreparation(ch chan<- kubeutils.ActionDone) {
	var wg sync.WaitGroup
	for _, pod := range m.preparation.pods {
		wg.Add(1)
		if m.preparation.err != "" {
			break
		}
		go func(p PodInfo) {
			defer wg.Done()
			testInfo := kubeutils.TestInfo{
				PodName:          p.name,
				PropFileName:     p.propsFilePath,
				ScenarioFileName: p.scenarioFilePath,
			}
			err := m.cluster.PreparePod(m.preparation.ctx, testInfo, ch)
			if err != nil {
				m.preparation.err = err.Error()
				m.preparation.quitting = true
				return
			}
		}(pod)
	}
	wg.Wait()
	m.preparation.quitting = true
}

func (m *ConfiguratorModel) collectResults() {
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

func (m *ConfiguratorModel) saveResults(ch chan<- kubeutils.ActionDone) {
	var wg sync.WaitGroup
	for _, pod := range m.run.pods {
		wg.Add(1)
		go func(p RunPodInfo) {
			defer wg.Done()

			testInfo := kubeutils.TestInfo{
				PodName: p.name,
			}
			err := m.cluster.CollectResultsFromPod(m.preparation.ctx, testInfo, ch)
			if err != nil {
				m.resultsCollection.err = err
				return
			}
		}(pod)
	}
	wg.Wait()
	m.resultsCollection.isCollected = true
	m.resultsCollection.showConfirmation = true
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

	duration := time.Duration(m.updateIntervalSec) * time.Second
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
			m.logger.Error(
				"error occured during test progress check for pod.",
				slog.Any("pod", pod.name),
				slog.Any("err", err.Error()))
		}

		if strings.EqualFold(pod.data.logs, logs) || err != nil {
			podUpd.staleCounter += 1
			m.logger.Warn("stale counter increased", slog.Any("pod", pod.name), slog.Any("cnt", podUpd.staleCounter))
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

func (m *ConfiguratorModel) deletePods() {
	m.resultsCollection.showConfirmation = false

	var wg sync.WaitGroup
	for _, pod := range m.run.pods {
		wg.Add(1)
		go func(p RunPodInfo) {
			defer wg.Done()
			deleteStart := time.Now()
			err := m.cluster.DeletePod(m.resultsCollection.ctx, p.name)
			if err != nil {
				m.logger.Error(err.Error())
				m.resultsCollection.err = err
				return
			}
			m.logger.Info("remove pod goroutine complete")

			result := kubeutils.ActionDone{
				PodName:  p.name,
				Name:     "pod has been terminated",
				Duration: time.Since(deleteStart),
			}
			m.Update(result)
		}(pod)
	}

	wg.Wait()
	m.resultsCollection.quitting = true
}
