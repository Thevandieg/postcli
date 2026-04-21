package channelsui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"postcli/internal/channels"
	"postcli/internal/store"
	"postcli/internal/theme"
)

// Action is returned after the TUI exits.
type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionConfigureX
)

// Run shows the channel list; use Enter on a row to choose an action.
func Run(stats []channels.Status) (Action, error) {
	_ = theme.Load()
	m := model{
		stats:  stats,
		cursor: 0,
	}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return ActionNone, err
	}
	out, ok := final.(model)
	if !ok {
		return ActionNone, nil
	}
	return out.action, nil
}

type model struct {
	stats   []channels.Status
	cursor  int
	errLine string
	action  Action

	width  int
	height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.action = ActionQuit
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.errLine = ""
			return m, nil
		case "down", "j":
			if m.cursor < len(m.stats)-1 {
				m.cursor++
			}
			m.errLine = ""
			return m, nil
		case "enter":
			return m.activate()
		}
	}
	return m, nil
}

func (m model) activate() (tea.Model, tea.Cmd) {
	if len(m.stats) == 0 {
		m.errLine = "No channels."
		return m, nil
	}
	s := m.stats[m.cursor]
	switch s.Entry.ID {
	case store.ChannelX:
		m.action = ActionConfigureX
		return m, tea.Quit
	default:
		m.errLine = fmt.Sprintf("%s is preview-only — integration coming later.", s.Entry.Title)
		return m, nil
	}
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle().Render("postx — channels"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle().Render("↑/↓ j/k: move · enter: configure · esc/q: quit"))
	b.WriteString("\n\n")

	for i, s := range m.stats {
		line := rowLine(i == m.cursor, s)
		b.WriteString(line)
		b.WriteString("\n")
	}

	if m.errLine != "" {
		b.WriteString("\n")
		b.WriteString(errStyle().Render(m.errLine))
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func rowLine(selected bool, s channels.Status) string {
	state := "not configured"
	if s.Configured {
		state = "configured"
	}
	label := s.Entry.Title
	if s.Entry.Subtitle != "" {
		label += " · " + s.Entry.Subtitle
	}
	line := fmt.Sprintf("[%s] %s — %s", state, label, s.Detail)
	if selected {
		return cursorStyle().Render("▸ ") + menuSelStyle().Render(" "+line+" ")
	}
	return menuIdleStyle().Render("    ") + menuIdleStyle().Render(line)
}

func p() theme.Palette {
	return theme.Current()
}

func titleStyle() lipgloss.Style {
	x := p()
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(x.Accent).
		BorderStyle(lipgloss.Border{Left: "│"}).
		BorderForeground(x.Border).
		PaddingLeft(1)
}

func subtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Muted).Italic(true)
}

func errStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Rose).Bold(true)
}

func cursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Accent2).Bold(true)
}

func menuSelStyle() lipgloss.Style {
	x := p()
	return lipgloss.NewStyle().
		Foreground(x.Text).
		Background(x.MenuBG).
		Padding(0, 1).
		Bold(true)
}

func menuIdleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Muted)
}
