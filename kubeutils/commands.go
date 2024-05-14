package kubeutils

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const logFileName = "newlog.jtl"
const resultsPath = "LoadTestResutls/"

// Pod setup
const (
	installAndUpdateDeps = "apt update && apt install openjdk-11-jre-headless wget unzip nano -y"
	downloadJmeter       = "mkdir jmeter && cd jmeter && wget https://dlcdn.apache.org//jmeter/binaries/apache-jmeter-5.6.3.tgz"
	unpackJmeterArchive  = "cd jmeter && tar -xf apache-jmeter-5.6.3.tgz && rm apache-jmeter-5.6.3.tgz"
	downloadPlugin       = "cd jmeter && wget https://jmeter-plugins.org/files/packages/jpgc-casutg-2.10.zip"
	unpackPluginArchive  = "cd jmeter && unzip jpgc-casutg-2.10.zip -d apache-jmeter-5.6.3/ && rm jpgc-casutg-2.10.zip"
	testJmeter           = "jmeter/apache-jmeter-5.6.3/bin/jmeter --help"
)

// Test reset
const (
	removeResultsDir  = "rm -r jmeter/" + resultsPath
	removeJmeterLog   = "rm jmeter/jmeter.log"
	removeRequestsLog = "rm jmeter/newlog.jtl"
)

func getPodSetupCommands() []remoteCommand {
	var cmds []remoteCommand

	cmds = append(cmds, remoteCommand{
		displayName: "updating packages and installing jdk",
		command:     installAndUpdateDeps,
	})

	cmds = append(cmds, remoteCommand{
		displayName: "downloading JMeter",
		command:     downloadJmeter,
	})

	cmds = append(cmds, remoteCommand{
		displayName: "unarchiving JMeter and removing archive",
		command:     unpackJmeterArchive,
	})

	cmds = append(cmds, remoteCommand{
		displayName: "downloading threads plugin",
		command:     downloadPlugin,
	})

	cmds = append(cmds, remoteCommand{
		displayName: "unpacking plugin and removing archive",
		command:     unpackPluginArchive,
	})

	cmds = append(cmds, remoteCommand{
		displayName: "testing JMeter installation",
		command:     testJmeter,
	})

	return cmds
}

func getTestUploadCommands(test TestInfo, namespace string) []localCommand {
	var cmds []localCommand
	_, scenarioFilePath := filepath.Split(test.ScenarioFileName)
	_, propertiesFilePath := filepath.Split(test.PropFileName)

	var cpyScenarioCmd *exec.Cmd
	var cpyProprsCmd *exec.Cmd

	if runtime.GOOS == "windows" {
		currDir, _ := os.Getwd()
		relPathToScenario, _ := filepath.Rel(currDir, test.ScenarioFileName)
		cpyScenarioCmd = exec.Command(
			"kubectl",
			"cp",
			"-n",
			namespace,
			relPathToScenario,
			test.PodName+":/jmeter/"+scenarioFilePath,
			"-c",
			test.PodName,
		)

		relPathToProperties, _ := filepath.Rel(currDir, test.PropFileName)
		cpyProprsCmd = exec.Command(
			"kubectl",
			"cp",
			"-n",
			namespace,
			relPathToProperties,
			test.PodName+":/jmeter/"+propertiesFilePath,
			"-c",
			test.PodName,
		)
	} else {
		cpyScenarioCmd = exec.Command(
			"kubectl",
			"cp",
			"-n",
			namespace,
			test.ScenarioFileName,
			test.PodName+":/jmeter/"+scenarioFilePath,
			"-c",
			test.PodName,
		)

		cpyProprsCmd = exec.Command(
			"kubectl",
			"cp",
			"-n",
			namespace,
			test.PropFileName,
			test.PodName+":/jmeter/"+propertiesFilePath,
			"-c",
			test.PodName,
		)
	}

	uploadScenario := localCommand{
		displayName: "upload scenario file",
		command:     cpyScenarioCmd,
	}

	uploadProperties := localCommand{
		displayName: "upload properties file",
		command:     cpyProprsCmd,
	}

	cmds = append(cmds, uploadScenario, uploadProperties)
	return cmds
}

func getPrepareRunTestCommand(test TestInfo) string {
	copyScenario := fmt.Sprintf(
		"touch jmeter/run.sh &&"+
			"echo \"apache-jmeter-5.6.3/bin/jmeter -q %s -n -t '%s' -e -o %s -l %s\" > jmeter/run.sh &&"+
			"chmod +x jmeter/run.sh",
		test.PropFileName,
		test.ScenarioFileName,
		resultsPath,
		logFileName)
	return copyScenario
}

func getRunTestCommand() string {
	runTestCmd := "cd jmeter && sh ./run.sh &"
	return runTestCmd
}

func getStopTestCommand() string {
	stopCmd := "sh jmeter/apache-jmeter-5.6.3/bin/stoptest.sh"
	return stopCmd
}

func getResetTestCommands() []string {
	resetCmds := []string{removeResultsDir, removeJmeterLog, removeRequestsLog}

	return resetCmds
}

func getPackResultsCommand() string {
	resultFolderName := strings.TrimSuffix(resultsPath, "/")
	packResultsCmd := fmt.Sprintf("tar -zcvf jmeter/%s.tar.gz /jmeter/%s", resultFolderName, resultFolderName)
	return packResultsCmd
}

func getDownloadResultsCommand(test TestInfo, namespace string, podPrefix string) localCommand {
	ext := ".tar.gz"
	podResultsName := strings.TrimSuffix(resultsPath, "/")
	archivePath := podResultsName + ext

	pathToDir := fmt.Sprintf("./%s_results/%s/", podPrefix, test.PodName)
	os.MkdirAll(pathToDir, fs.ModePerm)

	localResultFilePath := pathToDir + podResultsName + ext

	cpyProprsCmd := exec.Command(
		"kubectl",
		"cp",
		"-n",
		namespace,
		test.PodName+":/jmeter/"+archivePath,
		localResultFilePath,
		"-c",
		test.PodName,
	)

	cmd := localCommand{
		displayName: "results saved to " + localResultFilePath,
		command:     cpyProprsCmd,
	}

	return cmd
}

func getCheckSuccessfulFinishCommand() string {
	finishedRunIndicator := "cd jmeter/" + resultsPath
	return finishedRunIndicator
}

func getCheckJmeterStateCommand() string {
	checkJmeterCmd := "top -bn1 | grep jmeter && echo 'running' || echo 'stopped'"
	return checkJmeterCmd
}
