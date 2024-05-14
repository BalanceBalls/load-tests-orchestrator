package tui

import (
	"context"
	"log/slog"
	"terminalui/kubeutils"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/huh"
)

type PodInfo struct {
	id               int
	name             string
	data             PodLogs
	propsFilePath    string
	scenarioFilePath string
}

type ConfigDone struct {
	ConnectionOk bool
}

type PodLogs struct {
	// How many consecutive checks logs remain unchanged
	staleFor int
	logs     string
}

type ConfigViewModel struct {
	focusIndex            int
	inputs                []textinput.Model
	spinner               spinner.Model
	showSpinner           bool
	connectionEstablished bool
	err                   error
}

type PreparePodsModel struct {
	pods     []PodInfo
	spinner  spinner.Model
	results  []kubeutils.ActionDone
	quitting bool
	err      string

	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

type PrepareResultsModel struct {
	pods              []PodInfo
	spinner           spinner.Model
	results           []kubeutils.ActionDone
	isCollected       bool
	quitting          bool
	deletePodsConfirm *huh.Form
	showConfirmation  bool

	err    error
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

type ConfirmationModel struct {
	isConfirmed      bool
	content          string
	ready            bool
	viewport         viewport.Model
	confirmationForm huh.Form
}

type ConfiguratorModel struct {
	ctx               context.Context
	logger            *slog.Logger
	currentView       AppViewState
	pods              []PodInfo
	podKeepAliveSec   int
	updateIntervalSec int

	cluster           *kubeutils.Cluster
	paginator         *paginator.Model
	filepicker        *FilePickerModule
	setupConfirmation *ConfirmationModel
	preparation       *PreparePodsModel
	configForm        *ConfigViewModel
	resultsCollection *PrepareResultsModel
	run               *TestRunModel
	err               error
}

type TestRunModel struct {
	runState     TestRunState
	namespace    string
	pods         []RunPodInfo
	isTableView  bool
	currentPod   int
	podViews     []viewport.Model
	pages        paginator.Model
	confirm      *huh.Form
	table        string
	spinner      spinner.Model
	showSpinner  bool
	isConfirmed  bool
	showConfirm  bool
	prevRunState *TestRunState
}

type FilePickerModule struct {
	model        filepicker.Model
	selectedFile string
	mode         int
}

type PodUpdate struct {
	podIndex     int
	logs         string
	staleCounter int
	inProgress   bool
	state        TestRunState
	err          error
}

type RunPodInfo struct {
	PodInfo

	runState   TestRunState
	err        error
	resultPath string
}

type ClearErrorMsg struct{}

type AppViewState uint

const (
	Config AppViewState = iota
	FilePick
	PodsSetup
	ReviewSetup
	PreparePods
	Run
	Collect
	Finish
)

type TestRunState uint

const (
	NotStarted TestRunState = iota
	StartConfirm
	InProgress
	Completed
	CancelConfirm
	Cancelled
	ResetConfirm
	Failed
	Done
)
