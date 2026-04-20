package post

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"postcli/internal/schedule"
	"postcli/internal/store"
	"postcli/internal/xapi"
)

type step int

const (
	stepKind step = iota
	stepWhen
	stepSchedule
	stepBody
	stepMedia
	stepConfirm
)

type model struct {
	step step

	kindCursor int
	whenCursor int // 0 now, 1 schedule

	scheduleInput textinput.Model
	body          textarea.Model
	picker        filepicker.Model
	spinner       spinner.Model

	store  *store.Store
	client *xapi.Client
	runner *schedule.Runner

	width  int
	height int
	err    string

	working bool
	doneMsg string
}

// Run launches the post wizard TUI.
func Run(st *store.Store, client *xapi.Client, runner *schedule.Runner) error {
	ti := textinput.New()
	ti.Placeholder = "2006-01-02 15:04 (local)"
	ti.CharLimit = 32
	ti.SetWidth(40)

	ta := textarea.New()
	ta.Placeholder = "What's happening?"
	ta.SetWidth(56)
	ta.SetHeight(6)
	ta.CharLimit = 280

	fp := filepicker.New()
	fp.AllowedTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	fp.CurrentDirectory = "."

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		step:          stepKind,
		scheduleInput: ti,
		body:          ta,
		picker:        fp,
		spinner:       sp,
		store:         st,
		client:        client,
		runner:        runner,
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func blinkInput() tea.Cmd {
	return func() tea.Msg { return textinput.Blink() }
}

func blinkTA() tea.Cmd {
	return func() tea.Msg { return textarea.Blink() }
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.body.SetWidth(min(56, msg.Width-8))
		m.picker.SetHeight(max(8, msg.Height-12))
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step == stepMedia || (m.step == stepBody && m.body.Focused()) {
				break
			}
			return m, tea.Quit
		}

	case submitErrMsg:
		m.working = false
		if msg.err != nil {
			m.err = msg.err.Error()
			return m, nil
		}
		m.doneMsg = msg.summary
		fmt.Fprintln(os.Stderr, msg.summary)
		return m, tea.Quit

	case spinner.TickMsg:
		if m.working {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	if m.working {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.step {
	case stepKind:
		return m.updateKind(msg)
	case stepWhen:
		return m.updateWhen(msg)
	case stepSchedule:
		return m.updateSchedule(msg)
	case stepBody:
		return m.updateBody(msg)
	case stepMedia:
		return m.updateMedia(msg)
	case stepConfirm:
		return m.updateConfirm(msg)
	}
	return m, nil
}

func (m model) updateKind(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up", "k":
			if m.kindCursor > 0 {
				m.kindCursor--
			}
			return m, nil
		case "down", "j":
			if m.kindCursor < 1 {
				m.kindCursor++
			}
			return m, nil
		case "enter":
			m.step = stepWhen
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateWhen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "up", "k":
			if m.whenCursor > 0 {
				m.whenCursor--
			}
			return m, nil
		case "down", "j":
			if m.whenCursor < 1 {
				m.whenCursor++
			}
			return m, nil
		case "enter":
			if m.whenCursor == 1 {
				m.step = stepSchedule
				m.scheduleInput.Focus()
				return m, blinkInput()
			}
			m.step = stepBody
			m.body.Focus()
			return m, blinkTA()
		}
	}
	return m, nil
}

func (m model) updateSchedule(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			if _, err := m.parsedSchedule(); err != nil {
				m.err = err.Error()
				return m, nil
			}
			m.err = ""
			m.step = stepBody
			m.body.Focus()
			return m, blinkTA()
		}
	}
	var cmd tea.Cmd
	m.scheduleInput, cmd = m.scheduleInput.Update(msg)
	return m, cmd
}

func (m model) updateBody(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if k.String() == "ctrl+d" {
			txt := strings.TrimSpace(m.body.Value())
			if txt == "" {
				m.err = "text required"
				return m, nil
			}
			m.err = ""
			if m.kindCursor == 1 {
				m.step = stepMedia
				m.picker.Path = ""
				return m, m.picker.Init()
			}
			m.step = stepConfirm
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(msg)
	return m, cmd
}

func (m model) updateMedia(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevPath := m.picker.Path
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	if m.picker.Path != "" && m.picker.Path != prevPath {
		m.step = stepConfirm
		return m, nil
	}
	return m, cmd
}

func (m model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "n", "left", "h":
			m.step = stepBody
			m.body.Focus()
			return m, blinkTA()
		case "y", "enter":
			m.err = ""
			m.working = true
			return m, tea.Batch(
				func() tea.Msg { return m.spinner.Tick() },
				m.submitCmd(),
			)
		}
	}
	return m, nil
}

type submitErrMsg struct {
	err     error
	summary string
}

func (m model) submitCmd() tea.Cmd {
	return func() tea.Msg {
		kind := store.KindText
		if m.kindCursor == 1 {
			kind = store.KindTextWithMedia
		}
		payload := store.PostPayload{Text: strings.TrimSpace(m.body.Value())}
		if kind == store.KindTextWithMedia {
			payload.MediaPath = m.picker.Path
		}
		var sched time.Time
		var err error
		if m.whenCursor == 0 {
			sched = time.Now().UTC()
		} else {
			sched, err = m.parsedSchedule()
			if err != nil {
				return submitErrMsg{err: err}
			}
			sched = sched.UTC()
		}
		ctx := context.Background()
		id, err := m.store.InsertPost(ctx, kind, payload, sched, store.StatusPending, "")
		if err != nil {
			return submitErrMsg{err: err}
		}
		summary := fmt.Sprintf("Queued post #%d for %s", id, sched.Format(time.RFC3339))
		if m.whenCursor == 0 {
			if err := m.runner.FlushDue(ctx, time.Now().UTC()); err != nil {
				return submitErrMsg{err: fmt.Errorf("queued but flush failed: %w", err)}
			}
			summary = fmt.Sprintf("Posted #%d", id)
		}
		return submitErrMsg{summary: summary}
	}
}

func (m model) parsedSchedule() (time.Time, error) {
	loc := time.Local
	s := strings.TrimSpace(m.scheduleInput.Value())
	if s == "" {
		return time.Time{}, fmt.Errorf("enter date/time")
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("use format 2006-01-02 15:04 (local)")
	}
	return t, nil
}

func (m model) View() tea.View {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252")).Render("postx — compose")
	b.WriteString(title)
	b.WriteString("\n\n")

	if m.working {
		b.WriteString(m.spinner.View())
		b.WriteString(" working…\n")
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	if m.doneMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render(m.doneMsg))
		b.WriteString("\n")
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	if m.err != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(m.err))
		b.WriteString("\n\n")
	}

	switch m.step {
	case stepKind:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Content type") + "\n")
		items := []string{"Text only", "Text + image (jpg/png/gif/webp)"}
		for i, it := range items {
			cursor := "  "
			if m.kindCursor == i {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, it))
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("enter: next · q/esc: quit"))

	case stepWhen:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("When") + "\n")
		opts := []string{"Post now (queue + flush immediately)", "Schedule for later"}
		for i, it := range opts {
			cursor := "  "
			if m.whenCursor == i {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, it))
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("enter: next"))

	case stepSchedule:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Schedule (local time)") + "\n\n")
		b.WriteString(m.scheduleInput.View())
		b.WriteString("\n\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("enter: continue"))

	case stepBody:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Compose (ctrl+d: done)") + "\n\n")
		b.WriteString(m.body.View())
		b.WriteString("\n")

	case stepMedia:
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Pick media · enter: select file") + "\n\n")
		b.WriteString(m.picker.View())

	case stepConfirm:
		kind := "text"
		if m.kindCursor == 1 {
			kind = "text + media"
		}
		when := "now"
		if m.whenCursor == 1 {
			t, _ := m.parsedSchedule()
			when = t.Format(time.RFC3339)
		}
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Confirm") + "\n\n")
		b.WriteString(fmt.Sprintf("Kind: %s\nWhen: %s\n", kind, when))
		if m.kindCursor == 1 {
			b.WriteString(fmt.Sprintf("File: %s\n", m.picker.Path))
		}
		preview := strings.TrimSpace(m.body.Value())
		if len(preview) > 160 {
			preview = preview[:160] + "…"
		}
		b.WriteString(fmt.Sprintf("\n%s\n\n", preview))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("y/enter: submit · n: back"))
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
