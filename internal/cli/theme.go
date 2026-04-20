package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"postcli/internal/config"
	"postcli/internal/theme"
)

func cmdTheme() *cobra.Command {
	root := &cobra.Command{
		Use:   "theme",
		Short: "List or set the TUI color theme",
		Long:  "Themes apply to postx post and postx status. Choice is saved under the config directory (see theme file path on set).",
	}
	root.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List available themes",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, id := range theme.IDs() {
				fmt.Printf("  %-8s — %s\n", id, theme.Summary(id))
			}
			return nil
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "set NAME",
		Short: "Persist a theme (violet, sky, orange, neutral, green; aliases: blue, pink, …)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := theme.Set(args[0]); err != nil {
				return err
			}
			fmt.Printf("postx: theme set to %q (%s)\n", theme.Current().ID, theme.Current().Desc)
			fmt.Println("postx: config:", config.ThemePath())
			return nil
		},
	})
	root.RunE = func(cmd *cobra.Command, args []string) error {
		_ = theme.Load()
		cur := theme.Current()
		fmt.Printf("Current theme: %s — %s\n\n", cur.ID, cur.Desc)
		fmt.Println("Available:")
		for _, id := range theme.IDs() {
			mark := " "
			if id == cur.ID {
				mark = "*"
			}
			fmt.Printf(" %s %-8s — %s\n", mark, id, theme.Summary(id))
		}
		fmt.Println("\nUse: postx theme set <name>   postx theme ls")
		return nil
	}
	return root
}
