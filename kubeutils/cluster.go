package kubeutils

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *PodsCache) TryGet(ctx context.Context, logger slog.Logger, podName, namespace string, clientset *kubernetes.Clientset) (*v1.Pod, error) {
	if c.Pods[podName] != nil {
		return c.Pods[podName], nil
	}

	logger.Info("pod cache missed. initializing cache value", slog.Any("pod", podName))
	c.mu.Lock()
	defer c.mu.Unlock()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, v1meta.GetOptions{})

	if err != nil {
		return nil, err
	}

	c.Pods[podName] = pod
	return pod, err
}

func (c *Cluster) Ping(ctx context.Context) (bool, error) {
	connected, err := checkClusterConnection(ctx, c.Clientset)
	if err != nil {
		c.Logger.Error(err.Error())
	}
	return connected, err
}

func (c *Cluster) CreatePod(ctx context.Context, podName string) error {
	pod, err := createPod(ctx, c.Clientset, c.Namespace, podName, c.PodKeepAliveSec)
	if err != nil {
		c.Logger.Error("failed to create pod: ", slog.Any("err", err))
		return err
	}

	testCmd := "pwd"
	_, _, err = executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, testCmd)

	return nil
}

func (c *Cluster) PreparePod(ctx context.Context, testInfo TestInfo, ch chan<- ActionDone) error {
	podCreationStart := time.Now()
	pod, err := createPod(ctx, c.Clientset, c.Namespace, testInfo.PodName, c.PodKeepAliveSec)
	if err != nil {
		c.Logger.Error("failed to create pod: ", slog.Any("err", err.Error()))
		return err
	}

	ch <- ActionDone{
		PodName:  testInfo.PodName,
		Name:     "creating pod",
		Duration: time.Since(podCreationStart),
	}

	podCheckStart := time.Now()
	testCmd := "pwd"
	_, _, err = executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, testCmd)

	if err != nil {
		c.Logger.Error(err.Error())
	}

	ch <- ActionDone{
		PodName:  testInfo.PodName,
		Name:     "sending a test command to the pod",
		Duration: time.Since(podCheckStart),
	}

	for _, cmd := range getPodSetupCommands() {
		start := time.Now()
		strBuf, errBuf, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, cmd.command)
		if err != nil {
			c.Logger.Error("failed to execute command: ", slog.Any("err", err.Error()))
			return err
		}

		ch <- ActionDone{
			PodName:  testInfo.PodName,
			Name:     cmd.displayName,
			Duration: time.Since(start),
		}

		c.Logger.Info("command strbuff: " + strBuf)
		c.Logger.Info("command errbuff: " + errBuf)
	}

	cpCmds := getTestUploadCommands(testInfo, c.Namespace)

	switchLocalK8sContext(c.KubeCtxName)
	for _, cpCmd := range cpCmds {
		start := time.Now()
		cpCmd.command.Stdout = os.Stdout
		cpCmd.command.Stderr = os.Stderr

		c.Logger.Info("executing cmd: " + cpCmd.command.String())

		err := cpCmd.command.Run()
		if err != nil {
			c.Logger.Error("failed to copy file to pod: ", slog.Any("err", err.Error()))
			return err
		}

		ch <- ActionDone{
			PodName:  testInfo.PodName,
			Name:     cpCmd.displayName,
			Duration: time.Since(start),
		}
	}

	return nil
}

func (c *Cluster) CheckProgress(ctx context.Context, testInfo TestInfo) (bool, string, error) {
	isFinished := false
	pod, err := c.PodsCache.TryGet(ctx, c.Logger, testInfo.PodName, c.Namespace, c.Clientset)

	if err != nil {
		c.Logger.Error(err.Error())
		isFinished = true
		err = errors.New("failed to reach pod")
		return isFinished, "", err
	}

	stdOut, errOut, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, "cat jmeter/jmeter.log")
	if err != nil {
		c.Logger.Error(err.Error())
		c.Logger.Error(errOut)
		isFinished = true
		err = errors.New("failed to check run state. check pod's health")
		return isFinished, stdOut, err
	}

	finishedRunIndicator := getCheckSuccessfulFinishCommand()
	checkJmeterCmd := getCheckJmeterStateCommand()

	jmeterState, errOut, jErr := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, checkJmeterCmd)
	if jErr != nil {
		c.Logger.Error(jErr.Error())
		c.Logger.Error(errOut)
		isFinished = true
		err = errors.New("failed to check run state. check pod's health")
	}

	c.Logger.Info("jmeter state", slog.String("pod", testInfo.PodName), slog.String("state", jmeterState))
	c.Logger.Info("jmeter state err output", slog.String("pod", testInfo.PodName), slog.String("errOut", errOut))

	jmeterState = strings.TrimSuffix(jmeterState, "\n")
	if jmeterState == "stopped" {
		isFinished = true
		_, _, fErr := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, finishedRunIndicator)
		if fErr != nil {
			err = errors.New("run did not produce results")
		}
	}

	return isFinished, stdOut, err
}

func (c *Cluster) KickstartTestForPod(ctx context.Context, testInfo TestInfo) error {
	pod, err := c.PodsCache.TryGet(ctx, c.Logger, testInfo.PodName, c.Namespace, c.Clientset)

	if err != nil {
		c.Logger.Error(err.Error())
	}

	cmd := getPrepareRunTestCommand(testInfo)
	_, _, err = executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, cmd)

	go executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, getRunTestCommand())
	return err
}

func (c *Cluster) CancelRunForPod(ctx context.Context, testInfo TestInfo) error {
	pod, err := c.PodsCache.TryGet(ctx, c.Logger, testInfo.PodName, c.Namespace, c.Clientset)

	if err != nil {
		c.Logger.Error(err.Error())
	}

	stdOut, _, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, getStopTestCommand())
	c.Logger.Info(stdOut)

	return err
}

func (c *Cluster) ResetPodForNewRun(ctx context.Context, testInfo TestInfo) error {
	pod, err := c.PodsCache.TryGet(ctx, c.Logger, testInfo.PodName, c.Namespace, c.Clientset)

	if err != nil {
		c.Logger.Error(err.Error())
	}

	for _, cmd := range getResetTestCommands() {
		stdOut, _, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, cmd)
		if err != nil {
			return err
		}
		c.Logger.Info(stdOut)
	}

	return err
}

func (c *Cluster) CollectResultsFromPod(ctx context.Context, testInfo TestInfo, ch chan<- ActionDone) error {
	pod, err := c.PodsCache.TryGet(ctx, c.Logger, testInfo.PodName, c.Namespace, c.Clientset)

	if err != nil {
		c.Logger.Error(err.Error())
	}

	packStart := time.Now()
	packResultsCmd := getPackResultsCommand()
	stdOut, _, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, packResultsCmd)
	if err != nil {
		return err
	}
	c.Logger.Info(stdOut)
	ch <- ActionDone{
		PodName:  testInfo.PodName,
		Name:     "pack results into archive",
		Duration: time.Since(packStart),
	}

	downloadStart := time.Now()
	downloadResultsCmd := getDownloadResultsCommand(testInfo, c.Namespace, c.PodPrefix)

	downloadResultsCmd.command.Stdout = os.Stdout
	downloadResultsCmd.command.Stderr = os.Stderr

	c.Logger.Info("executing cmd: " + downloadResultsCmd.command.String())

	err = downloadResultsCmd.command.Run()
	if err != nil {
		c.Logger.Error("failed to download results from pod: ", slog.Any("err", err.Error()))
		return err
	}

	ch <- ActionDone{
		PodName:  testInfo.PodName,
		Name:     downloadResultsCmd.displayName,
		Duration: time.Since(downloadStart),
	}

	return err
}

func (c *Cluster) DeletePod(ctx context.Context, podName string) error {
	err := deletePod(ctx, c.Clientset, c.Namespace, podName)
	if err != nil {
		c.Logger.Error("failed to delete pod: ", slog.Any("err", err.Error()))
		return err
	}

	return nil
}

func BuildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
