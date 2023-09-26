package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/DimmyJing/valise/log"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type choice int

const (
	createOp choice = iota
	readOp
	updateOp
	deleteOp
	invalidOp
)

type encryptValue int

const (
	notDecided encryptValue = iota
	doEncrypt
	notEncrypt
)

type model struct {
	envVars      map[string]string
	envVarKeys   []string
	cursor       int
	choice       choice
	selected     string
	textInput    textinput.Model
	matches      []fuzzy.Match
	value        string
	encryptValue encryptValue
	key          []byte
}

func (m model) readEnvVar() string {
	return decrypt(m.key, m.envVars[m.selected])
}

var errEncryptNotDecided = errors.New("encrypt value not decided")

func (m model) getValue() string {
	if m.encryptValue == notEncrypt {
		return m.value
	} else if m.encryptValue == doEncrypt {
		return encrypt(m.key, m.value)
	}

	log.Panic(errEncryptNotDecided)

	return ""
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) restart() (tea.Model, tea.Cmd) { //nolint:ireturn
	m.cursor = 0
	m.choice = invalidOp
	m.selected = ""
	m.textInput.SetValue("")
	m.textInput.SetCursor(0)
	m.textInput.Blur()
	m.textInput.Placeholder = "Environmental Variable Name"
	m.matches = []fuzzy.Match{}
	m.value = ""
	m.envVarKeys = []string{}
	m.encryptValue = notDecided

	for k := range m.envVars {
		m.envVarKeys = append(m.envVarKeys, k)
	}

	return m, nil
}

var errInvalidChoice = errors.New("invalid choice")

func (m model) processOp() (tea.Model, tea.Cmd) { //nolint:ireturn,cyclop
	if (m.choice == createOp || m.choice == updateOp) && m.encryptValue == notDecided {
		return m, nil
	}

	switch m.choice {
	case createOp, updateOp:
		m.envVars[m.selected] = m.getValue()
	case deleteOp:
		delete(m.envVars, m.selected)
	case readOp, invalidOp:
		log.Panic(errInvalidChoice)
	}

	if m.choice != readOp {
		res, err := json.MarshalIndent(m.envVars, "", "  ")
		if err != nil {
			log.Panic(err)
		}

		for _, name := range findFile("env.json") {
			//nolint:gosec,gomnd
			err := os.WriteFile(name, res, 0o644)
			if err != nil {
				log.Panic(err)
			}
		}
	}

	return m, nil
}

func (m model) Update( //nolint:ireturn,funlen,gocognit,gocyclo,cyclop
	msg tea.Msg,
) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	//nolint:gocritic
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if msg.String() != "q" || !m.textInput.Focused() {
				return m, tea.Quit
			}
		case "y":
			if m.value != "" && m.encryptValue == notDecided {
				m.encryptValue = doEncrypt

				return m.processOp()
			}
		case "n":
			if m.value != "" && m.encryptValue == notDecided {
				m.encryptValue = notEncrypt

				return m.processOp()
			}
		case "j", "down":
			//nolint:nestif
			if m.choice == invalidOp {
				//nolint:gomnd
				if m.cursor < 3 {
					m.cursor++
				}

				return m, nil
			} else if m.selected == "" {
				if msg.String() == "down" {
					if m.cursor < len(m.matches)-1 {
						m.cursor++
					}

					return m, nil
				}
			}
		case "k", "up":
			//nolint:nestif
			if m.choice == invalidOp {
				if m.cursor > 0 {
					m.cursor--
				}

				return m, nil
			} else if m.selected == "" {
				if msg.String() == "up" {
					if m.cursor > 0 {
						m.cursor--
					}

					return m, nil
				}
			}
		//nolint:goconst
		case "enter", "tab":
			//nolint:nestif
			if m.choice == invalidOp {
				if msg.String() == "enter" {
					m.choice = choice(m.cursor)
					m.cursor = 0
					cmd := m.textInput.Focus()

					return m, cmd
				}
			} else if m.selected == "" {
				if msg.String() == "tab" || (msg.String() == "enter" && m.choice != createOp) {
					if m.cursor >= 0 && m.cursor < len(m.matches) {
						m.textInput.SetValue(m.matches[m.cursor].Str)
						m.textInput.SetCursor(len(m.matches[m.cursor].Str))
						if msg.String() == "enter" {
							m.textInput.Placeholder = "Environmental Variable Value"
							m.selected = m.matches[m.cursor].Str
							m.textInput.SetValue("")
							m.textInput.SetCursor(0)

							return m.processOp()
						}
					}
				} else if slices.Index(m.envVarKeys, m.textInput.Value()) == -1 {
					m.textInput.Placeholder = "Environmental Variable Value"
					m.selected = m.textInput.Value()
					m.textInput.SetValue("")
					m.textInput.SetCursor(0)

					return m.processOp()
				}
			} else {
				if msg.String() == "enter" && (m.choice == createOp || m.choice == updateOp) && m.value == "" {
					m.value = m.textInput.Value()

					return m.processOp()
				}
				if m.value == "" || m.encryptValue != notDecided {
					return m.restart()
				}
			}

			return m, nil
		}
	}

	m.matches = fuzzy.Find(m.textInput.Value(), m.envVarKeys)
	if m.choice != invalidOp && m.selected == "" {
		if m.cursor >= len(m.matches) {
			m.cursor = len(m.matches) - 1
		} else if m.cursor < 0 {
			m.cursor = 0
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)

	return m, cmd
}

//nolint:gochecknoglobals
var (
	colorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("147"))
	radioOn    = "◉"
	radioOff   = "◯"
)

func (m model) selectChoiceView(builder *strings.Builder) {
	choices := []string{"Create", "Read", "Update", "Delete"}

	builder.WriteString("What do you want to do?\n\n")

	for idx, choice := range choices {
		var dot string
		if m.cursor == idx {
			dot = radioOn
		} else {
			dot = radioOff
		}

		line := fmt.Sprintf("%s %s environmental variable", dot, choice)

		var style lipgloss.Style

		if m.cursor == idx {
			style = colorStyle
		} else {
			style = lipgloss.NewStyle()
		}

		builder.WriteString(fmt.Sprintf("  %s\n", style.Render(line)))
	}

	builder.WriteString("\nPress q to quit.\n")
}

func (m model) selectEnvVarView(builder *strings.Builder) {
	choices := []string{"create", "read", "update", "delete"}
	builder.WriteString(
		fmt.Sprintf("Please choose an environmental variable to %s:\n\n", choices[m.choice]),
	)
	builder.WriteString(m.textInput.View())
	builder.WriteString("\n")

	for idx, match := range m.matches {
		if m.cursor == idx {
			builder.WriteString(colorStyle.Render(radioOn + " "))
		} else {
			builder.WriteString("  ")
		}

		for idx, char := range match.Str {
			if slices.Index(match.MatchedIndexes, idx) != -1 {
				builder.WriteString(colorStyle.Render(string(char)))
			} else {
				builder.WriteRune(char)
			}
		}

		builder.WriteString("\n")
	}
}

func (m model) valueView(builder *strings.Builder) {
	builder.WriteString("Please enter the value for " + m.selected + ":\n\n")
	builder.WriteString(m.textInput.View())
	builder.WriteString("\n")
}

func (m model) decideEncrypt(builder *strings.Builder) {
	builder.WriteString("Do you want to encrypt this value? (y/n)")
}

var errInvalidOperation = errors.New("invalid operation")

func (m model) operationEnvVarView(builder *strings.Builder) {
	switch m.choice {
	case createOp:
		builder.WriteString("Successfully created " + m.selected)
	case readOp:
		builder.WriteString(
			fmt.Sprintf("The value of \"%s\" is \"%s\"", m.selected, m.readEnvVar()),
		)
	case updateOp:
		builder.WriteString("Successfully updated " + m.selected)
	case deleteOp:
		builder.WriteString("Successfully deleted " + m.selected)
	case invalidOp:
		log.Panic(errInvalidOperation)
	}

	builder.WriteString("\n\nPress enter to continue.")
}

func (m model) View() string {
	var stringBuilder strings.Builder

	switch {
	case m.choice == invalidOp:
		m.selectChoiceView(&stringBuilder)
	case m.selected == "":
		m.selectEnvVarView(&stringBuilder)
	case (m.choice == createOp || m.choice == updateOp) && m.value == "":
		m.valueView(&stringBuilder)
	case m.value != "" && m.encryptValue == notDecided:
		m.decideEncrypt(&stringBuilder)
	default:
		m.operationEnvVarView(&stringBuilder)
	}

	return stringBuilder.String()
}

func initialModel(envJSON []byte) model {
	textinp := textinput.New()
	textinp.Placeholder = "Environmental Variable Name"
	textinp.CharLimit = 1000

	var model model

	err := json.Unmarshal(envJSON, &model.envVars)
	if err != nil {
		log.Panic(err)
	}

	for k := range model.envVars {
		model.envVarKeys = append(model.envVarKeys, k)
	}

	model.choice = invalidOp
	model.textInput = textinp
	model.encryptValue = notDecided

	model.key, err = getKey()
	if err != nil {
		log.Panic(err)
	}

	return model
}

func Interactive(envJSON []byte) error {
	_, err := getKey()
	if err != nil {
		return errInvalidKey
	}

	p := tea.NewProgram(initialModel(envJSON))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("interactive env error: %w", err)
	}

	return nil
}
