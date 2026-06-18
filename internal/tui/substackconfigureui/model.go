package substackconfigureui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"postcli/internal/theme"
)

// Deps supplies environment and side effects from the CLI layer (avoids import cycles).
type Deps struct {
	Ctx context.Context

	Publication func() string
	Cookie      func() string
	SendEmail   func() bool
	PersistEnv  func(values map[string]string) error
}

type phase int

const (
	phaseInputs phase = iota
	phaseConfirm
	phaseSaved
)

type model struct {
	deps Deps

	phase     phase
	activeIdx int // 0: Publication, 1: Cookie, 2: SendEmail
	inputs    []textinput.Model
	sendEmail bool

	errLine  string
	infoLine string
	width    int
}

// Run starts the full-screen Substack configuration TUI.
func Run(d Deps) error {
	_ = theme.Load()
	m := NewModel(d)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func NewModel(d Deps) model {
	pubInput := textinput.New()
	pubInput.Placeholder = "e.g., myblog (or myblog.substack.com)"
	pubInput.SetValue(d.Publication())
	pubInput.SetWidth(50)
	pubInput.Focus()

	cookieInput := textinput.New()
	cookieInput.Placeholder = "Paste your substack.sid session cookie value"
	cookieInput.SetValue(d.Cookie())
	cookieInput.SetWidth(60)

	inputs := []textinput.Model{pubInput, cookieInput}

	return model{
		deps:      d,
		phase:     phaseInputs,
		activeIdx: 0,
		inputs:    inputs,
		sendEmail: d.SendEmail(),
	}
}

func (m model) Init() tea.Cmd {
	return func() tea.Msg { return textinput.Blink() }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.phase == phaseConfirm {
				m.phase = phaseInputs
				m.errLine = ""
				m.inputs[0].Focus()
				m.activeIdx = 0
				return m, nil
			}
			return m, tea.Quit

		case "up", "shift+tab":
			if m.phase == phaseInputs {
				if m.activeIdx < 2 {
					m.inputs[m.activeIdx].Blur()
				}
				m.activeIdx--
				if m.activeIdx < 0 {
					m.activeIdx = 2
				}
				m.errLine = ""
				if m.activeIdx < 2 {
					m.inputs[m.activeIdx].Focus()
				}
			}
			return m, nil

		case "down", "tab":
			if m.phase == phaseInputs {
				if m.activeIdx < 2 {
					m.inputs[m.activeIdx].Blur()
				}
				m.activeIdx++
				if m.activeIdx > 2 {
					m.activeIdx = 0
				}
				m.errLine = ""
				if m.activeIdx < 2 {
					m.inputs[m.activeIdx].Focus()
				}
			}
			return m, nil

		case "left", "right":
			if m.phase == phaseInputs && m.activeIdx == 2 {
				m.sendEmail = !m.sendEmail
			}
			return m, nil

		case "enter":
			if m.phase == phaseInputs {
				if m.activeIdx == 0 {
					pubVal := strings.TrimSpace(m.inputs[0].Value())
					if pubVal == "" {
						m.errLine = "Publication slug or domain is required"
						return m, nil
					}
					m.inputs[0].Blur()
					m.activeIdx = 1
					m.inputs[1].Focus()
				} else if m.activeIdx == 1 {
					cookieVal := strings.TrimSpace(m.inputs[1].Value())
					if cookieVal == "" {
						m.errLine = "Substack session cookie (substack.sid) is required"
						return m, nil
					}
					m.inputs[1].Blur()
					m.activeIdx = 2
				} else {
					pubVal := strings.TrimSpace(m.inputs[0].Value())
					cookieVal := strings.TrimSpace(m.inputs[1].Value())
					if pubVal == "" {
						m.errLine = "Publication slug or domain is required"
						m.activeIdx = 0
						m.inputs[0].Focus()
						return m, nil
					}
					if cookieVal == "" {
						m.errLine = "Substack session cookie (substack.sid) is required"
						m.activeIdx = 1
						m.inputs[1].Focus()
						return m, nil
					}
					m.phase = phaseConfirm
				}
				m.errLine = ""
				return m, nil
			} else if m.phase == phaseConfirm {
				pubVal := strings.TrimSpace(m.inputs[0].Value())
				cookieVal := strings.TrimSpace(m.inputs[1].Value())
				sendEmailVal := "false"
				if m.sendEmail {
					sendEmailVal = "true"
				}

				envMap := map[string]string{
					"POSTX_SUBSTACK_PUBLICATION": pubVal,
					"POSTX_SUBSTACK_COOKIE":      cookieVal,
					"POSTX_SUBSTACK_SEND_EMAIL":   sendEmailVal,
				}

				if err := m.deps.PersistEnv(envMap); err != nil {
					m.errLine = "Failed to save: " + err.Error()
					m.phase = phaseInputs
					return m, nil
				}
				m.phase = phaseSaved
				return m, nil
			} else if m.phase == phaseSaved {
				return m, tea.Quit
			}
		}
	}

	var cmds []tea.Cmd
	if m.phase == phaseInputs && m.activeIdx < 2 {
		var cmd tea.Cmd
		m.inputs[m.activeIdx], cmd = m.inputs[m.activeIdx].Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	var b strings.Builder
	p := theme.Current()

	b.WriteString(titleStyle().Render("Configure Substack Channel"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle().Render("Enter your Substack publication slug and cookie credentials"))
	b.WriteString("\n\n")

	if m.phase == phaseInputs {
		// Render inputs
		// 1. Publication
		pubLabel := "1. Substack Publication (Slug or Custom Domain)"
		if m.activeIdx == 0 {
			pubLabel = activeLabelStyle().Render("▸ " + pubLabel)
		} else {
			pubLabel = inactiveLabelStyle().Render("  " + pubLabel)
		}
		b.WriteString(pubLabel + "\n")
		b.WriteString(m.inputs[0].View() + "\n\n")

		// 2. Cookie
		cookieLabel := "2. Substack Session Cookie (substack.sid)"
		if m.activeIdx == 1 {
			cookieLabel = activeLabelStyle().Render("▸ " + cookieLabel)
		} else {
			cookieLabel = inactiveLabelStyle().Render("  " + cookieLabel)
		}
		b.WriteString(cookieLabel + "\n")
		b.WriteString(m.inputs[1].View() + "\n\n")

		// 3. Send Email
		emailLabel := "3. Automatically Email Subscribers on New Posts?"
		if m.activeIdx == 2 {
			emailLabel = activeLabelStyle().Render("▸ " + emailLabel)
		} else {
			emailLabel = inactiveLabelStyle().Render("  " + emailLabel)
		}
		b.WriteString(emailLabel + "\n")
		toggleText := " [ ] No (recommended for status updates - web draft only) "
		if m.sendEmail {
			toggleText = " [x] Yes (web + emails newsletter to all subscribers!) "
		}
		if m.activeIdx == 2 {
			toggleText = selectedToggleStyle().Render(toggleText)
		} else {
			toggleText = unselectedToggleStyle().Render(toggleText)
		}
		b.WriteString(toggleText + "  (Use Left/Right keys to toggle)\n\n")

		b.WriteString(hintStyle().Render("Tab/Shift+Tab: switch fields · Enter: continue · Esc: cancel"))
		b.WriteString("\n")
	} else if m.phase == phaseConfirm {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Accent).Bold(true).Render("Please confirm your settings:") + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  Publication: ") + m.inputs[0].Value() + "\n")
		maskedCookie := m.inputs[1].Value()
		if len(maskedCookie) > 8 {
			maskedCookie = maskedCookie[:8] + "..."
		}
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  Cookie:      ") + maskedCookie + "\n")
		emailStatus := "No (Publish to Web Only)"
		if m.sendEmail {
			emailStatus = "Yes (Web + Email Newsletter)"
		}
		b.WriteString(lipgloss.NewStyle().Foreground(p.Muted).Render("  Email subs:  ") + emailStatus + "\n\n")

		b.WriteString(lipgloss.NewStyle().Foreground(p.Accent2).Bold(true).Render("▸ Press ENTER to save these credentials to config and shell profile") + "\n")
		b.WriteString(hintStyle().Render("Esc: cancel / go back"))
		b.WriteString("\n")
	} else if m.phase == phaseSaved {
		b.WriteString(lipgloss.NewStyle().Foreground(p.Lime).Bold(true).Render("✓ Substack credentials saved successfully!") + "\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(p.Text).Render("The environment variables have been updated and synced with your shell profile.") + "\n")
		b.WriteString(hintStyle().Render("Press ENTER to return to the channels menu"))
		b.WriteString("\n")
	}

	if m.errLine != "" {
		b.WriteString("\n")
		b.WriteString(errStyle().Render("Error: " + m.errLine) + "\n")
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func titleStyle() lipgloss.Style {
	p := theme.Current()
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Accent).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderForeground(p.Border).
		PaddingLeft(1)
}

func subtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Muted).Italic(true)
}

func activeLabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Accent2).Bold(true)
}

func inactiveLabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Muted)
}

func selectedToggleStyle() lipgloss.Style {
	p := theme.Current()
	return lipgloss.NewStyle().Foreground(p.Text).Background(p.MenuBG).Bold(true)
}

func unselectedToggleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Muted)
}

func hintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Muted).Italic(true)
}

func errStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(theme.Current().Rose).Bold(true)
}
