package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle  = focusedStyle.Copy()
	noStyle      = lipgloss.NewStyle()
	helpStyle    = blurredStyle.Copy()

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))
	alertStyle    = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#fc0313")).
			Blink(true)
	configuredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#03fc52")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#383838"))
	configInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5"))
	accentInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f54287"))
	podLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#378524")).
			PaddingLeft(1).
			PaddingRight(1)
	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(lipgloss.Color("#383838")).
		String()
)

const (
	Config int = iota
	FilePick
	PodsSetup
	ReviewSetup
	PreparePods
	Run
	Finish
)

type PodInfo struct {
	id               int
	name             string
	logs             string
	propsFilePath    string
	scenarioFilePath string
}

type MainModel struct {
	currentView int
	currentPod  int
	pods        []PodInfo

	paginator         paginator.Model
	filepicker        FilePickerModule
	setupConfirmation ConfirmationModel
	preparation       *PreparePodsModel
	run               *TestRunModel
	configForm        *ConfigViewModel
	err               error
}

type FilePickerModule struct {
	model        filepicker.Model
	selectedFile string
	mode         int
}

func initialModel() *MainModel {
	m := MainModel{
		currentView: Config}

	m.initConfigForm()
	m.initPaginatorView(0)

	return &m
}

func (m *MainModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.currentView == FilePick {
				m.currentView = PodsSetup
			}
		}
	}

	switch m.currentView {
	case Config:
		return m.handleConfigFormUpdate(msg)
	case PodsSetup:
		return m.handlePaginatorUpdate(msg)
	case FilePick:
		return m.handleFilepickerUpdate(msg)
	case ReviewSetup:
		return m.handleConfirmationUpdate(msg)
	case PreparePods:
		return m.handlePodsPreparationUpdate(msg)
	case Run:
		return m.handleRunUpdate(msg)
	default:
		return m, nil
	}
}

func (m *MainModel) View() string {
	switch m.currentView {
	case Config:
		return m.handleConfigFormView()
	case PodsSetup:
		return m.handlePaginatorView()
	case FilePick:
		return m.handleFilepickerView()
	case ReviewSetup:
		return m.handleConfirmationView()
	case PreparePods:
		return m.handlePodsPreparationView()
	case Run:
		return m.handleRunView()
	default:
		return ""
	}
}

func DisplayUI() {
	if _, err := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion()).
		Run(); err != nil {
		fmt.Printf("could not start program: %s\n", err)
		os.Exit(1)
	}
}
