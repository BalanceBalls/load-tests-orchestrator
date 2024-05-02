package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

const viewportHeight = 20

var (
	// common styles
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

	// run view styles
	tableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))
	notStartedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	inProgressStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	completedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// prepare view styles
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	dotStyle      = helpStyle.Copy().UnsetMargins()
	durationStyle = dotStyle.Copy()
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ebeb13"))

	// confirm config view styles
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				MarginRight(2).
				Underline(true)

	propsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3268a8")).
			Italic(true).
			MarginLeft(1)
)
