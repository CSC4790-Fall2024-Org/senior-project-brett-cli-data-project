package tui

import (
	"context"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:

		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case progress.FrameMsg:
		var cmd tea.Cmd
		var newModel tea.Model
		newModel, cmd = m.progress.Update(msg)
		m.progress = newModel.(progress.Model) // Type assertion to progress.Model
		return m, cmd
	case progressMsg:
		m.progressValue += float64(msg)
		if m.progressValue > 1.0 {
			m.progressValue = 1.0
		}
		cmd := m.progress.SetPercent(m.progressValue)
		if m.progressValue < 1.0 && m.currentScreen == "running_script" {
			return m, tea.Batch(cmd, incrementProgressCmd())
		}
		return m, cmd

	case scriptSuccessMsg:
		m.scriptCancel = nil
		m.currentScreen = "pipeline_created"
		m.scriptOutput = string(msg)
		m.progressValue = 1.0
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case scriptErrorMsg:
		m.scriptCancel = nil
		m.currentScreen = "pipeline_error"
		m.scriptOutput = msg.err.Error()
		m.progressValue = 1.0
		cmd := m.progress.SetPercent(1.0)
		return m, cmd

	case createDataLakeSuccessMsg:
		return m, nil

	case createDataLakeErrorMsg:
		return m, nil

	case queryResultMsg:
		if msg.err != nil {
			m.queryResult = msg.err.Error()
		} else {
			m.queryResult = msg.result
		}
		return m, nil

	case tea.KeyMsg:

		if m.confirmReset {
			switch msg.String() {
			case "y", "Y":
				m.confirmReset = false
				m.currentScreen = ""
				m.stage = 0
				m.cursorPosition = 0
				m.inputs = []string{"", "", ""}
				m.state = "welcome"
			case "n", "N":
				m.confirmReset = false
			}
			return m, nil
		}

		// Handle Escape key to return home or quit
		if msg.Type == tea.KeyEsc {
			if m.state != "welcome" {
				if m.currentScreen == "running_script" {
					if m.scriptCancel != nil {
						m.scriptCancel()
					}
					m.currentScreen = "welcome"
					m.state = "welcome"
					m.stage = 0
					m.inputs = []string{"", "", ""}
					m.cursorPosition = 0
					return m, nil
				}
				m.confirmReset = true
			} else {
				return m, tea.Quit
			}
			return m, nil
		}

		// During text input stages, process keys as input characters
		if m.state == "create_pipeline" && m.stage == 0 {
			switch msg.Type {
			case tea.KeyEnter:
				if len(m.inputs[0]) > 0 {
					m.stage++
					m.cursorPosition = 0
				}
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.inputs[0]) > 0 {
					m.inputs[0] = m.inputs[0][:len(m.inputs[0])-1]
				}
			case tea.KeyRunes:
				m.inputs[0] += msg.String()
			default:

			}
			return m, nil
		}

		// Handle action bar shortcuts
		if m.currentScreen != "" {

			switch msg.String() {
			case "?":
				m.currentScreen = "help"
			case "a":
				m.currentScreen = "about"
			case "p":
				m.currentScreen = "pipelines"
			case "s":
				m.currentScreen = "save"
			case "q", "ctrl+c", "ctrl+q":
				return m, tea.Quit
			case "c":
				m.state = "create_pipeline"
				m.stage = 0
				m.currentScreen = ""
			case "e":
				m.currentScreen = "query editor"
				m.inDataLakeSelect = true
				m.selectedDataLake = 0
				return m, nil

			default:
				// This causes any other key that is pressed to exit the screen
				m.currentScreen = ""
			}
			return m, nil
		}
		// Handle data lake selection
		if m.inDataLakeSelect {
			switch msg.String() {
			case "up":
				if m.selectedDataLake > 0 {
					m.selectedDataLake--
				}
			case "down":
				if m.selectedDataLake < len(m.dataLakes)-1 {
					m.selectedDataLake++
				}
			case "enter":
				m.inDataLakeSelect = false
				m.inQueryEditor = true
				m.queryInput = ""
				m.queryResult = ""
			case "esc":
				m.inDataLakeSelect = false
			}
			return m, nil
		}
		// Handle key messages when in query editor
		if m.inQueryEditor {
			switch msg.Type {
			case tea.KeyEnter:
				// Execute the query
				return m, executeQueryCmd(m.dataLakes[m.selectedDataLake], m.queryInput)
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.queryInput) > 0 {
					m.queryInput = m.queryInput[:len(m.queryInput)-1]
				}
			case tea.KeySpace:
				m.queryInput += " "
			case tea.KeyRunes:
				m.queryInput += string(msg.Runes)
			case tea.KeyTab:
				// Exit the query editor
				m.inQueryEditor = false
				m.inDataLakeSelect = true
				m.selectedDataLake = 0
				m.queryInput = ""
				m.queryResult = ""

			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "ctrl+q", "q":
			return m, tea.Quit

		case "?":
			m.currentScreen = "help"
			return m, nil

		case "a":
			m.currentScreen = "about"
			return m, nil

		case "p":
			m.currentScreen = "pipelines"
			return m, nil

		case "s":
			m.currentScreen = "save"
			return m, nil

		case "c":
			m.state = "create_pipeline"
			m.stage = 0
			return m, nil

		case "e":
			m.currentScreen = "query editor"
			m.inDataLakeSelect = true
			m.selectedDataLake = 0
			return m, nil
		}

		// Handle input based on the current state
		if m.state == "welcome" {

			if msg.Type == tea.KeyEnter {
				m.state = "create_pipeline"
				m.stage = 0
			}
		} else if m.state == "create_pipeline" {

			switch m.stage {
			case 1:
				switch msg.String() {
				case "up":
					if m.cursorPosition > 0 {
						m.cursorPosition--
					}
				case "down":
					if m.cursorPosition < len(m.services)-1 {
						m.cursorPosition++
					}
				case "enter":
					m.selectedService = m.cursorPosition
					m.stage++
					m.cursorPosition = 0
				}
			case 2:
				switch msg.String() {
				case "up":
					if m.cursorPosition > 0 {
						m.cursorPosition--
					}
				case "down":
					if m.cursorPosition < len(m.dataTypes)-1 {
						m.cursorPosition++
					}
				case "enter":
					m.selectedDataType = m.cursorPosition
					m.stage++
				}
			case 3:
				if msg.String() == "enter" {
					m.currentScreen = "running_script"
					m.progressValue = 0.0
					m.progress.SetPercent(0.0)
					// Create a context to cancel the script if needed
					var ctx context.Context
					ctx, m.scriptCancel = context.WithCancel(context.Background())
					// Start the script and progress bar
					return m, tea.Batch(runScriptCmd(ctx), incrementProgressCmd())
				}
			}
		}
	}

	return m, nil
}
