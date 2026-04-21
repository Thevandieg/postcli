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
	"postcli/internal/theme"
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
	_ = theme.Load()
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
	applyBubbleStyles(&ti, &ta, &fp, &sp)

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
				return submitErrMsg{err: fmt.Errorf("%s", xapi.UserMessage(err))}
			}
			post, err := m.store.GetPost(ctx, id)
			if err != nil {
				return submitErrMsg{err: err}
			}
			if post.Status != store.StatusPosted {
				msg := strings.TrimSpace(post.LastError)
				if msg == "" {
					msg = fmt.Sprintf("post stayed %s (not published)", post.Status)
				}
				return submitErrMsg{err: fmt.Errorf("%s", xapi.UserMessage(fmt.Errorf("%s", msg)))}
			}
			summary = fmt.Sprintf("Success: posted #%d to X (tweet %s).", id, post.TweetID)
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
	writeHeader := func() {
		b.WriteString(headerBlock(m.step))
		b.WriteString("\n\n")
	}

	if m.working {
		writeHeader()
		line := lipgloss.JoinHorizontal(lipgloss.Center, m.spinner.View(), "  ", okStyle().Render("Submitting…"))
		b.WriteString(line)
		b.WriteString("\n")
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	if m.doneMsg != "" {
		writeHeader()
		b.WriteString(framedBlock(okStyle().Render(m.doneMsg)))
		b.WriteString("\n")
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	writeHeader()
	if m.err != "" {
		b.WriteString(errStyle().Render(m.err))
		b.WriteString("\n\n")
	}

	switch m.step {
	case stepKind:
		items := []string{"Text only", "Text + image (jpg/png/gif/webp)"}
		for i, it := range items {
			b.WriteString(menuLine(m.kindCursor == i, it))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(hintLine("↑/↓ j/k: move · enter: next · esc: quit"))

	case stepWhen:
		opts := []string{"Post now (queue + flush immediately)", "Schedule for later"}
		for i, it := range opts {
			b.WriteString(menuLine(m.whenCursor == i, it))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(hintLine("↑/↓ j/k: move · enter: continue"))

	case stepSchedule:
		b.WriteString(framedBlock(
			subtitleStyle().Render("Local time") + "\n" + dimTextStyle().Render("Format: 2006-01-02 15:04") + "\n\n" + m.scheduleInput.View(),
		))
		b.WriteString("\n")
		b.WriteString(hintLine("enter: continue when valid"))

	case stepBody:
		b.WriteString(framedBlock(m.body.View()))
		b.WriteString("\n")
		b.WriteString(hintLine("ctrl+d: done composing · esc: quit (when not editing)"))

	case stepMedia:
		b.WriteString(framedBlock(m.picker.View()))
		b.WriteString("\n")
		b.WriteString(hintLine("enter: pick file · h/esc: parent · q: quit"))

	case stepConfirm:
		kind := "Text only"
		if m.kindCursor == 1 {
			kind = "Text + media"
		}
		when := "Immediately (post after save)"
		if m.whenCursor == 1 {
			t, _ := m.parsedSchedule()
			when = t.Format(time.RFC3339)
		}
		var sum strings.Builder
		sum.WriteString(labelStyle().Render("Kind "))
		sum.WriteString(valueStyle().Render(kind))
		sum.WriteString("\n")
		sum.WriteString(labelStyle().Render("When "))
		sum.WriteString(valueStyle().Render(when))
		sum.WriteString("\n")
		if m.kindCursor == 1 {
			sum.WriteString(labelStyle().Render("File "))
			sum.WriteString(valueStyle().Render(m.picker.Path))
			sum.WriteString("\n")
		}
		preview := strings.TrimSpace(m.body.Value())
		if len(preview) > 160 {
			preview = preview[:160] + "…"
		}
		sum.WriteString("\n")
		sum.WriteString(subtitleStyle().Render("Preview"))
		sum.WriteString("\n")
		sum.WriteString(mutedItalicStyle().Render(preview))

		b.WriteString(framedBlock(sum.String()))
		b.WriteString("\n")
		b.WriteString(hintLine("y / enter: submit · n / h / ← : back to edit"))
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
