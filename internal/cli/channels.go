package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"postcli/internal/channels"
	"postcli/internal/config"
	"postcli/internal/tui/channelsui"
	"postcli/internal/xapi"
)

const (
	envMarkerStart = "# >>> postx managed env >>>"
	envMarkerEnd   = "# <<< postx managed env <<<"
)

func cmdChannels() *cobra.Command {
	c := &cobra.Command{
		Use:   "channels",
		Short: "Browse and configure publishing channels (interactive)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChannelsInteractive(context.Background())
		},
	}
	c.AddCommand(cmdChannelsConfigure())
	return c
}

func cmdChannelsConfigure() *cobra.Command {
	var timeout time.Duration
	c := &cobra.Command{
		Use:   "configure [channel]",
		Short: "Configure one channel (credentials + setup)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ch := "x"
			if len(args) > 0 {
				ch = strings.ToLower(strings.TrimSpace(args[0]))
			} else {
				var err error
				ch, err = promptChannelSelection()
				if err != nil {
					return err
				}
			}
			switch ch {
			case "x", "twitter":
				return configureXChannel(context.Background(), timeout)
			default:
				return fmt.Errorf("channel %q not supported yet", ch)
			}
		},
	}
	c.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "max wait for browser login callback")
	return c
}

func promptChannelSelection() (string, error) {
	fmt.Println("Select a channel to configure:")
	fmt.Println("  1) X (Twitter)")
	fmt.Println("  2) Mastodon (coming soon)")
	fmt.Println("  3) Bluesky (coming soon)")
	fmt.Println("  4) Threads (coming soon)")
	fmt.Print("Enter choice [1-4]: ")
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		return "", err
	}
	switch strings.TrimSpace(line) {
	case "1", "x", "twitter":
		return "x", nil
	case "2":
		return "", fmt.Errorf("Mastodon integration is preview-only for now")
	case "3":
		return "", fmt.Errorf("Bluesky integration is preview-only for now")
	case "4":
		return "", fmt.Errorf("Threads integration is preview-only for now")
	default:
		return "", fmt.Errorf("invalid selection")
	}
}

func runChannelsInteractive(ctx context.Context) error {
	const loginTimeout = 5 * time.Minute
	for {
		st, err := openStore(ctx)
		if err != nil {
			return err
		}
		stats := channels.Statuses(ctx, st, channels.XConfig{
			ClientID:     ClientID(),
			ClientSecret: ClientSecret(),
		})
		st.Close()

		act, err := channelsui.Run(stats)
		if err != nil {
			return err
		}
		switch act {
		case channelsui.ActionQuit:
			return nil
		case channelsui.ActionConfigureX:
			if err := configureXChannel(ctx, loginTimeout); err != nil {
				fmt.Fprintf(os.Stderr, "postx: configure: %v\n", err)
			}
			continue
		default:
			return nil
		}
	}
}

func configureXChannel(ctx context.Context, timeout time.Duration) error {
	in := bufio.NewReader(os.Stdin)
	fmt.Print("X OAuth Client ID: ")
	clientID, err := in.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Print("X OAuth Client Secret: ")
	clientSecret, err := in.ReadString('\n')
	if err != nil {
		return err
	}
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("both client ID and client secret are required")
	}
	if err := config.EnsureDir(); err != nil {
		return err
	}
	envMap, err := config.LoadEnvMap()
	if err != nil {
		return err
	}
	envMap["POSTX_CLIENT_ID"] = clientID
	envMap["POSTX_CLIENT_SECRET"] = clientSecret
	if envMap["POSTX_REDIRECT_URI"] == "" {
		envMap["POSTX_REDIRECT_URI"] = RedirectURI()
	}
	if err := config.SaveEnvMap(envMap); err != nil {
		return err
	}
	if err := syncShellProfile(envMap); err != nil {
		fmt.Fprintf(os.Stderr, "postx: warning: shell profile update failed: %v\n", err)
	}
	_ = os.Setenv("POSTX_CLIENT_ID", clientID)
	_ = os.Setenv("POSTX_CLIENT_SECRET", clientSecret)
	_ = os.Setenv("POSTX_REDIRECT_URI", envMap["POSTX_REDIRECT_URI"])

	st, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer st.Close()

	loginCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := doXLogin(loginCtx, st, RedirectURI(), timeout); err != nil {
		return err
	}
	fmt.Println("postx: X channel configured and ready.")
	return nil
}

func syncShellProfile(envMap map[string]string) error {
	target, fishStyle, err := detectShellProfile()
	if err != nil {
		return err
	}
	if target == "" {
		return nil
	}
	content := buildEnvBlock(envMap, fishStyle)
	b, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existing := string(b)
	updated := replaceManagedBlock(existing, content)
	if updated == existing {
		return nil
	}
	return os.WriteFile(target, []byte(updated), 0o600)
}

func detectShellProfile() (path string, fishStyle bool, err error) {
	shell := filepath.Base(strings.TrimSpace(os.Getenv("SHELL")))
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc"), false, nil
	case "bash":
		return filepath.Join(home, ".bashrc"), false, nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), true, nil
	default:
		return "", false, nil
	}
}

func buildEnvBlock(envMap map[string]string, fishStyle bool) string {
	var b strings.Builder
	b.WriteString(envMarkerStart + "\n")
	keys := []string{"POSTX_CLIENT_ID", "POSTX_CLIENT_SECRET", "POSTX_REDIRECT_URI"}
	for _, k := range keys {
		v := strings.TrimSpace(envMap[k])
		if v == "" {
			continue
		}
		if fishStyle {
			fmt.Fprintf(&b, "set -gx %s %q\n", k, v)
		} else {
			fmt.Fprintf(&b, "export %s=%q\n", k, v)
		}
	}
	b.WriteString(envMarkerEnd + "\n")
	return b.String()
}

func replaceManagedBlock(existing, content string) string {
	start := strings.Index(existing, envMarkerStart)
	end := strings.Index(existing, envMarkerEnd)
	if start >= 0 && end > start {
		end += len(envMarkerEnd)
		return existing[:start] + content + strings.TrimPrefix(existing[end:], "\n")
	}
	if strings.TrimSpace(existing) == "" {
		return content
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + "\n" + content
}

func makeOAuthConfig(redirect string) xapi.OAuthConfig {
	return xapi.OAuthConfig{
		ClientID:     ClientID(),
		ClientSecret: ClientSecret(),
		RedirectURI:  redirect,
	}
}
