# Load tests orchestrator
[load-test-orchestrator.webm](https://github.com/BalanceBalls/terminal-ui/assets/29193297/98d13ac8-72ef-4b74-bdca-27c16189d5ec)


## What it does
This tool automates setting up k8s pods for jmeter load tests and provides contol over tests execution:
 * Creating pods
 * Setting up JMeter and plugins for each pod
 * Uploading jmeter scenarios and properties
 * Starting load test runs simultaniously
 * Cancel / reset runs
 * Logs streaming from pods
 * Archiving / downloading results
 * Terminating pods

## How to use
 * use 'hjkl' or arrow keys for navigation
 * 's' select .jmx scenario
 * 'p' select .properties file for a scenario
 * 'c' to proceed to another form (where applicable)
 * 'b' go to previous form (where applicable)
 * 'ctrl+c' exit
 * 'ctrl+s' starts run
 * 'ctrl+k' cancels run
 * 'ctrl+r' resets run

## Reuqirements 
 * kubectl installed and configured
 * go
