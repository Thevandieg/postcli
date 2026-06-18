package store

// Channel identifies where a queued post should be published.
type Channel string

const (
	ChannelX        Channel = "x"
	ChannelSubstack Channel = "substack"

	// Preview-only IDs (not persisted until integrations ship).
	ChannelMastodon Channel = "mastodon"
	ChannelBluesky  Channel = "bluesky"
	ChannelThreads  Channel = "threads"
)

// Label returns a short display name for TUIs.
func (c Channel) Label() string {
	switch c {
	case ChannelX:
		return "X (Twitter)"
	case ChannelSubstack:
		return "Substack"
	case ChannelMastodon:
		return "Mastodon"
	case ChannelBluesky:
		return "Bluesky"
	case ChannelThreads:
		return "Threads"
	default:
		return string(c)
	}
}
