package tui

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m *MainModel) handleFilepickerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			if m.currentView == FilePick {
				m.currentView = PodsSetup
			}
		case "enter":
			if m.currentView == FilePick && m.filepicker.selectedFile != "" {
				switch m.filepicker.mode {
				case 0:
					m.pods[m.paginator.Page].scenarioFilePath = m.filepicker.selectedFile
				case 1:
					m.pods[m.paginator.Page].propsFilePath = m.filepicker.selectedFile
				}

				m.currentView = PodsSetup
				return m, nil
			}
		}
	case clearErrorMsg:
		m.err = nil
	}

	var cmd tea.Cmd
	m.filepicker.model, cmd = m.filepicker.model.Update(msg)

	if didSelect, path := m.filepicker.model.DidSelectFile(msg); didSelect {
		m.filepicker.selectedFile = path
	}

	if didSelect, path := m.filepicker.model.DidSelectDisabledFile(msg); didSelect {
		m.err = errors.New(path + " is not valid.")
		m.filepicker.selectedFile = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m *MainModel) handleFilepickerView() string {
	var s strings.Builder

	if m.filepicker.mode == 0 {
		s.WriteString(accentInfo.Render("Set scenario to run in a pod"))
	} else {
		s.WriteString(accentInfo.Render("Set propetries for a scenario"))
	}

	s.WriteString("\n")
	if m.err != nil {
		s.WriteString(m.filepicker.model.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.filepicker.selectedFile == "" {
		s.WriteString("\nPick a file:")
	} else {
		s.WriteString("Selected file: " + m.filepicker.model.Styles.Selected.Render(m.filepicker.selectedFile))
	}
	s.WriteString("\n\n" + m.filepicker.model.View() + "\n")

	s.WriteString("\n\n  h/l ←/→ • enter: accept • ctrl+c: quit\n")
	return s.String()
}

func GetFilePicker() filepicker.Model {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".mod", ".sum", ".go", ".txt", ".md"}
	fp.CurrentDirectory, _ = os.Getwd()
	fp.AutoHeight = false
	fp.Height = 15

	return fp
}
