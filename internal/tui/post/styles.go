package post

import (
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"postcli/internal/theme"
)

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
	return lipgloss.NewStyle().Foreground(p().Muted).Bold(true)
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

func badgeStyle() lipgloss.Style {
	x := p()
	return lipgloss.NewStyle().Foreground(x.Panel).Background(x.Border).Padding(0, 1).Bold(true)
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

func cursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Accent2).Bold(true)
}

func frameStyle() lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p().Border).Padding(0, 1)
}

var frameInnerStyle = lipgloss.NewStyle().Padding(1, 2)

func labelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Amber).Bold(true)
}

func valueStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Text)
}

func applyBubbleStyles(ti *textinput.Model, ta *textarea.Model, fp *filepicker.Model, sp *spinner.Model) {
	x := p()
	tis := textinput.DefaultDarkStyles()
	tis.Focused.Prompt = lipgloss.NewStyle().Foreground(x.Accent).Bold(true)
	tis.Focused.Text = lipgloss.NewStyle().Foreground(x.Text)
	tis.Focused.Placeholder = lipgloss.NewStyle().Foreground(x.Dim)
	tis.Blurred.Text = lipgloss.NewStyle().Foreground(x.Muted)
	tis.Cursor.Color = x.Accent2
	ti.SetStyles(tis)

	tas := textarea.DefaultDarkStyles()
	tas.Focused.Text = lipgloss.NewStyle().Foreground(x.Text)
	tas.Focused.Placeholder = lipgloss.NewStyle().Foreground(x.Dim)
	tas.Focused.Prompt = lipgloss.NewStyle().Foreground(x.Accent).Bold(true)
	tas.Focused.LineNumber = lipgloss.NewStyle().Foreground(x.Dim)
	tas.Blurred.Text = lipgloss.NewStyle().Foreground(x.Muted)
	tas.Cursor.Color = x.Accent2
	ta.SetStyles(tas)

	fs := filepicker.DefaultStyles()
	fs.Cursor = lipgloss.NewStyle().Foreground(x.Accent2).Bold(true)
	fs.Selected = lipgloss.NewStyle().Foreground(x.Text).Background(x.MenuBG).Bold(true)
	fs.Directory = lipgloss.NewStyle().Foreground(x.Accent)
	fs.File = lipgloss.NewStyle().Foreground(x.Muted)
	fs.Symlink = lipgloss.NewStyle().Foreground(x.Border)
	fs.Permission = lipgloss.NewStyle().Foreground(x.Dim)
	fs.FileSize = lipgloss.NewStyle().Foreground(x.Dim)
	fs.EmptyDirectory = lipgloss.NewStyle().Foreground(x.Dim).Italic(true)
	fp.Styles = fs

	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(x.Accent).Bold(true)
}

func stepLabel(s step) string {
	switch s {
	case stepKind:
		return "Content type"
	case stepBody:
		return "Compose"
	case stepMedia:
		return "Media file"
	case stepChannels:
		return "Channels"
	case stepWhen:
		return "When to post"
	case stepSchedule:
		return "Schedule"
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
	case stepBody:
		return 2
	case stepMedia:
		return 3
	case stepChannels:
		return 4
	case stepWhen:
		return 5
	case stepSchedule:
		return 6
	case stepConfirm:
		return 7
	default:
		return 0
	}
}

func progressDots(s step) string {
	x := p()
	const n = 7
	idx := stepIndex(s)
	if idx < 1 {
		idx = 1
	}
	var parts []string
	for i := 1; i <= n; i++ {
		if i == idx {
			parts = append(parts, lipgloss.NewStyle().Foreground(x.Accent2).Bold(true).Render("●"))
		} else if i < idx {
			parts = append(parts, lipgloss.NewStyle().Foreground(x.Lime).Render("✓"))
		} else {
			parts = append(parts, lipgloss.NewStyle().Foreground(x.Subtle).Render("·"))
		}
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(x.Dim).Render("  "))
}

func headerBlock(s step) string {
	badge := badgeStyle().Render(strings.ToUpper(stepLabel(s)))
	line := lipgloss.JoinHorizontal(lipgloss.Center, badge, "  ", progressDots(s))
	sub := subtitleStyle().Render(stepLabel(s))
	return lipgloss.JoinVertical(lipgloss.Left, titleStyle().Render("postx compose"), "", line, sub)
}

func hintLine(s string) string {
	return hintStyle().Render(s)
}

func menuLine(selected bool, label string) string {
	if selected {
		return cursorStyle().Render("▸ ") + menuSelStyle().Render(" "+label+" ")
	}
	return menuIdleStyle().Render("    ") + menuIdleStyle().Render(label)
}

func framedBlock(inner string) string {
	return frameStyle().Render(frameInnerStyle.Render(inner))
}

func dimTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Dim)
}

func mutedItalicStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(p().Muted).Italic(true)
}
