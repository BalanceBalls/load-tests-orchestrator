package kubeutils

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

const logFileName = "newlog.jtl"
const resultPathEnding = "Results"
const resultsPath = "LoadTestResutls/"

// Pod setup
const (
	installAndUpdateDeps = "apt update && apt install openjdk-11-jre-headless wget unzip nano -y"
	downloadJmeter       = "mkdir jmeter && cd jmeter && wget http://www.gtlib.gatech.edu/pub/apache/jmeter/binaries/apache-jmeter-5.6.3.tgz"
	unpackJmeterArchive  = "cd jmeter && tar -xf apache-jmeter-5.6.3.tgz && rm apache-jmeter-5.6.3.tgz"
	downloadPlugin       = "cd jmeter && wget https://jmeter-plugins.org/files/packages/jpgc-casutg-2.10.zip"
	unpackPluginArchive  = "cd jmeter && unzip jpgc-casutg-2.10.zip -d apache-jmeter-5.6.3/ && rm jpgc-casutg-2.10.zip"
	testJmeter           = "jmeter/apache-jmeter-5.6.3/bin/jmeter --help"
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

	cpyScenarioCmd := exec.Command(
		"kubectl",
		"cp",
		"-n",
		namespace,
		test.ScenarioFileName,
		test.PodName+":/jmeter/"+scenarioFilePath,
		"-c",
		test.PodName,
	)

	cpyProprsCmd := exec.Command(
		"kubectl",
		"cp",
		"-n",
		namespace,
		test.PropFileName,
		test.PodName+":/jmeter/"+propertiesFilePath,
		"-c",
		test.PodName,
	)

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

func getRunTestCommand(test TestInfo) string {
	copyScenario := fmt.Sprintf(
		"touch jmeter/run.sh &&" + 
		"echo \"apache-jmeter-5.6.3/bin/jmeter -q %s -n -t '%s' -e -o %s -l %s\" > jmeter/run.sh &&" +
		"chmod +x jmeter/run.sh",
		test.PropFileName,
		test.ScenarioFileName,
		resultsPath,
		logFileName)
	return copyScenario
}
