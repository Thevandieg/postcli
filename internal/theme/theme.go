package theme

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"

	"postcli/internal/config"
)

// ID names persisted on disk and accepted by `postx theme set`.
const (
	Violet  = "violet"
	Sky     = "sky"
	Orange  = "orange"
	Neutral = "neutral"
	Green   = "green"
)

// Palette holds 256-color terminal roles shared by post + status TUIs.
type Palette struct {
	ID   string
	Desc string
	// Colors are ANSI 256 indices as used by lipgloss.Color("n").
	Accent, Accent2, Border, Amber, Lime, Rose, Text, Muted, Dim, Panel, Subtle, MenuBG color.Color
}

var (
	mu      sync.RWMutex
	current = presets[Violet]
)

// presets is keyed by canonical ID.
var presets = map[string]Palette{
	Violet: {
		ID: Violet, Desc: "Cyan & pink on violet (default)",
		Accent: lipgloss.Color("86"), Accent2: lipgloss.Color("213"), Border: lipgloss.Color("141"),
		Amber: lipgloss.Color("214"), Lime: lipgloss.Color("156"), Rose: lipgloss.Color("203"),
		Text: lipgloss.Color("252"), Muted: lipgloss.Color("245"), Dim: lipgloss.Color("240"),
		Panel: lipgloss.Color("236"), Subtle: lipgloss.Color("238"), MenuBG: lipgloss.Color("57"),
	},
	Sky: {
		ID: Sky, Desc: "Cool blues (sky)",
		Accent: lipgloss.Color("81"), Accent2: lipgloss.Color("117"), Border: lipgloss.Color("68"),
		Amber: lipgloss.Color("214"), Lime: lipgloss.Color("120"), Rose: lipgloss.Color("210"),
		Text: lipgloss.Color("255"), Muted: lipgloss.Color("247"), Dim: lipgloss.Color("244"),
		Panel: lipgloss.Color("24"), Subtle: lipgloss.Color("60"), MenuBG: lipgloss.Color("25"),
	},
	Orange: {
		ID: Orange, Desc: "Warm orange & amber",
		Accent: lipgloss.Color("208"), Accent2: lipgloss.Color("214"), Border: lipgloss.Color("130"),
		Amber: lipgloss.Color("220"), Lime: lipgloss.Color("178"), Rose: lipgloss.Color("167"),
		Text: lipgloss.Color("230"), Muted: lipgloss.Color("245"), Dim: lipgloss.Color("240"),
		Panel: lipgloss.Color("94"), Subtle: lipgloss.Color("130"), MenuBG: lipgloss.Color("94"),
	},
	Neutral: {
		ID: Neutral, Desc: "Low-contrast gray",
		Accent: lipgloss.Color("248"), Accent2: lipgloss.Color("250"), Border: lipgloss.Color("245"),
		Amber: lipgloss.Color("246"), Lime: lipgloss.Color("249"), Rose: lipgloss.Color("174"),
		Text: lipgloss.Color("255"), Muted: lipgloss.Color("249"), Dim: lipgloss.Color("246"),
		Panel: lipgloss.Color("238"), Subtle: lipgloss.Color("240"), MenuBG: lipgloss.Color("236"),
	},
	Green: {
		ID: Green, Desc: "Forest & mint greens",
		Accent: lipgloss.Color("84"), Accent2: lipgloss.Color("118"), Border: lipgloss.Color("65"),
		Amber: lipgloss.Color("178"), Lime: lipgloss.Color("154"), Rose: lipgloss.Color("131"),
		Text: lipgloss.Color("194"), Muted: lipgloss.Color("247"), Dim: lipgloss.Color("242"),
		Panel: lipgloss.Color("22"), Subtle: lipgloss.Color("28"), MenuBG: lipgloss.Color("22"),
	},
}

// IDs returns sorted theme ids for CLI help.
func IDs() []string {
	return []string{Violet, Sky, Orange, Neutral, Green}
}

// Summary returns id + one-line description for CLI listing.
func Summary(id string) string {
	p := presets[strings.ToLower(strings.TrimSpace(id))]
	if p.Desc == "" {
		return id
	}
	return p.Desc
}

// CanonicalID maps user input to a preset id, or returns an error.
func CanonicalID(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case Violet, "pink", "default", "":
		return Violet, nil
	case Sky, "bluesky", "blue":
		return Sky, nil
	case Orange, "warm":
		return Orange, nil
	case Neutral, "gray", "grey":
		return Neutral, nil
	case Green, "mint":
		return Green, nil
	default:
		return "", fmt.Errorf("unknown theme %q; choose one of: %s", strings.TrimSpace(raw), strings.Join(IDs(), ", "))
	}
}

// ByID resolves a canonical id (violet, sky, …) to a palette; invalid falls back to violet.
func ByID(id string) Palette {
	if p, ok := presets[strings.ToLower(strings.TrimSpace(id))]; ok {
		return p
	}
	return presets[Violet]
}

// Current returns the in-memory palette (after Load / Set).
func Current() Palette {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Load reads ~/.config/postcli/theme (or XDG). Missing file keeps violet.
func Load() error {
	path := config.ThemePath()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			mu.Lock()
			current = presets[Violet]
			mu.Unlock()
			return nil
		}
		return err
	}
	mu.Lock()
	canon, err := CanonicalID(strings.TrimSpace(string(b)))
	if err != nil {
		current = presets[Violet]
	} else {
		current = presets[canon]
	}
	mu.Unlock()
	return nil
}

// Set validates id, writes theme file, and updates Current.
func Set(raw string) error {
	if err := config.EnsureDir(); err != nil {
		return err
	}
	canon, err := CanonicalID(raw)
	if err != nil {
		return err
	}
	p := presets[canon]
	path := config.ThemePath()
	if err := os.WriteFile(path, []byte(p.ID+"\n"), 0o600); err != nil {
		return err
	}
	mu.Lock()
	current = p
	mu.Unlock()
	return nil
}
