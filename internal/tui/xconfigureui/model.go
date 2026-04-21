package xconfigureui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"postcli/internal/theme"
)

type oauthDoneMsg struct {
	err error
}

type phase int

const (
	phaseMenu phase = iota
	phaseInput
	phaseYesNo
	phaseWorking
)

type formKind int

const (
	formNone formKind = iota
	formFull
	formID
	formSecret
	formRedirect
)

// ynKind drives the yes/no follow-up state machine.
type ynKind int

const (
	ynNone ynKind = iota
	ynFullBrowser
	ynOAuthNow
	ynOpenBrowser
)

type model struct {
	deps Deps

	phase      phase
	menuCursor int
	errLine    string
	infoLine   string

	form     formKind
	formStep int
	draftID  string
	draftSec string
	draftRed string
	commitID string
	commitSec string

	ti textinput.Model

	ynKind     ynKind
	ynQuestion string
	ynCursor   int

	width int
}

// Run starts the full-screen X configuration TUI.
func Run(d Deps) error {
	_ = theme.Load()
	m := NewModel(d)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func blinkTI() tea.Cmd {
	return func() tea.Msg { return textinput.Blink() }
}

func NewModel(d Deps) model {
	ti := textinput.New()
	applyInputStyles(&ti)
	return model{
		deps:       d,
		phase:      phaseMenu,
		menuCursor: 0,
		ti:         ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.ti.SetWidth(min(72, max(40, msg.Width-10)))
		return m, nil

	case oauthDoneMsg:
		m.phase = phaseMenu
		if msg.err != nil {
			m.errLine = msg.err.Error()
			m.infoLine = ""
		} else {
			m.errLine = ""
			m.infoLine = "OAuth login complete."
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.phase == phaseWorking {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

		switch m.phase {
		case phaseMenu:
			return m.updateMenu(msg)
		case phaseInput:
			return m.updateInput(msg)
		case phaseYesNo:
			return m.updateYesNo(msg)
		}
	}

	if m.phase == phaseInput {
		var cmd tea.Cmd
		m.ti, cmd = m.ti.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) updateMenu(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		if m.infoLine != "" || m.errLine != "" {
			m.infoLine, m.errLine = "", ""
			return m, nil
		}
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
		m.errLine, m.infoLine = "", ""
		return m, nil
	case "down", "j":
		if m.menuCursor < 6 {
			m.menuCursor++
		}
		m.errLine, m.infoLine = "", ""
		return m, nil
	case "enter":
		return m.activateMenu()
	}
	return m, nil
}

func (m model) activateMenu() (tea.Model, tea.Cmd) {
	m.errLine, m.infoLine = "", ""
	switch m.menuCursor {
	case 0:
		return m.beginFullSetup()
	case 1:
		return m.beginSetID()
	case 2:
		return m.beginSetSecret()
	case 3:
		return m.startOAuth(false)
	case 4:
		return m.startOAuth(true)
	case 5:
		return m.beginSetRedirect()
	case 6:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) beginFullSetup() (tea.Model, tea.Cmd) {
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.form = formFull
	m.formStep = 0
	m.draftID = firstNonEmpty(strings.TrimSpace(envMap["POSTX_CLIENT_ID"]), m.deps.ClientID())
	m.draftSec = firstNonEmpty(strings.TrimSpace(envMap["POSTX_CLIENT_SECRET"]), m.deps.ClientSecret())
	m.draftRed = firstNonEmpty(strings.TrimSpace(envMap["POSTX_REDIRECT_URI"]), m.deps.RedirectURI())
	if m.draftRed == "" {
		m.draftRed = "http://127.0.0.1:8080/callback"
	}
	m.commitID, m.commitSec = "", ""
	m.phase = phaseInput
	m.ti.SetValue("")
	m.ti.Placeholder = placeholderID(m.draftID)
	m.ti.Focus()
	return m, blinkTI()
}

func (m model) beginSetID() (tea.Model, tea.Cmd) {
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.form = formID
	m.formStep = 0
	m.draftID = firstNonEmpty(strings.TrimSpace(envMap["POSTX_CLIENT_ID"]), m.deps.ClientID())
	m.phase = phaseInput
	m.ti.SetValue("")
	m.ti.Placeholder = placeholderID(m.draftID)
	m.ti.Focus()
	return m, blinkTI()
}

func (m model) beginSetSecret() (tea.Model, tea.Cmd) {
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.form = formSecret
	m.formStep = 0
	m.draftSec = firstNonEmpty(strings.TrimSpace(envMap["POSTX_CLIENT_SECRET"]), m.deps.ClientSecret())
	m.phase = phaseInput
	m.ti.SetValue("")
	m.ti.Placeholder = placeholderSecret(m.draftSec)
	m.ti.Focus()
	return m, blinkTI()
}

func (m model) beginSetRedirect() (tea.Model, tea.Cmd) {
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.form = formRedirect
	m.formStep = 0
	m.draftRed = firstNonEmpty(strings.TrimSpace(envMap["POSTX_REDIRECT_URI"]), m.deps.RedirectURI())
	if m.draftRed == "" {
		m.draftRed = "http://127.0.0.1:8080/callback"
	}
	m.phase = phaseInput
	m.ti.SetValue("")
	m.ti.Placeholder = m.draftRed
	m.ti.Focus()
	return m, blinkTI()
}

func (m model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = phaseMenu
		m.form = formNone
		m.errLine = ""
		return m, nil
	case "enter":
		return m.submitInput()
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m model) submitInput() (tea.Model, tea.Cmd) {
	v := strings.TrimSpace(m.ti.Value())
	switch m.form {
	case formFull:
		return m.submitFullStep(v)
	case formID:
		return m.submitSetID(v)
	case formSecret:
		return m.submitSetSecret(v)
	case formRedirect:
		return m.submitSetRedirect(v)
	}
	return m, nil
}

func (m model) submitFullStep(v string) (tea.Model, tea.Cmd) {
	switch m.formStep {
	case 0:
		id := v
		if id == "" {
			id = m.draftID
		}
		if id == "" {
			m.errLine = "Client ID is required (or leave empty only if one is already saved)"
			return m, nil
		}
		m.commitID = id
		m.formStep = 1
		m.ti.SetValue("")
		m.ti.Placeholder = placeholderSecret(m.draftSec)
		return m, blinkTI()
	case 1:
		sec := v
		if sec == "" {
			sec = m.draftSec
		}
		if sec == "" {
			m.errLine = "Client Secret is required (or keep existing with empty line)"
			return m, nil
		}
		m.commitSec = sec
		m.formStep = 2
		m.ti.SetValue("")
		m.ti.Placeholder = m.draftRed
		return m, blinkTI()
	case 2:
		red := v
		if red == "" {
			red = m.draftRed
		}
		envMap, err := m.deps.LoadEnvMap()
		if err != nil {
			m.errLine = err.Error()
			return m, nil
		}
		envMap["POSTX_CLIENT_ID"] = m.commitID
		envMap["POSTX_CLIENT_SECRET"] = m.commitSec
		envMap["POSTX_REDIRECT_URI"] = red
		if err := m.deps.PersistEnv(envMap); err != nil {
			m.errLine = err.Error()
			return m, nil
		}
		m.deps.ApplyEnv(envMap)
		m.form = formNone
		m.phase = phaseYesNo
		m.ynKind = ynFullBrowser
		m.startYesNo("Open browser automatically for OAuth?", true)
		return m, nil
	}
	return m, nil
}

func (m model) submitSetID(v string) (tea.Model, tea.Cmd) {
	id := v
	if id == "" {
		id = m.draftID
	}
	if id == "" {
		m.errLine = "Client ID cannot be empty"
		return m, nil
	}
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	envMap["POSTX_CLIENT_ID"] = id
	if strings.TrimSpace(envMap["POSTX_REDIRECT_URI"]) == "" {
		envMap["POSTX_REDIRECT_URI"] = m.deps.RedirectURI()
	}
	if err := m.deps.PersistEnv(envMap); err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.deps.ApplyEnv(envMap)
	m.form = formNone
	m.infoLine = "Saved Client ID."
	m.phase = phaseYesNo
	m.ynKind = ynOAuthNow
	m.startYesNo("Run OAuth login now?", false)
	return m, nil
}

func (m model) submitSetSecret(v string) (tea.Model, tea.Cmd) {
	sec := v
	if sec == "" {
		sec = m.draftSec
	}
	if sec == "" {
		m.errLine = "Client Secret cannot be empty"
		return m, nil
	}
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	envMap["POSTX_CLIENT_SECRET"] = sec
	if strings.TrimSpace(envMap["POSTX_REDIRECT_URI"]) == "" {
		envMap["POSTX_REDIRECT_URI"] = m.deps.RedirectURI()
	}
	if err := m.deps.PersistEnv(envMap); err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.deps.ApplyEnv(envMap)
	m.form = formNone
	m.infoLine = "Saved Client Secret."
	m.phase = phaseYesNo
	m.ynKind = ynOAuthNow
	m.startYesNo("Run OAuth login now?", false)
	return m, nil
}

func (m model) submitSetRedirect(v string) (tea.Model, tea.Cmd) {
	red := v
	if red == "" {
		red = m.draftRed
	}
	envMap, err := m.deps.LoadEnvMap()
	if err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	envMap["POSTX_REDIRECT_URI"] = red
	if err := m.deps.PersistEnv(envMap); err != nil {
		m.errLine = err.Error()
		return m, nil
	}
	m.deps.ApplyEnv(envMap)
	m.form = formNone
	m.phase = phaseMenu
	m.infoLine = "Saved redirect URI."
	return m, nil
}

func (m model) startYesNo(q string, defaultYes bool) {
	m.ynQuestion = q
	if defaultYes {
		m.ynCursor = 0
	} else {
		m.ynCursor = 1
	}
}

func (m model) updateYesNo(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.phase = phaseMenu
		m.ynKind = ynNone
		return m, nil
	case "up", "k", "left", "h":
		if m.ynCursor > 0 {
			m.ynCursor--
		}
		return m, nil
	case "down", "j", "right", "l":
		if m.ynCursor < 1 {
			m.ynCursor++
		}
		return m, nil
	case "tab":
		m.ynCursor = 1 - m.ynCursor
		return m, nil
	case "enter":
		return m.submitYesNo()
	}
	return m, nil
}

func (m model) submitYesNo() (tea.Model, tea.Cmd) {
	yes := m.ynCursor == 0
	switch m.ynKind {
	case ynFullBrowser:
		m.ynKind = ynNone
		skip := !yes
		return m.startOAuthJob(skip)
	case ynOAuthNow:
		if !yes {
			m.ynKind = ynNone
			m.phase = phaseMenu
			return m, nil
		}
		m.ynKind = ynOpenBrowser
		m.startYesNo("Open browser automatically?", true)
		return m, nil
	case ynOpenBrowser:
		m.ynKind = ynNone
		skip := !yes
		return m.startOAuthJob(skip)
	}
	return m, nil
}

func (m model) startOAuth(skipBrowser bool) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(m.deps.ClientID()) == "" || strings.TrimSpace(m.deps.ClientSecret()) == "" {
		m.errLine = "Set Client ID and Secret first (menu items 1–2 or Full setup)."
		return m, nil
	}
	return m.startOAuthJob(skipBrowser)
}

func (m model) startOAuthJob(skipBrowser bool) (tea.Model, tea.Cmd) {
	m.phase = phaseWorking
	m.errLine = ""
	ctx := m.deps.Ctx
	deps := m.deps
	return m, func() tea.Msg {
		err := deps.OAuthLogin(ctx, skipBrowser)
		return oauthDoneMsg{err: err}
	}
}

func (m model) View() tea.View {
	var b strings.Builder
	b.WriteString(titleStyle().Render("postx — configure X (Twitter)"))
	b.WriteString("\n")
	b.WriteString(dimStyle().Render(statusBlock(m)))
	b.WriteString("\n\n")

	switch m.phase {
	case phaseMenu:
		b.WriteString(subtitleStyle().Render("Choose an action"))
		b.WriteString("\n\n")
		items := menuLabels()
		for i, it := range items {
			line := it
			if i == m.menuCursor {
				b.WriteString(cursorStyle().Render("▸ ") + selStyle().Render(" "+line+" "))
			} else {
				b.WriteString(idleStyle().Render("    " + line))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(hintStyle().Render("↑/↓ j/k · enter · esc/q quit"))
	case phaseInput:
		b.WriteString(subtitleStyle().Render(inputTitle(m.form, m.formStep)))
		b.WriteString("\n\n")
		b.WriteString(frameStyle().Render(innerPad().Render(m.ti.View())))
		b.WriteString("\n\n")
		b.WriteString(hintStyle().Render("enter: continue · esc: back to menu"))
	case phaseYesNo:
		b.WriteString(subtitleStyle().Render(m.ynQuestion))
		b.WriteString("\n\n")
		yesSel := m.ynCursor == 0
		b.WriteString(ynLine(yesSel, "Yes"))
		b.WriteString("\n")
		b.WriteString(ynLine(!yesSel, "No"))
		b.WriteString("\n\n")
		b.WriteString(hintStyle().Render("↑/↓ j/k · tab: toggle · enter · esc: menu"))
	case phaseWorking:
		b.WriteString(subtitleStyle().Render("OAuth in progress…"))
		b.WriteString("\n")
		b.WriteString(dimStyle().Render("Complete authorization in the browser, then return here."))
	}

	if m.infoLine != "" {
		b.WriteString("\n\n")
		b.WriteString(okStyle().Render(m.infoLine))
	}
	if m.errLine != "" {
		b.WriteString("\n\n")
		b.WriteString(errStyle().Render(m.errLine))
	}

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func menuLabels() []string {
	return []string{
		"1. Full setup: Client ID + Secret + redirect + OAuth",
		"2. Set Client ID only (empty input keeps current)",
		"3. Set Client Secret only (empty keeps current)",
		"4. OAuth login — open browser",
		"5. OAuth login — copy URL only (no auto-open browser)",
		"6. Set redirect URI",
		"7. Done / back",
	}
}

func inputTitle(f formKind, step int) string {
	switch f {
	case formFull:
		switch step {
		case 0:
			return "Client ID"
		case 1:
			return "Client Secret"
		case 2:
			return "Redirect URI"
		}
	case formID:
		return "Client ID"
	case formSecret:
		return "Client Secret"
	case formRedirect:
		return "Redirect URI"
	}
	return ""
}

func statusBlock(m model) string {
	var b strings.Builder
	cid := m.deps.ClientID()
	if cid == "" {
		b.WriteString("Client ID:     (not set)\n")
	} else {
		fmt.Fprintf(&b, "Client ID:     %s\n", maskMiddle(cid))
	}
	if m.deps.ClientSecret() == "" {
		b.WriteString("Client Secret: (not set)\n")
	} else {
		b.WriteString("Client Secret: (set)\n")
	}
	fmt.Fprintf(&b, "Redirect URI:  %s\n", m.deps.RedirectURI())
	has, det, err := m.deps.LoginStatus(m.deps.Ctx)
	if err != nil {
		fmt.Fprintf(&b, "Login:         (error: %v)\n", err)
	} else if has {
		fmt.Fprintf(&b, "Login:         %s\n", det)
	} else {
		fmt.Fprintf(&b, "Login:         %s\n", det)
	}
	return strings.TrimRight(b.String(), "\n")
}

func placeholderID(cur string) string {
	if cur == "" {
		return "required"
	}
	return fmt.Sprintf("empty = keep %s", maskMiddle(cur))
}

func placeholderSecret(cur string) string {
	if cur == "" {
		return "required"
	}
	return "empty = keep current secret"
}

func maskMiddle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 10 {
		return "(set)"
	}
	return s[:4] + "…" + s[len(s)-4:]
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func applyInputStyles(ti *textinput.Model) {
	x := theme.Current()
	st := textinput.DefaultDarkStyles()
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(x.Accent).Bold(true)
	st.Focused.Text = lipgloss.NewStyle().Foreground(x.Text)
	st.Focused.Placeholder = lipgloss.NewStyle().Foreground(x.Dim)
	st.Blurred.Text = lipgloss.NewStyle().Foreground(x.Muted)
	st.Cursor.Color = x.Accent2
	ti.SetStyles(st)
}

func p() theme.Palette { return theme.Current() }

func titleStyle() lipgloss.Style {
	x := p()
	return lipgloss.NewStyle().Bold(true).Foreground(x.Accent).
		BorderStyle(lipgloss.Border{Left: "│"}).BorderForeground(x.Border).PaddingLeft(1)
}

func subtitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Muted).Bold(true)
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Dim)
}

func hintStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Dim).Italic(true)
}

func errStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Rose).Bold(true)
}

func okStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Lime).Bold(true)
}

func cursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Accent2).Bold(true)
}

func selStyle() lipgloss.Style {
	x := p()
	return lipgloss.NewStyle().Foreground(x.Text).Background(x.MenuBG).Padding(0, 1).Bold(true)
}

func idleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Muted)
}

func frameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p().Border).Padding(0, 1)
}

func innerPad() lipgloss.Style {
	return lipgloss.NewStyle().Padding(1, 2)
}

func ynLine(selected bool, label string) string {
	if selected {
		return cursorStyle().Render("▸ ") + selStyle().Render(" "+label+" ")
	}
	return idleStyle().Render("    " + label)
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
