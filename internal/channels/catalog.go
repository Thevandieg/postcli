package channels

import "postcli/internal/store"

// Entry describes one row in the channel picker (interactive post flow).
type Entry struct {
	ID         store.Channel
	Title      string
	Subtitle   string
	Selectable bool
}

// Catalog lists destinations. Only selectable entries are queued to the database.
func Catalog() []Entry {
	return []Entry{
		{
			ID:         store.ChannelX,
			Title:      "X (Twitter)",
			Subtitle:   "Available",
			Selectable: true,
		},
		{
			ID:         store.ChannelMastodon,
			Title:      "Mastodon",
			Subtitle:   "Preview · coming soon",
			Selectable: false,
		},
		{
			ID:         store.ChannelBluesky,
			Title:      "Bluesky",
			Subtitle:   "Preview · coming soon",
			Selectable: false,
		},
		{
			ID:         store.ChannelThreads,
			Title:      "Threads",
			Subtitle:   "Preview · coming soon",
			Selectable: false,
		},
	}
}
