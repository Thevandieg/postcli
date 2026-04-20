package post

import (
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// Dark-terminal palette (256-color).
var (
	colAccent   = lipgloss.Color("86")  // cyan
	colAccent2  = lipgloss.Color("213") // pink
	colViolet   = lipgloss.Color("141")
	colAmber    = lipgloss.Color("214")
	colLime     = lipgloss.Color("156")
	colRose     = lipgloss.Color("203")
	colText     = lipgloss.Color("252")
	colMuted    = lipgloss.Color("245")
	colDim      = lipgloss.Color("240")
	colPanel    = lipgloss.Color("236")
	colSubtle   = lipgloss.Color("238")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent).
			BorderStyle(lipgloss.Border{Left: "│"}).
			BorderForeground(colViolet).
			PaddingLeft(1)

	styleSubtitle = lipgloss.NewStyle().Foreground(colMuted).Bold(true)
	styleHint     = lipgloss.NewStyle().Foreground(colDim).Italic(true)
	styleErr      = lipgloss.NewStyle().Foreground(colRose).Bold(true)
	styleOk       = lipgloss.NewStyle().Foreground(colLime).Bold(true)
	styleBadge    = lipgloss.NewStyle().Foreground(colPanel).Background(colViolet).Padding(0, 1).Bold(true)

	styleMenuSel = lipgloss.NewStyle().
			Foreground(colText).
			Background(lipgloss.Color("57")).
			Padding(0, 1).
			Bold(true)

	styleMenuIdle = lipgloss.NewStyle().Foreground(colMuted)
	styleCursor     = lipgloss.NewStyle().Foreground(colAccent2).Bold(true)
	styleFrame      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colViolet).Padding(0, 1)
	styleFrameInner = lipgloss.NewStyle().Padding(1, 2)

	styleLabel = lipgloss.NewStyle().Foreground(colAmber).Bold(true)
	styleValue = lipgloss.NewStyle().Foreground(colText)
)

func applyBubbleStyles(ti *textinput.Model, ta *textarea.Model, fp *filepicker.Model, sp *spinner.Model) {
	tis := textinput.DefaultDarkStyles()
	tis.Focused.Prompt = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	tis.Focused.Text = lipgloss.NewStyle().Foreground(colText)
	tis.Focused.Placeholder = lipgloss.NewStyle().Foreground(colDim)
	tis.Blurred.Text = lipgloss.NewStyle().Foreground(colMuted)
	tis.Cursor.Color = colAccent2
	ti.SetStyles(tis)

	tas := textarea.DefaultDarkStyles()
	tas.Focused.Text = lipgloss.NewStyle().Foreground(colText)
	tas.Focused.Placeholder = lipgloss.NewStyle().Foreground(colDim)
	tas.Focused.Prompt = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	tas.Focused.LineNumber = lipgloss.NewStyle().Foreground(colDim)
	tas.Blurred.Text = lipgloss.NewStyle().Foreground(colMuted)
	tas.Cursor.Color = colAccent2
	ta.SetStyles(tas)

	fs := filepicker.DefaultStyles()
	fs.Cursor = lipgloss.NewStyle().Foreground(colAccent2).Bold(true)
	fs.Selected = lipgloss.NewStyle().Foreground(colText).Background(lipgloss.Color("57")).Bold(true)
	fs.Directory = lipgloss.NewStyle().Foreground(colAccent)
	fs.File = lipgloss.NewStyle().Foreground(colMuted)
	fs.Symlink = lipgloss.NewStyle().Foreground(colViolet)
	fs.Permission = lipgloss.NewStyle().Foreground(colDim)
	fs.FileSize = lipgloss.NewStyle().Foreground(colDim)
	fs.EmptyDirectory = lipgloss.NewStyle().Foreground(colDim).Italic(true)
	fp.Styles = fs

	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
}

func stepLabel(s step) string {
	switch s {
	case stepKind:
		return "Content type"
	case stepWhen:
		return "When to post"
	case stepSchedule:
		return "Schedule"
	case stepBody:
		return "Compose"
	case stepMedia:
		return "Media file"
	case stepConfirm:
		return "Confirm"
	default:
		return ""
	}
}

func stepIndex(s step) int {
	switch s {
	case stepKind:
		return 1
	case stepWhen:
		return 2
	case stepSchedule:
		return 3
	case stepBody:
		return 4
	case stepMedia:
		return 5
	case stepConfirm:
		return 6
	default:
		return 0
	}
}

// progressDots shows a compact step indicator (6 steps max).
func progressDots(s step) string {
	const n = 6
	idx := stepIndex(s)
	if idx < 1 {
		idx = 1
	}
	var parts []string
	for i := 1; i <= n; i++ {
		if i == idx {
			parts = append(parts, lipgloss.NewStyle().Foreground(colAccent2).Bold(true).Render("●"))
		} else if i < idx {
			parts = append(parts, lipgloss.NewStyle().Foreground(colLime).Render("✓"))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(colSubtle).Render("·"))
		}
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(colDim).Render("  "))
}

func headerBlock(step step) string {
	badge := styleBadge.Render(strings.ToUpper(stepLabel(step)))
	line := lipgloss.JoinHorizontal(lipgloss.Center, badge, "  ", progressDots(step))
	sub := styleSubtitle.Render(stepLabel(step))
	return lipgloss.JoinVertical(lipgloss.Left, styleTitle.Render("postx compose"), "", line, sub)
}

func hintLine(s string) string {
	return styleHint.Render(s)
}

func menuLine(selected bool, label string) string {
	if selected {
		return styleCursor.Render("▸ ") + styleMenuSel.Render(" "+label+" ")
	}
	return styleMenuIdle.Render("    ") + styleMenuIdle.Render(label)
}

func framedBlock(inner string) string {
	return styleFrame.Render(styleFrameInner.Render(inner))
}
