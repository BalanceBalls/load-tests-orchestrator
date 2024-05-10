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
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("57")).
			PaddingLeft(1).
			PaddingRight(1)
	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(lipgloss.Color("#383838")).
		String()

	// run view styles
	tableBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	notStartedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	inProgressStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	completedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	podLogsStyle     = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10")).
				Border(lipgloss.NormalBorder()).
				BorderTop(true).
				BorderBottom(true).
				BorderRight(false).
				BorderLeft(false).
				TabWidth(2).
				BorderForeground(lipgloss.Color("11"))

	// prepare view styles
	spinnerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	dotStyle      = helpStyle.Copy().UnsetMargins()
	durationStyle = dotStyle.Copy()
	appStyle      = lipgloss.NewStyle().Margin(1, 2, 0, 2)
	stepNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ebeb13"))

	listItemStyle = lipgloss.NewStyle().
			MarginLeft(1).
			Border(lipgloss.HiddenBorder()).
			BorderRight(false).
			BorderLeft(false).
			BorderForeground(lipgloss.Color("11"))
)
