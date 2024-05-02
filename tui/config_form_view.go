package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type configDone struct {
	connectionOk bool
}
type ConfigViewModel struct {
	focusIndex            int
	cursorMode            cursor.Mode
	inputs                []textinput.Model
	spinner               spinner.Model
	showSpinner           bool
	connectionEstablished bool
	done                  configDone
	err                   error
}

func (m ConfiguratorModel) handleConfigFormView() string {
	var b strings.Builder

	b.WriteString(focusedStyle.Render("Set cluster config\n"))

	if m.configForm.err != nil {
		b.WriteString("\nError: " + m.configForm.err.Error())
	}

	for i := range m.configForm.inputs {
		if i == 0 {
			b.WriteString("\n")
		}
		b.WriteString(m.configForm.inputs[i].View())
		if i < len(m.configForm.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.configForm.focusIndex == len(m.configForm.inputs) {
		button = &focusedButton
	}

	if m.configForm.showSpinner && !m.configForm.connectionEstablished {
		b.WriteString("\n" + m.configForm.spinner.View())
		b.WriteString(accentInfo.Render("Establising k8s cluster connection...\n"))
	}

	if m.configForm.connectionEstablished {
		b.WriteString(accentInfo.Render("\nSuccessfully connected to k8s cluster"))
	}

	fmt.Fprintf(&b, "\n\n%s\n\n", *button)
	return b.String()
}

func (m *ConfiguratorModel) initConfigForm() {
	m.configForm = &ConfigViewModel{}
	m.configForm.inputs = make([]textinput.Model, 3)
	var t textinput.Model
	for i := range m.configForm.inputs {
		t = textinput.New()
		t.Cursor.Style = cursorStyle
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "Pod name prefix"
			t.Focus()
			t.PromptStyle = focusedStyle
			t.TextStyle = focusedStyle
		case 1:
			t.Placeholder = "Namespace"
			t.CharLimit = 64
		case 2:
			t.Placeholder = "Amount of pods"
			t.Validate = func(input string) error {
				numPod, err := strconv.Atoi(input)
				if err != nil {
					m.configForm.err = errors.New("Input must be a number")
					return m.configForm.err
				}
				if numPod < 1 {
					m.configForm.err = errors.New("Need at least one pod")
					return m.configForm.err
				}
				if numPod > 99 {
					m.configForm.err = errors.New("Max amount exceeded")
					return m.configForm.err
				}

				return nil
			}
		}
		m.configForm.inputs[i] = t
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Spinner.FPS = 200 * time.Millisecond
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m.configForm.spinner = s
	m.configForm.connectionEstablished = false
	m.configForm.showSpinner = false
}

func (m *ConfiguratorModel) handleConfigFormUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.configForm.showSpinner || m.configForm.connectionEstablished {
			return m, nil
		}

		switch msg.String() {
		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			if s == "enter" && m.configForm.focusIndex == len(m.configForm.inputs) {
				if m.configForm.inputs[0].Value() == "" ||
					m.configForm.inputs[1].Value() == "" ||
					m.configForm.inputs[2].Value() == "" {
					m.configForm.err = errors.New("Incorrect configuration")
					return m, nil
				}
				m.configForm.err = nil
				m.configForm.showSpinner = true

				go func() {
					ch := make(chan configDone)
					defer close(ch)

					go m.configForm.checkClusterConnection(ch)

					r := <-ch
					m.Update(r)
				}()
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.configForm.focusIndex--
			} else {
				m.configForm.focusIndex++
			}

			if m.configForm.focusIndex > len(m.configForm.inputs) {
				m.configForm.focusIndex = 0
			} else if m.configForm.focusIndex < 0 {
				m.configForm.focusIndex = len(m.configForm.inputs)
			}

			cmds := make([]tea.Cmd, len(m.configForm.inputs))
			for i := 0; i <= len(m.configForm.inputs)-1; i++ {
				if i == m.configForm.focusIndex {
					// Set focused state
					cmds[i] = m.configForm.inputs[i].Focus()
					m.configForm.inputs[i].PromptStyle = focusedStyle
					m.configForm.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.configForm.inputs[i].Blur()
				m.configForm.inputs[i].PromptStyle = noStyle
				m.configForm.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	case configDone:
		totalPages, err := strconv.Atoi(m.configForm.inputs[2].Value())
		if err != nil {
			panic(err)
		}

		m.configForm.showSpinner = false
		time.Sleep(1 * time.Second)
		m.initPaginatorView(totalPages)
		m.currentView = PodsSetup

		return m, nil
	}

	var updSpinner spinner.Model
	updSpinner, cmdS := m.configForm.spinner.Update(msg)
	m.configForm.spinner = updSpinner
	cmds = append(cmds, cmdS)

	cmd := m.updateInputs(msg)
	cmds = append(cmds, cmd)
	cmds = append(cmds, m.configForm.spinner.Tick)
	return m, tea.Batch(cmds...)
}

func (m *ConfiguratorModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.configForm.inputs))
	for i := range m.configForm.inputs {
		m.configForm.inputs[i], cmds[i] = m.configForm.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m *ConfigViewModel) checkClusterConnection(ch chan<- configDone) {
	// Perform check
	time.Sleep(1 * time.Second)

	m.connectionEstablished = true
	ch <- configDone{connectionOk: true}
}
