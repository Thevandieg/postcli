package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"postcli/internal/store"
	"postcli/internal/theme"
)

type model struct {
	store      *store.Store
	month      time.Time
	day        time.Time
	monthPosts []store.Post

	vp     viewport.Model
	width  int
	height int
}

// Run shows the calendar status TUI.
func Run(st *store.Store) error {
	_ = theme.Load()
	now := time.Now().UTC()
	m := &model{
		store: st,
		month: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		day:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		vp: viewport.New(viewport.WithWidth(72), viewport.WithHeight(14)),
	}
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.vp.SetWidth(max(40, msg.Width-36))
		m.vp.SetHeight(max(8, msg.Height-8))
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		// One day (calendar grid left/right)
		case "left", "h":
			m.addDays(-1)
			return m, nil
		case "right", "l":
			m.addDays(1)
			return m, nil
		// One week (calendar grid up/down)
		case "up", "k":
			m.addDays(-7)
			return m, nil
		case "down", "j":
			m.addDays(7)
			return m, nil
		// Month
		case "[":
			m.month = m.month.AddDate(0, -1, 0)
			m.day = clampDay(m.day, m.month)
			m.refreshDetail()
			return m, nil
		case "]":
			m.month = m.month.AddDate(0, 1, 0)
			m.day = clampDay(m.day, m.month)
			m.refreshDetail()
			return m, nil
		// Jump to today (UTC date)
		case "t", "T":
			now := time.Now().UTC()
			m.day = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			m.month = time.Date(m.day.Year(), m.day.Month(), 1, 0, 0, 0, 0, time.UTC)
			m.refreshDetail()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m *model) addDays(delta int) {
	m.day = m.day.AddDate(0, 0, delta)
	m.month = time.Date(m.day.Year(), m.day.Month(), 1, 0, 0, 0, 0, time.UTC)
	m.refreshDetail()
}

func clampDay(day, month time.Time) time.Time {
	y, mo, d := day.Date()
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	if y != month.Year() || mo != month.Month() {
		return first
	}
	if d > last.Day() {
		return last
	}
	return time.Date(month.Year(), month.Month(), d, 0, 0, 0, 0, time.UTC)
}

func (m *model) refreshDetail() {
	ctx := context.Background()
	mp, err := m.store.ListPostsInMonth(ctx, m.month)
	if err != nil {
		m.monthPosts = nil
		m.vp.SetContent(err.Error())
		return
	}
	m.monthPosts = mp
	posts := postsOnDay(m.monthPosts, m.day)
	var b strings.Builder
	if len(posts) == 0 {
		b.WriteString("(no posts this day)")
	} else {
		x := theme.Current()
		for _, p := range posts {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(x.Accent).Render(fmt.Sprintf("#%d · %s · %s", p.ID, p.Kind, p.Status)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(x.Muted).Render(fmt.Sprintf("  %s UTC", p.ScheduledAt.UTC().Format("15:04:05"))) + "\n")
			if p.TweetID != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(x.Dim).Render(fmt.Sprintf("  tweet: %s", p.TweetID)) + "\n")
			}
			if p.LastError != "" {
				b.WriteString(lipgloss.NewStyle().Foreground(x.Rose).Render("  err: "+p.LastError) + "\n")
			}
			txt := strings.TrimSpace(p.Payload.Text)
			if len(txt) > 200 {
				txt = txt[:200] + "…"
			}
			b.WriteString(lipgloss.NewStyle().Foreground(x.Text).Render(fmt.Sprintf("  %s", txt)) + "\n\n")
		}
	}
	m.vp.SetContent(b.String())
}

func (m *model) View() tea.View {
	m.refreshDetail()
	x := theme.Current()
	header := lipgloss.NewStyle().Bold(true).Foreground(x.Accent).BorderStyle(lipgloss.Border{Left: "│"}).BorderForeground(x.Border).PaddingLeft(1).Render("postx — schedule")
	sub := lipgloss.NewStyle().Foreground(x.Muted).Render(
		m.month.Format("January 2006") + " · selected " + m.day.Format("2006-01-02") + " UTC",
	)
	cal := m.renderCalendar()
	help := lipgloss.NewStyle().Foreground(x.Dim).Italic(true).Render(
		"←/→ h/l: day · ↑/↓ j/k: week · [/]: month · t: today · q: quit",
	)

	left := lipgloss.JoinVertical(lipgloss.Left, header, "", sub, "", cal, "", help)
	right := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(x.Border).
		Padding(0, 1).
		Render(m.vp.View())

	row := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	v := tea.NewView(row)
	v.AltScreen = true
	return v
}

func postsOnDay(monthPosts []store.Post, day time.Time) []store.Post {
	var out []store.Post
	for _, p := range monthPosts {
		py, pm, pd := p.ScheduledAt.UTC().Date()
		dy, dm, dd := day.Date()
		if py == dy && pm == dm && pd == dd {
			out = append(out, p)
		}
	}
	return out
}

func (m *model) renderCalendar() string {
	x := theme.Current()
	first := m.month
	weekday := int(first.Weekday())
	start := first.AddDate(0, 0, -weekday)

	dim := lipgloss.NewStyle().Foreground(x.Dim)
	today := lipgloss.NewStyle().Foreground(x.Accent)
	sel := lipgloss.NewStyle().Foreground(x.Accent2).Bold(true)
	normal := lipgloss.NewStyle().Foreground(x.Text)
	mark := lipgloss.NewStyle().Foreground(x.Amber)

	var rows []string
	hdr := "Su Mo Tu We Th Fr Sa"
	rows = append(rows, dim.Render(hdr))

	now := time.Now().UTC()
	for w := 0; w < 6; w++ {
		var cells []string
		for d := 0; d < 7; d++ {
			day := start.AddDate(0, 0, w*7+d)
			label := fmt.Sprintf("%2d", day.Day())
			st := normal
			if day.Month() != m.month.Month() {
				st = dim
			}
			if sameDate(day, now) {
				st = today
			}
			if sameDate(day, m.day) {
				st = sel
			}
			if len(postsOnDay(m.monthPosts, day)) > 0 {
				label = mark.Render("·") + st.Render(fmt.Sprintf("%d", day.Day()))
			} else {
				label = st.Render(label)
			}
			cells = append(cells, label)
		}
		rows = append(rows, strings.Join(cells, " "))
	}
	return strings.Join(rows, "\n")
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
