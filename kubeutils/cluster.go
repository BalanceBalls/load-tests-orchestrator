package kubeutils

import (
	"context"
	"log/slog"
	"os"
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func (c *Cluster) Ping(ctx context.Context) (bool, error) {
	connected, err := checkClusterConnection(ctx, c.Clientset)
	if err != nil {
		c.Logger.Error(err.Error())
	}
	return connected, err
}

func (c *Cluster) CreatePod(ctx context.Context, podName string) error {
	pod, err := createPod(ctx, c.Clientset, c.Namespace, podName)
	if err != nil {
		c.Logger.Error("Failed to create pod: ", slog.Any("err", err))
		return err
	}

	testCmd := "pwd"
	_, _, err = executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, testCmd)

	return nil
}

func (c *Cluster) PreparePod(ctx context.Context, testInfo TestInfo, ch chan<- ActionDone) error {
	// pod, err := c.Clientset.CoreV1().Pods(c.Namespace).Get(ctx, testInfo.PodName, v1.GetOptions{})

	podCreationStart := time.Now()
	pod, err := createPod(ctx, c.Clientset, c.Namespace, testInfo.PodName)
	if err != nil {
		c.Logger.Error("Failed to create pod: ", slog.Any("err", err.Error()))
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
			c.Logger.Error("Failed to execute command: ", slog.Any("err", err.Error()))
			return err
		}

		ch <- ActionDone{
			PodName:  testInfo.PodName,
			Name:     cmd.displayName,
			Duration: time.Since(start),
		}

		c.Logger.Info("Command strbuff: " + strBuf)
		c.Logger.Info("Command errbuff: " + errBuf)
	}

	cpCmds := getTestUploadCommands(testInfo, c.Namespace)

	switchLocalK8sContext(c.KubeCtxName)
	for _, cpCmd := range cpCmds {
		start := time.Now()
		cpCmd.command.Stdout = os.Stdout
		cpCmd.command.Stderr = os.Stderr

		c.Logger.Info("Executing cmd: " + cpCmd.command.String())

		err := cpCmd.command.Run()
		if err != nil {
			c.Logger.Error("Failed to copy file to pod: ", slog.Any("err", err.Error()))
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
	pod, err := c.Clientset.CoreV1().Pods(c.Namespace).Get(ctx, testInfo.PodName, v1.GetOptions{})

	if err != nil {
		c.Logger.Error(err.Error())
	}

	stdOut, _, err := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, "cat jmeter/jmeter.log")

	finishedRunIndicator := "cd jmeter/" + resultsPath
	// TODO: consider 'top -bn1 > state | cat state | grep jmeter' to watch actual jmeter process
	_, _, fErr := executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, finishedRunIndicator)
	isFinished := false
	if fErr == nil {
		isFinished = true
	}

	return isFinished, stdOut, err
}

func (c *Cluster) KickstartTestForPod(ctx context.Context, testInfo TestInfo) error {
	pod, err := c.Clientset.CoreV1().Pods(c.Namespace).Get(ctx, testInfo.PodName, v1.GetOptions{})

	if err != nil {
		c.Logger.Error(err.Error())
	}

	cmd := getRunTestCommand(testInfo)
	_, _, err = executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, cmd)

	go executeRemoteCommand(ctx, c.RestCfg, c.Clientset, pod, "cd jmeter && sh ./run.sh")
	return err
}

func (c *Cluster) CollectResultsFromPod(ctx context.Context) error {
	return nil
}

func (c *Cluster) DeletePod(ctx context.Context, podName string) error {
	err := deletePod(ctx, c.Clientset, c.Namespace, podName)
	if err != nil {
		c.Logger.Error("Failed to delete pod: ", slog.Any("err", err.Error()))
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
